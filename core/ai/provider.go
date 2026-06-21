package ai

import "context"

// LLMProvider defines the interface for AI/LLM providers
type LLMProvider interface {
	// Translate translates text from one language to another
	Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error)

	// Analyze analyzes a track (lyrics, metadata) and returns insights
	Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error)

	// Decode analyzes the meaning and mood of a song
	Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error)

	// Name returns the provider name
	Name() string
}

// TranslateRequest contains translation parameters
type TranslateRequest struct {
	Text      string `json:"text"`
	FromLang  string `json:"fromLang,omitempty"`
	ToLang    string `json:"toLang"`
	Model     string `json:"model,omitempty"`
}

// TranslateResponse contains the translation result
type TranslateResponse struct {
	Translation string `json:"translation"`
	SourceLang  string `json:"sourceLang,omitempty"`
	Model       string `json:"model"`
}

// AnalyzeRequest contains track analysis parameters
type AnalyzeRequest struct {
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Year      int    `json:"year,omitempty"`
	Genre     string `json:"genre,omitempty"`
	Lyrics    string `json:"lyrics,omitempty"`
	Model     string `json:"model,omitempty"`
}

// AnalyzeResponse contains track analysis results
type AnalyzeResponse struct {
	Genre        string   `json:"genre,omitempty"`
	Mood         []string `json:"mood,omitempty"`
	Style        []string `json:"style,omitempty"`
	Themes       []string `json:"themes,omitempty"`
	Description  string   `json:"description,omitempty"`
	SimilarArtists []string `json:"similarArtists,omitempty"`
	Model        string   `json:"model"`
}

// DecodeRequest contains track decoding parameters
type DecodeRequest struct {
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Lyrics    string `json:"lyrics,omitempty"`
	Model     string `json:"model,omitempty"`
}

// DecodeResponse contains track decoding results
type DecodeResponse struct {
	Meaning      string   `json:"meaning"`
	Mood         string   `json:"mood"`
	Themes       []string `json:"themes"`
	Interpretation string `json:"interpretation"`
	Model        string   `json:"model"`
}

// Config holds AI provider configuration
type Config struct {
	Provider      string `json:"provider"`
	APIKey        string `json:"apiKey"`
	APIEndpoint   string `json:"apiEndpoint"`
	Model         string `json:"model"`
	DefaultLang   string `json:"defaultLanguage"`
}
