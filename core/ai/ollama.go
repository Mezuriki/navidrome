package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultOllamaEndpoint = "http://localhost:11434/api"
	defaultOllamaModel    = "llama3"
)

// OllamaProvider implements LLMProvider for Ollama (local LLM)
type OllamaProvider struct {
	client     *http.Client
	endpoint   string
	model      string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(endpoint, model string) (*OllamaProvider, error) {
	if endpoint == "" {
		endpoint = defaultOllamaEndpoint
	}

	if model == "" {
		model = defaultOllamaModel
	}

	return &OllamaProvider{
		client: &http.Client{
			Timeout: 120 * time.Second, // Ollama can be slower
		},
		endpoint: endpoint,
		model:    model,
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Translate translates text using Ollama
func (p *OllamaProvider) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
	systemPrompt := fmt.Sprintf("You are a professional translator. Translate the given text to %s. Only return the translated text, nothing else.", req.ToLang)

	prompt := fmt.Sprintf("Translate:\n\n%s", req.Text)

	resp, err := p.callGenerate(ctx, systemPrompt, prompt, req.Model)
	if err != nil {
		return nil, err
	}

	return &TranslateResponse{
		Translation: resp,
		Model:       p.getModel(req.Model),
	}, nil
}

// Analyze analyzes a track using Ollama
func (p *OllamaProvider) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	systemPrompt := `You are a music expert AI. Analyze the given track information and provide insights.
Return a JSON object with the following structure:
{
  "genre": "primary genre",
  "mood": ["mood1", "mood2"],
  "style": ["style1", "style2"],
  "themes": ["theme1", "theme2"],
  "description": "brief description of the track",
  "similarArtists": ["artist1", "artist2"]
}`

	userPrompt := fmt.Sprintf("Track: %s by %s\nAlbum: %s", req.Title, req.Artist, req.Album)
	if req.Year > 0 {
		userPrompt += fmt.Sprintf(" (%d)", req.Year)
	}
	if req.Genre != "" {
		userPrompt += fmt.Sprintf("\nOriginal Genre: %s", req.Genre)
	}
	if req.Lyrics != "" {
		userPrompt += fmt.Sprintf("\nLyrics excerpt: %s", truncateString(req.Lyrics, 500))
	}

	resp, err := p.callGenerate(ctx, systemPrompt, userPrompt, req.Model)
	if err != nil {
		return nil, err
	}

	var result AnalyzeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return &AnalyzeResponse{
			Description: resp,
			Model:       p.getModel(req.Model),
		}, nil
	}

	result.Model = p.getModel(req.Model)
	return &result, nil
}

// Decode analyzes the meaning of a song using Ollama
func (p *OllamaProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	systemPrompt := `You are a music analyst specializing in song interpretation. Analyze the meaning and mood of the song.
Return a JSON object with the following structure:
{
  "meaning": "overall meaning of the song",
  "mood": "primary mood",
  "themes": ["theme1", "theme2"],
  "interpretation": "detailed interpretation of the song's message"
}`

	userPrompt := fmt.Sprintf("Song: %s by %s\nAlbum: %s", req.Title, req.Artist, req.Album)
	if req.Lyrics != "" {
		userPrompt += fmt.Sprintf("\n\nLyrics:\n%s", req.Lyrics)
	} else {
		userPrompt += "\n\nNo lyrics provided. Analyze based on title and artist."
	}

	resp, err := p.callGenerate(ctx, systemPrompt, userPrompt, req.Model)
	if err != nil {
		return nil, err
	}

	var result DecodeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return &DecodeResponse{
			Meaning: resp,
			Model:   p.getModel(req.Model),
		}, nil
	}

	result.Model = p.getModel(req.Model)
	return &result, nil
}

// ollamaRequest represents the Ollama API request
type ollamaRequest struct {
	Model    string `json:"model"`
	System   string `json:"system,omitempty"`
	Prompt   string `json:"prompt"`
	Stream   bool   `json:"stream"`
	Format   string `json:"format,omitempty"`
	Options  *struct {
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"options,omitempty"`
}

// ollamaResponse represents the Ollama API response
type ollamaResponse struct {
	Response string `json:"Response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// callGenerate makes a generation request to Ollama
func (p *OllamaProvider) callGenerate(ctx context.Context, systemPrompt, userPrompt string, model string) (string, error) {
	req := ollamaRequest{
		Model:    p.getModel(model),
		System:   systemPrompt,
		Prompt:   userPrompt,
		Stream:   false,
		Format:   "json",
		Options: &struct {
			Temperature float64 `json:"temperature,omitempty"`
		}{
			Temperature: 0.7,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	return ollamaResp.Response, nil
}

func (p *OllamaProvider) getModel(model string) string {
	if model == "" {
		return p.model
	}
	return model
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
