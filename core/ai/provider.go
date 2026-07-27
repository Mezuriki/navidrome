package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// LLMProvider defines the interface for AI/LLM providers
type LLMProvider interface {
	// Translate recalls/recovers the song lyrics and translates them to the
	// target language. Responses are plain text (no JSON) for broad model support.
	Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error)

	// Analyze returns a free-form analysis of the track (genre, mood, style, themes).
	Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error)

	// Decode returns a free-form interpretation of the song's meaning and mood.
	Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error)

	// RecallLyrics attempts to recover the lyrics for a song the model knows and
	// returns them as LRC text with approximate timestamps spread across the
	// given duration. Returns an empty string if the model does not know it.
	RecallLyrics(ctx context.Context, req *RecallRequest) (*RecallResponse, error)

	// TestConnection sends a minimal request to verify that the provider is
	// reachable and the credentials are valid. Returns nil on success and a
	// descriptive error otherwise.
	TestConnection(ctx context.Context) error

	// Name returns the provider name
	Name() string
}

// TranslateRequest contains translation parameters.
// If Lyrics is empty, the model is expected to recall the song from Title/Artist.
// MediaFileID, when set, instructs the service to persist the translation to a
// sidecar file next to the media file.
type TranslateRequest struct {
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Lyrics      string `json:"lyrics,omitempty"`
	ToLang      string `json:"toLang"`
	Model       string `json:"model,omitempty"`
	MediaFileID string `json:"mediaFileId,omitempty"`
}

// TranslateResponse contains the translation result (plain text).
type TranslateResponse struct {
	Translation string `json:"translation"`
	Recalled    bool   `json:"recalled"` // true if the model had to recall the lyrics itself
	Model       string `json:"model"`
}

// AnalyzeRequest contains track analysis parameters
type AnalyzeRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Year        int    `json:"year,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Lyrics      string `json:"lyrics,omitempty"`
	Model       string `json:"model,omitempty"`
	MediaFileID string `json:"mediaFileId,omitempty"`
}

// AnalyzeResponse contains the analysis result (plain text).
type AnalyzeResponse struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// DecodeRequest contains track decoding parameters
type DecodeRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Lyrics      string `json:"lyrics,omitempty"`
	Model       string `json:"model,omitempty"`
	MediaFileID string `json:"mediaFileId,omitempty"`
}

// DecodeResponse contains the decoding result (plain text).
type DecodeResponse struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// RecallRequest asks the provider to recover and timestamp the lyrics.
type RecallRequest struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album,omitempty"`
	Duration float64 `json:"duration,omitempty"` // seconds
	Model    string `json:"model,omitempty"`
}

// RecallResponse holds the LRC-formatted lyrics recovered by the model.
type RecallResponse struct {
	LRC   string `json:"lrc"`
	Model string `json:"model"`
}

// Config holds AI provider configuration
type Config struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"apiKey"`
	APIEndpoint string `json:"apiEndpoint"`
	Model       string `json:"model"`
	DefaultLang string `json:"defaultLanguage"`
}

// LyricsTaskStatus is persisted in UserProps per media file so the UI can poll
// the progress of a background lyrics fetch. Steps advance in the order:
// queued -> lyrics -> translation -> done (or error).
type LyricsTaskStatus struct {
	Status    string `json:"status"`              // queued | running | done | error
	Step      string `json:"step,omitempty"`      // lyrics | translation | decode
	Error     string `json:"error,omitempty"`     // populated when status == error
	UpdatedAt int64  `json:"updatedAt"`           // unix seconds
	LyricsHit bool   `json:"lyricsHit,omitempty"` // whether the original was found
}

const (
	StatusQueued     = "queued"
	StatusRunning    = "running"
	StatusDone       = "done"
	StatusError      = "error"
	StepLyrics       = "lyrics"
	StepTranslation  = "translation"
)

// truncateString shortens a string to maxLen characters, appending "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// recallPrompts builds the system + user prompts used by RecallLyrics across all
// providers. The system prompt asks for LRC output with evenly spread, only
// approximate timestamps; the user prompt identifies the song and its duration.
func recallPrompts(req *RecallRequest) (system, user string) {
	system = "You return song lyrics as LRC text. Follow these rules exactly:\n" +
		"1. If you actually know the song, output ONLY the lyrics in LRC format, one line per vocal line, " +
		"each prefixed by a [mm:ss.xx] timestamp.\n" +
		"2. Spread the timestamps evenly across the WHOLE song duration given to you. " +
		"Timings need only be approximate; do not try to be word-precise.\n" +
		"3. If you do not actually know the song, output exactly: I could not find the lyrics for this song.\n" +
		"4. Output only the LRC. No explanations, no markdown."

	mins := 0
	secs := 0
	if req.Duration > 0 {
		total := int(req.Duration)
		mins = total / 60
		secs = total % 60
	}
	user = fmt.Sprintf("Song: \"%s\" by \"%s\".", req.Title, req.Artist)
	if req.Album != "" {
		user += fmt.Sprintf("\nAlbum: %s", req.Album)
	}
	if req.Duration > 0 {
		user += fmt.Sprintf("\nApproximate duration: %d:%02d (mm:ss).", mins, secs)
	}
	return system, user
}

// decodePrompts builds the system + user prompts for Decode, shared by all
// providers. The output is Markdown (Russian) with fixed section headings so the
// stored .ai.decode.md renders cleanly in the AI drawer.
func decodePrompts(req *DecodeRequest) (system, user string) {
	system = "Ты — вдумчивый музыкальный аналитик, объясняющий смысл песен на русском языке. " +
		"Проанализируй песню и верни ответ СТРОГО в формате Markdown со следующими разделами:\n\n" +
		"## Смысл\nКраткое описание общего смысла и посыла песни.\n\n" +
		"## Настроение\nЭмоциональная атмосфера и настроение.\n\n" +
		"## Темы\nОсновные темы и мотивы (списком).\n\n" +
		"## Комментарий\nКороткий комментарий о контексте, если уместно.\n\n" +
		"Пиши на русском. Не выводи JSON. Если не знаешь песню хорошо — опирайся на название и исполнителя и скажи об этом."

	var b strings.Builder
	fmt.Fprintf(&b, "Песня: %s\nИсполнитель: %s", req.Title, req.Artist)
	if req.Album != "" {
		fmt.Fprintf(&b, "\nАльбом: %s", req.Album)
	}
	if strings.TrimSpace(req.Lyrics) != "" {
		fmt.Fprintf(&b, "\n\nТекст песни:\n%s", req.Lyrics)
	} else {
		b.WriteString("\n\nТекст не предоставлен — интерпретируй по названию и исполнителю.")
	}
	return system, b.String()
}

// normalizeLyrics returns empty string for values that represent "no lyrics"
// (empty, "[]", "null", etc.), which is how Navidrome serializes an empty
// lyric list in the MediaFile datagrid payload.
func normalizeLyrics(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "", "[]", "[ ]", "null", `""`, `''`:
		return ""
	}
	return s
}

// thinkRe matches reasoning/thinking blocks that some models (Qwen3, DeepSeek-R1)
// emit inline, e.g. <think>...</think>. We strip them so the user only sees the
// final answer.
var thinkRe = regexp.MustCompile(`(?is)<think>.*?</think>`)

// stripThinking removes <think>...</think> blocks and trims whitespace.
func stripThinking(s string) string {
	s = thinkRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
