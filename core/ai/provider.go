package ai

import "context"

// LLMProvider defines the interface for AI/LLM providers
type LLMProvider interface {
	// Translate recalls/recovers the song lyrics and translates them to the
	// target language. Responses are plain text (no JSON) for broad model support.
	Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error)

	// Analyze returns a free-form analysis of the track (genre, mood, style, themes).
	Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error)

	// Decode returns a free-form interpretation of the song's meaning and mood.
	Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error)

	// Name returns the provider name
	Name() string
}

// TranslateRequest contains translation parameters.
// If Lyrics is empty, the model is expected to recall the song from Title/Artist.
type TranslateRequest struct {
	Title    string `json:"title,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Lyrics   string `json:"lyrics,omitempty"`
	ToLang   string `json:"toLang"`
	Model    string `json:"model,omitempty"`
}

// TranslateResponse contains the translation result (plain text).
type TranslateResponse struct {
	Translation string `json:"translation"`
	Recalled    bool   `json:"recalled"` // true if the model had to recall the lyrics itself
	Model       string `json:"model"`
}

// AnalyzeRequest contains track analysis parameters
type AnalyzeRequest struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Year   int    `json:"year,omitempty"`
	Genre  string `json:"genre,omitempty"`
	Lyrics string `json:"lyrics,omitempty"`
	Model  string `json:"model,omitempty"`
}

// AnalyzeResponse contains the analysis result (plain text).
type AnalyzeResponse struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// DecodeRequest contains track decoding parameters
type DecodeRequest struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Lyrics string `json:"lyrics,omitempty"`
	Model  string `json:"model,omitempty"`
}

// DecodeResponse contains the decoding result (plain text).
type DecodeResponse struct {
	Text  string `json:"text"`
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

// truncateString shortens a string to maxLen characters, appending "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
