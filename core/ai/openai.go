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

// NewOpenAIProvider creates a new OpenAI-compatible provider.
// The apiKey is optional — local OpenAI-compatible servers (Ollama, LocalAI)
// do not require one.
func NewOpenAIProvider(apiKey, endpoint, model string) (*OpenAIProvider, error) {
	if endpoint == "" {
		endpoint = defaultOpenAIEndpoint
	}

	if model == "" {
		model = defaultOpenAIModel
	}

	return &OpenAIProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
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

// Translate recalls (or uses provided) lyrics and translates them to the target language.
// The response is plain text for broad model compatibility.
func (p *OpenAIProvider) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
	systemPrompt := "You translate song lyrics. Follow these rules exactly:\n" +
		"1. If real lyrics are provided, translate ONLY those lyrics to the target language. Do not add or invent anything.\n" +
		"2. If no lyrics are provided and you are confident you know the song, write the original lyrics first, then a single line containing exactly ---, then the translation.\n" +
		"3. If you do not actually know the song, output exactly: I could not find the lyrics for this song.\n" +
		"4. Output only the result. No explanations, no preamble, no notes, no markdown."

	lyrics := normalizeLyrics(req.Lyrics)
	recalled := lyrics == ""
	var userContent string
	if !recalled {
		userContent = fmt.Sprintf("Song: \"%s\" by \"%s\".\n\nTranslate the lyrics below to %s. Output only the translation:\n\n%s",
			req.Title, req.Artist, langName(req.ToLang), lyrics)
	} else {
		userContent = fmt.Sprintf("Song: \"%s\" by \"%s\".\n"+
			"No lyrics were provided. If you know this song, output the original lyrics, then a line with ---, then the %s translation. "+
			"If you do not actually know this song, output exactly: I could not find the lyrics for this song.",
			req.Title, req.Artist, langName(req.ToLang))
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	resp, err := p.callChat(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	return &TranslateResponse{
		Translation: stripThinking(resp),
		Recalled:    recalled,
		Model:       p.getModel(req.Model),
	}, nil
}

// Analyze returns a free-form analysis of the track (plain text).
func (p *OpenAIProvider) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	systemPrompt := "You are a knowledgeable music critic. Analyze the given track and write a concise, " +
		"engaging description covering: likely genre and style, mood and atmosphere, themes, and a few " +
		"similar artists. Write in plain text using short paragraphs and bullet points. Do not output JSON."

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

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: b.String()},
	}

	resp, err := p.callChat(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	return &AnalyzeResponse{
		Text:  stripThinking(resp),
		Model: p.getModel(req.Model),
	}, nil
}

// Decode returns a free-form interpretation of the song's meaning (plain text).
func (p *OpenAIProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	systemPrompt := "You are a thoughtful music analyst who explains what songs mean. " +
		"Interpret the song: its overall meaning and message, the mood, the main themes, and a short " +
		"commentary on what it might be about. Write in plain text using short paragraphs. " +
		"If you don't know the song well, base your interpretation on the title and artist, and say so. " +
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

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: b.String()},
	}

	resp, err := p.callChat(ctx, messages, req.Model)
	if err != nil {
		return nil, err
	}

	return &DecodeResponse{
		Text:  stripThinking(resp),
		Model: p.getModel(req.Model),
	}, nil
}

// langName maps a language code to a human-readable name for prompts.
func langName(code string) string {
	names := map[string]string{
		"en": "English", "ru": "Russian", "de": "German", "fr": "French",
		"es": "Spanish", "it": "Italian", "pt": "Portuguese",
		"ja": "Japanese", "zh": "Chinese", "ko": "Korean",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
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
	// Some local servers (Ollama, LocalAI) don't require a key; only send it if set.
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

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
