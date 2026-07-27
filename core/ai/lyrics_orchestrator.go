package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// lyricsTaskTimeout caps how long a single background fetch may run.
const lyricsTaskTimeout = 5 * time.Minute

// orchestrator coordinates per-user background lyrics fetches. It is embedded in
// the Service so it can reuse the cached provider and the DataStore.
type orchestrator struct {
	mu    sync.Mutex
	inflight map[string]struct{} // mediaFileIds currently being processed
}

func newOrchestrator() *orchestrator {
	return &orchestrator{inflight: make(map[string]struct{})}
}

// statusPropKey is the UserProps key under which a media file's task status is
// stored: ai_lyrics_status:<mediaFileId>.
func statusPropKey(mediaFileId string) string {
	return "ai_lyrics_status:" + mediaFileId
}

// FetchLyrics queues a background fetch for the given media file. It is
// idempotent: a second call while the first is running returns the current
// status without starting a duplicate goroutine.
func (s *Service) FetchLyrics(ctx context.Context, userId, mediaFileId string) error {
	mf, err := s.ds.MediaFile(ctx).Get(mediaFileId)
	if err != nil {
		return fmt.Errorf("loading media file: %w", err)
	}

	// Mark as queued immediately so the UI reflects the new state.
	s.saveStatus(ctx, userId, mediaFileId, LyricsTaskStatus{
		Status: StatusQueued, Step: StepLyrics, UpdatedAt: time.Now().Unix(),
	})

	s.orch.mu.Lock()
	if _, ok := s.orch.inflight[mediaFileId]; ok {
		s.orch.mu.Unlock()
		return nil // already running
	}
	s.orch.inflight[mediaFileId] = struct{}{}
	s.orch.mu.Unlock()

	go s.runLyricsPipeline(context.Background(), userId, mf)
	return nil
}

// GetLyricsStatus returns the persisted status for a media file (or a zero
// status with Status="" if none has ever been recorded).
func (s *Service) GetLyricsStatus(ctx context.Context, userId, mediaFileId string) (LyricsTaskStatus, error) {
	raw, err := s.ds.UserProps(ctx).DefaultGet(userId, statusPropKey(mediaFileId), "")
	if err != nil {
		return LyricsTaskStatus{}, err
	}
	if raw == "" {
		return LyricsTaskStatus{}, nil
	}
	var st LyricsTaskStatus
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return LyricsTaskStatus{}, err
	}
	return st, nil
}

func (s *Service) saveStatus(ctx context.Context, userId, mediaFileId string, st LyricsTaskStatus) {
	st.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(st)
	if err != nil {
		log.Error(ctx, "Marshalling lyrics status", "error", err)
		return
	}
	if err := s.ds.UserProps(ctx).Put(userId, statusPropKey(mediaFileId), string(data)); err != nil {
		log.Error(ctx, "Persisting lyrics status", "error", err)
	}
}

// runLyricsPipeline is the background worker. It resolves the provider for the
// user, fetches the original (LRCLIB first, then Gemini with web search),
// translates to Russian, and writes two sidecar files.
func (s *Service) runLyricsPipeline(ctx context.Context, userId string, mf *model.MediaFile) {
	mediaFileId := mf.ID
	defer func() {
		s.orch.mu.Lock()
		delete(s.orch.inflight, mediaFileId)
		s.orch.mu.Unlock()
	}()

	taskCtx, cancel := context.WithTimeout(ctx, lyricsTaskTimeout)
	defer cancel()

	fail := func(step, msg string) {
		log.Error(taskCtx, "AI lyrics pipeline failed", "track", mf.Title, "step", step, "error", msg)
		s.saveStatus(ctx, userId, mediaFileId, LyricsTaskStatus{
			Status: StatusError, Step: step, Error: msg,
		})
	}

	// Provider must be ready; if not, surface a clear error.
	provider, err := s.getProvider(ctx, userId)
	if err != nil {
		fail(StepLyrics, "AI provider is not configured: "+err.Error())
		return
	}

	store := newLyricsStore()

	// --- Step 1: original synced lyrics via LRCLIB (exact then fuzzy search) ---
	s.saveStatus(ctx, userId, mediaFileId, LyricsTaskStatus{
		Status: StatusRunning, Step: StepLyrics, UpdatedAt: time.Now().Unix(),
	})

	original, lang, err := s.fetchOriginal(taskCtx, provider, mf)
	if err != nil {
		fail(StepLyrics, err.Error())
		return
	}
	if original == "" {
		fail(StepLyrics, "lyrics not found in LRCLIB")
		return
	}

	// --- Step 2: Russian translation, reusing the original's timing ---
	// Skip translation when the original is already Russian (or looks Russian)
	// — translating it would be wrong and waste Gemini quota.
	translation := ""
	if !isRussianText(stripLRCTimestamps(original)) {
		s.saveStatus(ctx, userId, mediaFileId, LyricsTaskStatus{
			Status: StatusRunning, Step: StepTranslation, LyricsHit: true,
		})
		translation, err = s.translateLyrics(taskCtx, provider, mf, original)
		if err != nil {
			fail(StepTranslation, err.Error())
			return
		}
	} else {
		log.Info(ctx, "Original lyrics are Russian, skipping translation", "track", mf.Title)
	}

	// --- Step 3: persist sidecar files ---
	if err := store.writeSidecar(mf, ".lrc", ensureLangHeader(original, lang)); err != nil {
		fail(StepLyrics, err.Error())
		return
	}
	if translation != "" {
		ru := alignToOriginalTiming(original, translation)
		if err := store.writeSidecar(mf, ".ru.lrc", ensureLangHeader(ru, "ru")); err != nil {
			fail(StepTranslation, err.Error())
			return
		}
	}

	// --- Step 4: write the LyricList JSON into media_file.lyrics so the web
	// player (which reads the DB column, not sidecars) shows it immediately. ---
	if err := s.persistLyricsToDB(ctx, mf, original, translation); err != nil {
		log.Warn(ctx, "Failed to update media_file.lyrics (sidecars are still written)", "error", err)
	}

	s.saveStatus(ctx, userId, mediaFileId, LyricsTaskStatus{
		Status: StatusDone, LyricsHit: true,
	})
	log.Info(ctx, "AI lyrics pipeline completed", "track", mf.Title, "artist", mf.Artist)
}

// persistLyricsToDB builds the JSON LyricList the web player expects
// ([{synced, line:[{start,value}]}]) from the original LRC and the optional
// Russian translation, and stores it in the media_file.lyrics column.
func (s *Service) persistLyricsToDB(ctx context.Context, mf *model.MediaFile, originalLRC, translation string) error {
	list := model.LyricList{}

	// Original (main). ParseLyrics needs a suffix to pick the format; ".lrc"
	// routes to the LRC parser which sets Synced=true and Line[].Start.
	if main, err := model.ParseLyrics(".lrc", "xxx", []byte(originalLRC)); err == nil {
		for i := range main {
			main[i].Kind = model.LyricKindMain
		}
		list = append(list, main...)
	}
	// Russian translation, if any. Re-align it to the original's timing and
	// mark it as a synced translation so the player shows both tracks.
	if translation != "" {
		ru := alignToOriginalTiming(originalLRC, translation)
		if ruList, err := model.ParseLyrics(".lrc", "ru", []byte(ru)); err == nil {
			for i := range ruList {
				ruList[i].Kind = model.LyricKindTranslation
				ruList[i].Lang = "ru"
			}
			list = append(list, ruList...)
		}
	}

	if len(list) == 0 {
		return nil
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.ds.MediaFile(ctx).UpdateLyrics(mf.ID, string(data))
}

// fetchOriginal resolves the synced LRC for the track from LRCLIB only. The
// client tries an exact /api/get first, then a fuzzy /api/search, so most
// tracks are covered without any AI call. Returns ("", "", nil) when nothing is
// found — the caller surfaces a "lyrics not found" status. (Recall via Gemini
// web search was intentionally removed: it was expensive on quota and less
// reliable than LRCLIB's curated timings.)
func (s *Service) fetchOriginal(ctx context.Context, provider LLMProvider, mf *model.MediaFile) (string, string, error) {
	durationSec := 0.0
	if mf.Duration > 0 {
		durationSec = float64(mf.Duration)
	}

	if c := s.lrclib; c != nil {
		if t, err := c.getSynced(ctx, mf.Artist, mf.Title, mf.Album, durationSec); err != nil {
			log.Warn(ctx, "LRCLIB lookup failed", "track", mf.Title, "error", err)
		} else if t != nil {
			log.Info(ctx, "Lyrics found via LRCLIB", "track", mf.Title)
			return t.SyncedLyric, t.Lang, nil
		}
	}
	log.Info(ctx, "Lyrics not found in LRCLIB", "track", mf.Title, "artist", mf.Artist)
	return "", "", nil
}

// translateLyrics asks the provider to translate the original LRC to Russian,
// preserving line structure so timings can be re-aligned afterwards.
func (s *Service) translateLyrics(ctx context.Context, provider LLMProvider, mf *model.MediaFile, originalLRC string) (string, error) {
	// Strip [mm:ss.xx] prefixes so the model translates pure text.
	plain := stripLRCTimestamps(originalLRC)

	resp, err := provider.Translate(ctx, &TranslateRequest{
		Title:  mf.Title,
		Artist: mf.Artist,
		Lyrics: plain,
		ToLang: "ru",
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Translation), nil
}

// stripLRCTimestamps removes leading [mm:ss.xx] markers from every line,
// returning the plain lyric text.
func stripLRCTimestamps(lrc string) string {
	var b strings.Builder
	for _, line := range strings.Split(lrc, "\n") {
		l := line
		for strings.HasPrefix(strings.TrimSpace(l), "[") {
			// drop a single leading tag like [00:12.34] or [lang:xx]
			trim := strings.TrimSpace(l)
			closeIdx := strings.Index(trim, "]")
			if closeIdx < 0 {
				break
			}
			l = trim[closeIdx+1:]
		}
		b.WriteString(strings.TrimSpace(l))
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

// alignToOriginalTiming copies the timestamp prefix from each original LRC line
// onto the corresponding translated line. If the translation has fewer or more
// lines, unmatched lines are emitted without a timestamp.
func alignToOriginalTiming(originalLRC, translation string) string {
	origLines := strings.Split(originalLRC, "\n")
	transLines := strings.Split(strings.TrimSpace(translation), "\n")

	type prefix struct{ ts string }
	// Extract timestamp prefixes from the original (skip non-timestamp lines).
	var ts []string
	for _, l := range origLines {
		tl := strings.TrimSpace(l)
		if !strings.HasPrefix(tl, "[") {
			ts = append(ts, "")
			continue
		}
		closeIdx := strings.Index(tl, "]")
		if closeIdx < 0 {
			ts = append(ts, "")
			continue
		}
		tag := tl[:closeIdx+1]
		if isTimestamp(tag) {
			ts = append(ts, tag)
		} else {
			// Header tag like [lang:] — keep it once at the top instead of per line.
			ts = append(ts, "")
		}
	}

	var b strings.Builder
	for i, line := range transLines {
		prefix := ""
		if i < len(ts) {
			prefix = ts[i]
		}
		text := strings.TrimSpace(stripLeadingTags(line))
		if text == "" {
			continue
		}
		if prefix != "" {
			b.WriteString(prefix + " ")
		}
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

// isTimestamp reports whether an LRC tag like "[00:12.34]" is a timing marker
// (as opposed to an ID tag like [lang:xx] or [ar:...]).
func isTimestamp(tag string) bool {
	inner := strings.TrimSuffix(strings.TrimPrefix(tag, "["), "]")
	if inner == "" {
		return false
	}
	// A timing tag starts with a digit, e.g. "00:12.34".
	return inner[0] >= '0' && inner[0] <= '9'
}

// stripLeadingTags removes any leading [..] tags from a line.
func stripLeadingTags(line string) string {
	for strings.HasPrefix(strings.TrimSpace(line), "[") {
		trim := strings.TrimSpace(line)
		closeIdx := strings.Index(trim, "]")
		if closeIdx < 0 {
			break
		}
		line = trim[closeIdx+1:]
	}
	return line
}

// isRussianText reports whether a body of text is predominantly Cyrillic, i.e.
// likely already Russian and therefore not worth translating. We count Cyrillic
// letters vs Latin letters among the alphabetic characters and treat the text as
// Russian when Cyrillic is the majority (and there are enough letters to tell).
func isRussianText(s string) bool {
	var cyrillic, latin int
	for _, r := range s {
		switch {
		case r >= 0x0400 && r <= 0x04FF: // Cyrillic block
			cyrillic++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		}
	}
	total := cyrillic + latin
	if total < 20 {
		// Too little text to judge reliably — don't translate to be safe.
		return false
	}
	return cyrillic*2 > latin
}
