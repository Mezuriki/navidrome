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
	defaultOpenAIEndpoint = "https://api.openai.com/v1"
	defaultOpenAIModel    = "gpt-4o-mini"
)

// OpenAIProvider implements LLMProvider for OpenAI API
type OpenAIProvider struct {
	client     *http.Client
	apiKey     string
	endpoint   string
	model      string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, endpoint, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if endpoint == "" {
		endpoint = defaultOpenAIEndpoint
	}

	if model == "" {
		model = defaultOpenAIModel
	}

	return &OpenAIProvider{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:   apiKey,
		endpoint: endpoint,
		model:    model,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Translate translates text using OpenAI
func (p *OpenAIProvider) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a professional translator. Translate the given text to the target language while preserving meaning, tone, and formatting. Only return the translated text, nothing else.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Translate to %s:\n\n%s", req.ToLang, req.Text),
		},
	}

	resp, err := p.callChat(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	return &TranslateResponse{
		Translation: resp,
		Model:       p.getModel(req.Model),
	}, nil
}

// Analyze analyzes a track using OpenAI
func (p *OpenAIProvider) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
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

	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	resp, err := p.callChatJSON(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	var result AnalyzeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Fallback: if JSON parsing fails, create a basic response
		return &AnalyzeResponse{
			Description: resp,
			Model:       p.getModel(req.Model),
		}, nil
	}

	result.Model = p.getModel(req.Model)
	return &result, nil
}

// Decode analyzes the meaning of a song using OpenAI
func (p *OpenAIProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
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

	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	resp, err := p.callChatJSON(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	var result DecodeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Fallback
		return &DecodeResponse{
			Meaning:        resp,
			Model:          p.getModel(req.Model),
		}, nil
	}

	result.Model = p.getModel(req.Model)
	return &result, nil
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest represents the API request
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// chatResponse represents the API response
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type   string `json:"type"`
	} `json:"error,omitempty"`
}

// callChat makes a chat completion request and returns the text response
func (p *OpenAIProvider) callChat(ctx context.Context, messages []Message, model string) (string, error) {
	respJSON, err := p.callChatJSON(ctx, messages, model)
	if err != nil {
		return "", err
	}

	var resp chatResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// callChatJSON makes a chat completion request and returns the raw JSON response
func (p *OpenAIProvider) callChatJSON(ctx context.Context, messages []Message, model string) (string, error) {
	req := chatRequest{
		Model:    p.getModel(model),
		Messages: messages,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	return string(body), nil
}

func (p *OpenAIProvider) getModel(model string) string {
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
