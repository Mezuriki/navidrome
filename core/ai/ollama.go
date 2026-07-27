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

// Translate translates text using Ollama (strict prompt)
func (p *OllamaProvider) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
	systemPrompt := "You translate song lyrics. Follow these rules exactly:\n" +
		"1. If real lyrics are provided, translate ONLY those lyrics to the target language. Do not add or invent anything.\n" +
		"2. If no lyrics are provided and you are confident you know the song, write the original lyrics first, then a single line containing exactly ---, then the translation.\n" +
		"3. If you do not actually know the song, output exactly: I could not find the lyrics for this song.\n" +
		"4. Output only the result. No explanations, no preamble, no notes, no markdown."

	lyrics := normalizeLyrics(req.Lyrics)
	recalled := lyrics == ""
	var prompt string
	if !recalled {
		prompt = fmt.Sprintf("Song: \"%s\" by \"%s\".\n\nTranslate the lyrics below to %s. Output only the translation:\n\n%s",
			req.Title, req.Artist, langName(req.ToLang), lyrics)
	} else {
		prompt = fmt.Sprintf("Song: \"%s\" by \"%s\".\n"+
			"No lyrics were provided. If you know this song, output the original lyrics, then a line with ---, then the %s translation. "+
			"If you do not actually know this song, output exactly: I could not find the lyrics for this song.",
			req.Title, req.Artist, langName(req.ToLang))
	}

	resp, err := p.callGenerate(ctx, systemPrompt, prompt, req.Model)
	if err != nil {
		return nil, err
	}

	return &TranslateResponse{
		Translation: stripThinking(resp),
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
		Text:  stripThinking(resp),
		Model: p.getModel(req.Model),
	}, nil
}

// Decode analyzes the meaning of a song using Ollama (plain-text response)
func (p *OllamaProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	systemPrompt, userPrompt := decodePrompts(req)
	resp, err := p.callGenerate(ctx, systemPrompt, userPrompt, req.Model)
	if err != nil {
		return nil, err
	}

	return &DecodeResponse{
		Text:  stripThinking(resp),
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

// TestConnection verifies the Ollama server is reachable by hitting its
// /api/tags endpoint.
func (p *OllamaProvider) TestConnection(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/tags", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, truncateString(string(body), 300))
	}
	return nil
}

// RecallLyrics recovers the lyrics for a song and returns them as LRC text with
// approximate timestamps. Quality depends entirely on the loaded model.
func (p *OllamaProvider) RecallLyrics(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
	systemPrompt, userPrompt := recallPrompts(req)
	resp, err := p.callGenerate(ctx, systemPrompt, userPrompt, req.Model)
	if err != nil {
		return nil, err
	}
	body := stripThinking(resp)
	if strings.Contains(body, "could not find the lyrics") {
		body = ""
	}
	return &RecallResponse{LRC: body, Model: p.getModel(req.Model)}, nil
}
