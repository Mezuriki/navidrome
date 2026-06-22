package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	systemPrompt := "You are a music expert and professional translator. " +
		"If you know the song, recall its full lyrics and translate them. " +
		"If you do not know the song and no lyrics were provided, say so honestly. " +
		"Answer in plain text. Show the original lyrics first, then a line with '---', then the translation."

	recalled := false
	var prompt string
	if strings.TrimSpace(req.Lyrics) != "" {
		prompt = fmt.Sprintf("Song: %s by %s\n\nHere are the lyrics to translate to %s:\n\n%s",
			req.Title, req.Artist, langName(req.ToLang), req.Lyrics)
	} else {
		recalled = true
		prompt = fmt.Sprintf("Song: %s by %s\n\n"+
			"No lyrics were provided. If you know this song, recall its lyrics and then translate them to %s. "+
			"If you don't know the song, reply: \"I couldn't find the lyrics for this song.\"",
			req.Title, req.Artist, langName(req.ToLang))
	}

	resp, err := p.callGenerate(ctx, systemPrompt, prompt, req.Model)
	if err != nil {
		return nil, err
	}

	return &TranslateResponse{
		Translation: resp,
		Recalled:    recalled,
		Model:       p.getModel(req.Model),
	}, nil
}

// Analyze analyzes a track using Ollama (plain-text response)
func (p *OllamaProvider) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	systemPrompt := "You are a knowledgeable music critic. Analyze the track and write a concise, engaging " +
		"description: genre and style, mood, themes, and a few similar artists. Plain text, short paragraphs " +
		"and bullet points. Do not output JSON."

	var b strings.Builder
	fmt.Fprintf(&b, "Track: %s\nArtist: %s\n", req.Title, req.Artist)
	if req.Album != "" {
		fmt.Fprintf(&b, "Album: %s\n", req.Album)
	}
	if req.Year > 0 {
		fmt.Fprintf(&b, "Year: %d\n", req.Year)
	}
	if req.Genre != "" {
		fmt.Fprintf(&b, "Listed genre: %s\n", req.Genre)
	}
	if strings.TrimSpace(req.Lyrics) != "" {
		fmt.Fprintf(&b, "\nLyrics excerpt:\n%s\n", truncateString(req.Lyrics, 600))
	}

	resp, err := p.callGenerate(ctx, systemPrompt, b.String(), req.Model)
	if err != nil {
		return nil, err
	}

	return &AnalyzeResponse{
		Text:  resp,
		Model: p.getModel(req.Model),
	}, nil
}

// Decode analyzes the meaning of a song using Ollama (plain-text response)
func (p *OllamaProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	systemPrompt := "You are a thoughtful music analyst who explains what songs mean. " +
		"Interpret the song: its meaning and message, mood, main themes, and a short commentary. " +
		"Plain text, short paragraphs. If you don't know the song, base it on the title/artist and say so. " +
		"Do not output JSON or empty responses."

	var b strings.Builder
	fmt.Fprintf(&b, "Song: %s\nArtist: %s\n", req.Title, req.Artist)
	if req.Album != "" {
		fmt.Fprintf(&b, "Album: %s\n", req.Album)
	}
	if strings.TrimSpace(req.Lyrics) != "" {
		fmt.Fprintf(&b, "\nLyrics:\n%s\n", req.Lyrics)
	} else {
		b.WriteString("\nNo lyrics provided — interpret based on the title and artist.\n")
	}

	resp, err := p.callGenerate(ctx, systemPrompt, b.String(), req.Model)
	if err != nil {
		return nil, err
	}

	return &DecodeResponse{
		Text:  resp,
		Model: p.getModel(req.Model),
	}, nil
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
		Model:  p.getModel(model),
		System: systemPrompt,
		Prompt: userPrompt,
		Stream: false,
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
