package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	squirrel "github.com/Masterminds/squirrel"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// aiConfigKey is the UserProps key under which the AI config (as JSON) is stored per-user.
const aiConfigKey = "ai_config"

// Z.ai (Zhipu GLM) defaults. The Coding Plan endpoint is separate from the
// standard public API; both are OpenAI-compatible.
const (
	defaultZAIEndpoint = "https://api.z.ai/api/coding/paas/v4"
	defaultZAIModel    = "glm-5-turbo"
)

// maskPlaceholder is the masked value returned to the client in place of the
// API key. When the client sends it back, the previously stored key is kept.
const maskPlaceholder = "********"

// Service manages AI providers and per-user configuration.
// Configuration is persisted in the user properties table (UserProps),
// so it survives server restarts and is specific to each user.
type Service struct {
	ds        model.DataStore
	mu        sync.Mutex
	providers map[string]LLMProvider // cached providers keyed by userId
	orch      *orchestrator
	lrclib    *lrclibClient
}

// NewService creates a new AI service backed by the given DataStore.
func NewService(ds model.DataStore) *Service {
	return &Service{
		ds:        ds,
		providers: make(map[string]LLMProvider),
		orch:      newOrchestrator(),
		lrclib:    newLrclibClient(),
	}
}

// GetConfig returns the stored configuration for the user.
// Returns a zero-value Config (with empty Provider) if none is configured.
func (s *Service) GetConfig(ctx context.Context, userId string) (Config, error) {
	raw, err := s.ds.UserProps(ctx).DefaultGet(userId, aiConfigKey, "")
	if err != nil {
		return Config{}, err
	}
	if raw == "" {
		return Config{DefaultLang: "en"}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{DefaultLang: "en"}, err
	}
	return cfg, nil
}

// UpdateConfig persists the configuration for the user and refreshes the cached provider.
// The API key is stored in plaintext in the DB (same as other Navidrome secrets);
// it is never returned to the client by GetConfig (masked as ********).
// If the client sends "********" as the API key, the previously stored key is kept.
func (s *Service) UpdateConfig(ctx context.Context, userId string, config Config) error {
	// If the key is the mask placeholder, keep the previously stored key.
	if config.APIKey == maskPlaceholder {
		if old, err := s.GetConfig(ctx, userId); err == nil {
			config.APIKey = old.APIKey
		}
	}

	// Validate the provider can be instantiated before saving.
	if _, err := s.createProvider(config); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := s.ds.UserProps(ctx).Put(userId, aiConfigKey, string(data)); err != nil {
		return err
	}

	// Invalidate the cached provider for this user.
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.providers, userId)
	log.Info(ctx, "AI configuration updated", "user", userId, "provider", config.Provider)
	return nil
}

// getProvider returns the provider for the user, building and caching it if needed.
func (s *Service) getProvider(ctx context.Context, userId string) (LLMProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.providers[userId]; ok && p != nil {
		return p, nil
	}

	cfg, err := s.GetConfig(ctx, userId)
	if err != nil {
		return nil, err
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	p, err := s.createProvider(cfg)
	if err != nil {
		return nil, err
	}
	s.providers[userId] = p
	return p, nil
}

// TestConfig validates the given configuration by instantiating the provider
// and sending a minimal probe request. It accepts a Config directly so the UI
// can test a key/endpoint before saving it. If config is the zero value
// (Provider empty), the user's currently stored config is tested instead.
func (s *Service) TestConfig(ctx context.Context, userId string, config Config) error {
	// Load the stored config once; we merge the incoming values on top of it so
	// the caller can test a partial form (e.g. just a new key) without resending
	// every field.
	stored, err := s.GetConfig(ctx, userId)
	if err != nil {
		return err
	}

	// Resolve which provider to test: the one in the request, or the stored one.
	if config.Provider == "" {
		config.Provider = stored.Provider
	}
	if config.Provider == "" {
		return fmt.Errorf("no AI provider configured")
	}

	// API key: the mask placeholder (or empty) means "use the stored key".
	if config.APIKey == "" || config.APIKey == maskPlaceholder {
		config.APIKey = stored.APIKey
	}
	// Endpoint/model default to the stored values when blank.
	if config.APIEndpoint == "" {
		config.APIEndpoint = stored.APIEndpoint
	}
	if config.Model == "" {
		config.Model = stored.Model
	}

	p, err := s.createProvider(config)
	if err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}
	return p.TestConnection(ctx)
}

// Translate translates text using the user's configured provider. When the
// request carries a MediaFileID, the translation is also persisted to a sidecar
// file (<base>.ai.translate.<lang>.txt) so it can be shown immediately next
// time the drawer is opened.
func (s *Service) Translate(ctx context.Context, userId string, req *TranslateRequest) (*TranslateResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	resp, err := p.Translate(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.MediaFileID != "" && strings.TrimSpace(resp.Translation) != "" {
		lang := req.ToLang
		if lang == "" {
			lang = "default"
		}
		s.persistResult(ctx, req.MediaFileID, ".ai.translate."+lang+".txt", resp.Translation)
	}
	return resp, nil
}

// Analyze analyzes a track using the user's configured provider. When the
// request carries a MediaFileID, the analysis is persisted to a sidecar
// (<base>.ai.analyze.txt).
func (s *Service) Analyze(ctx context.Context, userId string, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	resp, err := p.Analyze(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.MediaFileID != "" && strings.TrimSpace(resp.Text) != "" {
		s.persistResult(ctx, req.MediaFileID, ".ai.analyze.txt", resp.Text)
	}
	return resp, nil
}

// Decode analyzes the meaning of a song using the user's configured provider.
// When the request carries a MediaFileID, the decode is persisted to a sidecar
// (<base>.ai.decode.md) as Markdown.
func (s *Service) Decode(ctx context.Context, userId string, req *DecodeRequest) (*DecodeResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	resp, err := p.Decode(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.MediaFileID != "" && strings.TrimSpace(resp.Text) != "" {
		s.persistResult(ctx, req.MediaFileID, ".ai.decode.md", resp.Text)
	}
	return resp, nil
}

// persistResult writes an AI result (translate/decode/analyze) to a sidecar
// file next to the media file. The suffix is the COMPLETE suffix (leading "."
// and extension), e.g. ".ai.decode.md" or ".ai.translate.ru.txt".
func (s *Service) persistResult(ctx context.Context, mediaFileId, suffix, body string) {
	mf, err := s.ds.MediaFile(ctx).Get(mediaFileId)
	if err != nil {
		log.Warn(ctx, "AI persist: media file not found", "mediaFileId", mediaFileId, "error", err)
		return
	}
	store := newLyricsStore()
	if err := store.writeSidecar(mf, suffix, body); err != nil {
		log.Warn(ctx, "AI persist: failed to write sidecar", "mediaFileId", mediaFileId, "error", err)
	}
}

// GetStoredResult reads a previously persisted AI sidecar for a media file.
// The suffix is the COMPLETE suffix (leading "." and extension), e.g.
// ".ai.decode.md". Returns ("", false, nil) when no sidecar exists.
func (s *Service) GetStoredResult(ctx context.Context, mediaFileId, suffix string) (string, bool, error) {
	mf, err := s.ds.MediaFile(ctx).Get(mediaFileId)
	if err != nil {
		return "", false, err
	}
	store := newLyricsStore()
	body, ok := store.readSidecar(mf, suffix)
	return body, ok, nil
}

// IsConfigured returns whether the user has a provider configured.
// Local providers (ollama, localai) don't require an API key.
func (s *Service) IsConfigured(ctx context.Context, userId string) bool {
	cfg, err := s.GetConfig(ctx, userId)
	if err != nil {
		return false
	}
	if cfg.Provider == "" {
		return false
	}
	switch cfg.Provider {
	case "ollama", "localai":
		return cfg.APIEndpoint != ""
	default:
		return cfg.APIKey != ""
	}
}

// createProvider creates a provider instance based on config.
func (s *Service) createProvider(config Config) (LLMProvider, error) {
	switch config.Provider {
	case "gemini":
		// Gemini uses its own native API (generativelanguage.googleapis.com)
		// which is the only endpoint that accepts the new "AQ."-prefixed keys.
		return NewGeminiProvider(config.APIKey, config.APIEndpoint, config.Model)
	case "ollama":
		// If the endpoint points at Ollama's OpenAI-compatible API (ends in /v1),
		// use the OpenAI-compatible client which talks to /chat/completions.
		// Otherwise fall back to Ollama's native /api/generate endpoint.
		if isOpenAICompatible(config.APIEndpoint) {
			return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
		}
		return NewOllamaProvider(config.APIEndpoint, config.Model)
	case "zai":
		// Z.ai (Zhipu) — OpenAI-compatible. The Coding Plan uses a dedicated
		// endpoint at https://api.z.ai/api/coding/paas/v4 (the standard API is
		// .../api/paas/v4). Both accept the standard /chat/completions path and
		// Authorization: Bearer header, so the OpenAI client works unchanged.
		endpoint := config.APIEndpoint
		if endpoint == "" {
			endpoint = defaultZAIEndpoint
		}
		model := config.Model
		if model == "" {
			model = defaultZAIModel
		}
		return NewOpenAIProvider(config.APIKey, endpoint, model)
	case "openai", "localai", "openrouter", "anthropic":
		// All of these expose an OpenAI-compatible /v1/chat/completions endpoint.
		return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// isOpenAICompatible reports whether the endpoint exposes an OpenAI-compatible
// API (i.e. ends in /v1, in which case requests go to /v1/chat/completions).
func isOpenAICompatible(endpoint string) bool {
	return strings.HasSuffix(strings.TrimRight(endpoint, "/"), "/v1")
}

// GetSupportedProviders returns a list of supported provider names.
func GetSupportedProviders() []string {
	return []string{"gemini", "zai", "openai", "anthropic", "ollama", "localai", "openrouter"}
}

// MissingLyricsItem describes a track and whether it already has a synced
// lyrics sidecar (.lrc) and/or a Russian translation sidecar (.ru.lrc). Used
// by external orchestrators (Mixarr) to decide which tracks still need lyrics
// enrichment or RU translation.
type MissingLyricsItem struct {
	MediaFileID    string `json:"mediaFileId"`
	Title          string `json:"title"`
	Artist         string `json:"artist"`
	HasLyrics      bool   `json:"hasLyrics"`
	HasTranslation bool   `json:"hasTranslation"`
}

// MissingLyrics lists the tracks of an artist or album and flags those missing
// a synced lyrics sidecar. Exactly one of artistId/albumId should be set.
func (s *Service) MissingLyrics(ctx context.Context, userId, artistId, albumId string) ([]MissingLyricsItem, error) {
	var filters squirrel.Sqlizer
	switch {
	case albumId != "":
		filters = squirrel.Eq{"album_id": albumId}
	case artistId != "":
		// artist_id is the actual performing artist; album_artist_id covers the
		// album artist. Match either to be useful from an artist picker.
		filters = squirrel.Or{
			squirrel.Eq{"artist_id": artistId},
			squirrel.Eq{"album_artist_id": artistId},
		}
	default:
		filters = squirrel.Eq{"1": 0} // match nothing
	}

	mfs, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: filters,
		Sort:    "album",
		Max:     500,
	})
	if err != nil {
		return nil, err
	}

	store := newLyricsStore()
	items := make([]MissingLyricsItem, 0, len(mfs))
	for i := range mfs {
		mf := &mfs[i]
		hasLrc := store.hasSidecar(mf, ".lrc")
		// HasTranslation is true when a .ru.lrc sidecar exists OR when the original
		// .lrc is already in Russian (no translation needed — the original IS the
		// translation). This way Russian artists don't show as "missing RU".
		hasTrans := store.hasSidecar(mf, ".ru.lrc")
		if !hasTrans && hasLrc {
			if original, ok := store.readSidecar(mf, ".lrc"); ok && original != "" {
				if isRussianText(stripLRCTimestamps(original)) {
					hasTrans = true
				}
			}
		}
		items = append(items, MissingLyricsItem{
			MediaFileID:    mf.ID,
			Title:          mf.Title,
			Artist:         mf.Artist,
			HasLyrics:      hasLrc,
			HasTranslation: hasTrans,
		})
	}
	return items, nil
}

// MissingDecode lists the tracks of an artist/album that do not yet have a
// meaning-decode sidecar (.ai.decode.md). Used by external orchestrators (Mixarr)
// to decide which tracks still need decoding.
func (s *Service) MissingDecode(ctx context.Context, userId, artistId, albumId string) ([]MissingLyricsItem, error) {
	var filters squirrel.Sqlizer
	switch {
	case albumId != "":
		filters = squirrel.Eq{"album_id": albumId}
	case artistId != "":
		filters = squirrel.Or{
			squirrel.Eq{"artist_id": artistId},
			squirrel.Eq{"album_artist_id": artistId},
		}
	default:
		filters = squirrel.Eq{"1": 0}
	}

	mfs, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: filters,
		Sort:    "album",
		Max:     500,
	})
	if err != nil {
		return nil, err
	}

	store := newLyricsStore()
	items := make([]MissingLyricsItem, 0, len(mfs))
	for i := range mfs {
		mf := &mfs[i]
		items = append(items, MissingLyricsItem{
			MediaFileID: mf.ID,
			Title:       mf.Title,
			Artist:      mf.Artist,
			HasLyrics:   store.hasSidecar(mf, ".ai.decode.md"),
		})
	}
	return items, nil
}

// ──────────────────────────────────────────────────────────────────────────
// Phase endpoints (3-phase pipeline): LRCLIB-only, translate-batch, decode-batch.
// These let Mixarr drive the phases separately so Z.ai quota is spent only on
// what's actually missing, and translation/decode are batched (5 songs → 1 LLM
// call) to stay under Z.ai's concurrent-request limit.
// ──────────────────────────────────────────────────────────────────────────

// FetchOriginalLyrics fetches the ORIGINAL synced lyrics from LRCLIB only (no
// translation, no AI). Synchronous. Idempotent: if a .lrc sidecar already
// exists, it returns immediately. Writes the .lrc and updates the DB so the web
// player shows it. This is "Phase A" of the 3-phase pipeline.
func (s *Service) FetchOriginalLyrics(ctx context.Context, userId, mediaFileId string) (bool, error) {
	mf, err := s.ds.MediaFile(ctx).Get(mediaFileId)
	if err != nil {
		return false, fmt.Errorf("loading media file: %w", err)
	}

	store := newLyricsStore()
	if store.hasSidecar(mf, ".lrc") {
		return true, nil // already have original
	}

	// LRCLIB only (no AI provider needed for this step).
	original, lang, err := s.fetchOriginal(ctx, nil, mf)
	if err != nil {
		return false, err
	}
	if original == "" {
		return false, nil // not found in LRCLIB
	}

	if err := store.writeSidecar(mf, ".lrc", ensureLangHeader(original, lang)); err != nil {
		return false, err
	}

	// Update the DB so the web player shows it immediately.
	if err := s.persistLyricsToDB(ctx, mf, original, ""); err != nil {
		log.Warn(ctx, "Failed to update media_file.lyrics (sidecar written)", "error", err)
	}
	return true, nil
}

// TranslateBatchRequest is the body of POST /api/ai/translate/batch.
type TranslateBatchRequest struct {
	Items  []TranslateBatchItem `json:"items"`
	ToLang string               `json:"toLang"`
	Model  string               `json:"model,omitempty"`
}

// TranslateBatchItem is one song in a batch translation request.
type TranslateBatchItem struct {
	MediaFileID string `json:"mediaFileId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Lyrics      string `json:"lyrics"`
}

// TranslateBatchResult is the per-song outcome.
type TranslateBatchResult struct {
	MediaFileID string `json:"mediaFileId"`
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	Skipped     bool   `json:"skipped,omitempty"`
}

// TranslateBatch translates up to N songs in a SINGLE LLM call, splitting the
// model's output by a sentinel separator. This is "Phase B" — it requires the
// original .lrc to already exist (Phase A). Idempotent: songs with an existing
// .ru.lrc are skipped.
func (s *Service) TranslateBatch(ctx context.Context, userId string, req *TranslateBatchRequest) ([]TranslateBatchResult, error) {
	provider, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	if req.ToLang == "" {
		req.ToLang = "ru"
	}

	store := newLyricsStore()
	results := make([]TranslateBatchResult, 0, len(req.Items))

	// Partition: skip songs that already have .ru.lrc, and songs whose original
	// is Russian (nothing to translate). Collect the rest for the batch call.
	type pending struct {
		idx int
		mf  *model.MediaFile
		lrc string
	}
	var queue []pending
	for i, item := range req.Items {
		mf, err := s.ds.MediaFile(ctx).Get(item.MediaFileID)
		if err != nil {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, Error: "media file not found"})
			continue
		}
		if store.hasSidecar(mf, ".ru.lrc") {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, OK: true, Skipped: true})
			continue
		}
		// Read the original .lrc (must exist from Phase A).
		original, ok := store.readSidecar(mf, ".lrc")
		if !ok {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, Error: "no .lrc original"})
			continue
		}
		plain := stripLRCTimestamps(original)
		if isRussianText(plain) {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, OK: true, Skipped: true})
			continue
		}
		queue = append(queue, pending{idx: i, mf: mf, lrc: original})
	}

	if len(queue) == 0 {
		return results, nil
	}

	// Process each song with its OWN translate call (sequential, not batched).
	// provider.Translate rebuilds the prompt from TranslateRequest fields, so a
	// combined batch prompt gets mangled into "Song: batch by batch". Sequential
	// calls give Z.ai the real song title/artist/lyrics for each track.
	for _, p := range queue {
		plain := stripLRCTimestamps(p.lrc)
		resp, err := provider.Translate(ctx, &TranslateRequest{
			Title:  p.mf.Title,
			Artist: p.mf.Artist,
			Lyrics: plain,
			ToLang: req.ToLang,
			Model:  req.Model,
		})
		if err != nil {
			results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, Error: err.Error()})
			continue
		}
		translation := strings.TrimSpace(resp.Translation)
		if translation == "" || strings.Contains(strings.ToLower(translation), "could not find the lyrics") {
			results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, Error: "empty translation"})
			continue
		}
		ru := alignToOriginalTiming(p.lrc, translation)
		if err := store.writeSidecar(p.mf, ".ru.lrc", ensureLangHeader(ru, "ru")); err != nil {
			results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, Error: err.Error()})
			continue
		}
		if err := s.persistLyricsToDB(ctx, p.mf, p.lrc, translation); err != nil {
			log.Warn(ctx, "Failed to update DB lyrics (sidecar written)", "error", err)
		}
		results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, OK: true})
	}
	return results, nil
}

// DecodeBatchRequest is the body of POST /api/ai/decode/batch.
type DecodeBatchRequest struct {
	Items []DecodeBatchItem `json:"items"`
	Model string            `json:"model,omitempty"`
}

// DecodeBatchItem is one song in a batch decode request.
type DecodeBatchItem struct {
	MediaFileID string `json:"mediaFileId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Lyrics      string `json:"lyrics"`
}

// DecodeBatch generates meaning-decode for up to N songs in a SINGLE LLM call,
// splitting the model's output by a per-song marker. Idempotent: songs with an
// existing .ai.decode.md are skipped.
func (s *Service) DecodeBatch(ctx context.Context, userId string, req *DecodeBatchRequest) ([]TranslateBatchResult, error) {
	provider, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}

	store := newLyricsStore()
	results := make([]TranslateBatchResult, 0, len(req.Items))

	type pending struct {
		idx int
		mf  *model.MediaFile
		it  DecodeBatchItem
	}
	var queue []pending
	for i, item := range req.Items {
		mf, err := s.ds.MediaFile(ctx).Get(item.MediaFileID)
		if err != nil {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, Error: "media file not found"})
			continue
		}
		if store.hasSidecar(mf, ".ai.decode.md") {
			results = append(results, TranslateBatchResult{MediaFileID: item.MediaFileID, OK: true, Skipped: true})
			continue
		}
		queue = append(queue, pending{idx: i, mf: mf, it: item})
	}

	if len(queue) == 0 {
		return results, nil
	}

	// Process each song with its OWN decode call (sequential, not batched).
	// Batching via a combined prompt doesn't work because provider.Decode
	// rebuilds the prompt from DecodeRequest fields, ignoring the batch
	// instructions. Sequential calls are reliable and Z.ai handles them fine.
	for _, p := range queue {
		// Read lyrics from .lrc sidecar if available.
		lyricsText := p.it.Lyrics
		if strings.TrimSpace(lyricsText) == "" {
			if lrc, ok := store.readSidecar(p.mf, ".lrc"); ok {
				lyricsText = stripLRCTimestamps(lrc)
			}
		}
		resp, err := provider.Decode(ctx, &DecodeRequest{
			Title:       p.it.Title,
			Artist:      p.it.Artist,
			Album:       p.it.Album,
			Lyrics:      lyricsText,
			Model:       req.Model,
			MediaFileID: p.mf.ID,
		})
		if err != nil {
			results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, Error: err.Error()})
			continue
		}
		text := strings.TrimSpace(resp.Text)
		if text == "" {
			results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, Error: "empty decode"})
			continue
		}
		s.persistResult(ctx, p.mf.ID, ".ai.decode.md", text)
		results = append(results, TranslateBatchResult{MediaFileID: p.mf.ID, OK: true})
	}
	return results, nil
}
