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

	"github.com/navidrome/navidrome/log"
)

const (
	defaultGeminiEndpoint = "https://generativelanguage.googleapis.com"
	defaultGeminiModel    = "gemini-3.5-flash"
)

// geminiFallbackModels is the quota-fallback chain, ordered from most to least
// capable. When the primary model returns 429 / RESOURCE_EXHAUSTED (the Gemini
// free tier allows ~20 RPD per model), the call is retried with the next entry.
// Each model has its own independent daily quota, so walking the chain gives
// roughly 4x the per-day budget before giving up.
//
// Only models that support generateContent are listed, and only currently-live
// models (the 2.5 family was deprecated/removed by Google; 2.0 is the oldest
// available). Note that google_search grounding is honoured by the 3.x family;
// once we fall to 2.0 we rely on the model's own knowledge.
var geminiFallbackModels = []string{
	"gemini-3.5-flash",
	"gemini-3-flash-preview",
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
}

// GeminiProvider implements LLMProvider against Google's native Gemini API
// (generativelanguage.googleapis.com). This is the ONLY endpoint that accepts
// the new "AQ."-prefixed Authentication Keys; the OpenAI-compatible layer does
// not. For web-grounded answers (lyrics recall, translation) the google_search
// tool is enabled via the "tools" field.
type GeminiProvider struct {
	client   *http.Client
	apiKey   string
	endpoint string
	model    string
}

// NewGeminiProvider creates a new Gemini provider. apiKey is required for
// Google AI Studio keys; endpoint/model fall back to sane defaults.
func NewGeminiProvider(apiKey, endpoint, model string) (*GeminiProvider, error) {
	if endpoint == "" {
		endpoint = defaultGeminiEndpoint
	}
	if model == "" {
		model = defaultGeminiModel
	}
	return &GeminiProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey:   apiKey,
		endpoint: strings.TrimRight(endpoint, "/"),
		model:    model,
	}, nil
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Translate recalls (or uses provided) lyrics and translates them to the target
// language. The google_search tool is enabled so the model can look up lyrics it
// does not have memorized.
func (p *GeminiProvider) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
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

	resp, err := p.callGenerate(ctx, p.getModel(req.Model), systemPrompt, userContent, true)
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
func (p *GeminiProvider) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {	systemPrompt := "You are a knowledgeable music critic. Analyze the given track and write a concise, " +
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

	resp, err := p.callGenerate(ctx, p.getModel(req.Model), systemPrompt, b.String(), false)
	if err != nil {
		return nil, err
	}

	return &AnalyzeResponse{
		Text:  stripThinking(resp),
		Model: p.getModel(req.Model),
	}, nil
}

// Decode returns a free-form interpretation of the song's meaning (plain text).
// google_search is enabled so the model can ground its interpretation in real
// information about the song.
func (p *GeminiProvider) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	systemPrompt, userPrompt := decodePrompts(req)
	resp, err := p.callGenerate(ctx, p.getModel(req.Model), systemPrompt, userPrompt, false)
	if err != nil {
		return nil, err
	}

	return &DecodeResponse{
		Text:  stripThinking(resp),
		Model: p.getModel(req.Model),
	}, nil
}

// RecallLyrics recovers the lyrics for a song via google_search grounding and
// returns them as LRC text with approximate timestamps spread across the
// duration. Returns an empty LRC if the model does not know the song.
func (p *GeminiProvider) RecallLyrics(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
	systemPrompt, userPrompt := recallPrompts(req)
	resp, err := p.callGenerate(ctx, p.getModel(req.Model), systemPrompt, userPrompt, true)
	if err != nil {
		return nil, err
	}
	body := stripThinking(resp)
	if strings.Contains(body, "could not find the lyrics") {
		body = ""
	}
	return &RecallResponse{LRC: body, Model: p.getModel(req.Model)}, nil
}

// geminiContent mirrors the Gemini API content/parts schema.
type geminiContent struct {
	Role  string        `json:"role,omitempty"`
	Parts []geminiPart  `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

// geminiRequest is the body sent to :generateContent.
type geminiRequest struct {
	Contents          []geminiContent    `json:"contents"`
	SystemInstruction *geminiContent     `json:"systemInstruction,omitempty"`
	Tools             []geminiTool       `json:"tools,omitempty"`
	GenerationConfig  *geminiGeneration  `json:"generationConfig,omitempty"`
}

// geminiGeneration configures decoding. thinkingBudget=0 disables the built-in
// "thinking" mode of Gemini 3.5 Flash, which otherwise burns quota on hidden
// reasoning tokens and returns a thoughtSignature alongside the text.
type geminiGeneration struct {
	ThinkingConfig *geminiThinking `json:"thinkingConfig,omitempty"`
}

type geminiThinking struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

// geminiTool enables the google_search grounding tool when present.
type geminiTool struct {
	GoogleSearch map[string]interface{} `json:"google_search,omitempty"`
}

// geminiResponse mirrors the subset of the response we need.
type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason,omitempty"`
	} `json:"promptFeedback,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// callGenerate posts to the :generateContent endpoint and returns the assembled
// text from the first candidate. When useSearch is true, the google_search tool
// is attached so the model can ground its answer in web results; if that request
// fails (e.g. free-tier grounding quota exhausted), the call is retried once
// without the tool, falling back to the model's own knowledge.
//
// Thinking is always disabled (thinkingBudget=0): on Gemini 3.5 Flash it would
// otherwise spend hidden reasoning tokens that count against quota and add a
// thoughtSignature to every part.
func (p *GeminiProvider) callGenerate(ctx context.Context, model, systemPrompt, userPrompt string, useSearch bool) (string, error) {
	buildReq := func(withSearch bool) geminiRequest {
		req := geminiRequest{
			Contents: []geminiContent{
				{Role: "user", Parts: []geminiPart{{Text: userPrompt}}},
			},
			GenerationConfig: &geminiGeneration{
				ThinkingConfig: &geminiThinking{ThinkingBudget: 0},
			},
		}
		if systemPrompt != "" {
			req.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: systemPrompt}}}
		}
		if withSearch {
			req.Tools = []geminiTool{{GoogleSearch: map[string]interface{}{}}}
		}
		return req
	}

	// First attempt with the requested model. Try with google_search first; if
	// that fails for any reason, drop the tool and retry on the same model.
	current := model
	resp, err := p.doGenerate(ctx, current, buildReq(useSearch))
	if err != nil && useSearch {
		log.Warn(ctx, "Gemini grounded call failed, retrying without google_search", "model", current, "error", err)
		resp, err = p.doGenerate(ctx, current, buildReq(false))
	}

	// Quota fallback chain: while we keep hitting 429 / RESOURCE_EXHAUSTED, walk
	// down the model hierarchy. Each model has its own daily quota on the free
	// tier, so this multiplies the effective per-day budget. Only quota errors
	// trigger a switch; anything else (safety block, bad request, region) is
	// surfaced immediately.
	for isQuotaError(err) {
		next := nextFallbackModel(current)
		if next == "" {
			break // exhausted the chain
		}
		log.Warn(ctx, "Gemini quota exhausted, falling back to next model", "from", current, "to", next)
		current = next
		resp, err = p.doGenerate(ctx, current, buildReq(false))
	}
	if err != nil {
		return "", err
	}
	return resp, nil
}

// doGenerate serializes and sends a single request, returning the assembled
// text from the first candidate.
func (p *GeminiProvider) doGenerate(ctx context.Context, model string, req geminiRequest) (string, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", p.endpoint, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// The native Gemini endpoint accepts the key via the x-goog-api-key header.
	// The "?key=" query form also works but the header is preferred and is the
	// only form that accepts the new "AQ."-prefixed Authentication Keys reliably.
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if gr.Error != nil {
		return "", formatGeminiError(gr.Error)
	}

	if len(gr.Candidates) == 0 {
		// Could be blocked by safety filters or an empty response.
		if gr.PromptFeedback != nil && gr.PromptFeedback.BlockReason != "" {
			return "", fmt.Errorf("request blocked by Gemini: %s", gr.PromptFeedback.BlockReason)
		}
		return "", fmt.Errorf("no candidates returned")
	}

	var sb strings.Builder
	for _, part := range gr.Candidates[0].Content.Parts {
		if part.Text != "" {
			sb.WriteString(part.Text)
		}
	}
	return sb.String(), nil
}

// formatGeminiError turns the API error object into a human-readable message,
// calling out the well-known region-unsupported failure explicitly so the user
// understands they need a proxy/non-RU egress.
func formatGeminiError(e *struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}) error {
	if e.Status == "FAILED_PRECONDITION" && strings.Contains(strings.ToLower(e.Message), "location is not supported") {
		return fmt.Errorf("Gemini API rejected the request because the server's IP is in an unsupported region (likely RU). Route navidrome through a non-RU proxy/VPN. API message: %s", e.Message)
	}
	return &geminiAPIError{code: e.Code, status: e.Status, message: e.Message}
}

// geminiAPIError preserves the structured status so callers can branch on it
// (e.g. quota exhaustion triggers a model fallback).
type geminiAPIError struct {
	code    int
	status  string
	message string
}

func (e *geminiAPIError) Error() string {
	return fmt.Sprintf("Gemini API error (code %d, status %s): %s", e.code, e.status, e.message)
}

// isQuotaError reports whether the error is a transient/throttle failure that
// should trigger a switch to the next model in the chain. This covers explicit
// quota exhaustion (429/RESOURCE_EXHAUSTED) AND temporary overload (503
// "high demand"/UNAVAILABLE), which Gemini returns under spikes — treating it
// as retryable lets the fallback chain try a different, less-loaded model.
func isQuotaError(err error) bool {
	gae, ok := err.(*geminiAPIError)
	if !ok {
		return false
	}
	if gae.code == 429 || gae.code == 503 ||
		gae.status == "RESOURCE_EXHAUSTED" || gae.status == "UNAVAILABLE" {
		return true
	}
	// Some quota/overload messages arrive with a different code but clear wording.
	m := strings.ToLower(gae.message)
	return strings.Contains(m, "quota") ||
		strings.Contains(m, "rate limit") ||
		strings.Contains(m, "high demand") ||
		strings.Contains(m, "overloaded") ||
		strings.Contains(m, "try again later")
}

// nextFallbackModel returns the model that follows `current` in the fallback
// chain, or "" if `current` is the last (no further fallback available).
func nextFallbackModel(current string) string {
	for i, m := range geminiFallbackModels {
		if m == current && i+1 < len(geminiFallbackModels) {
			return geminiFallbackModels[i+1]
		}
	}
	return ""
}

// TestConnection verifies the provider is reachable and the API key is valid by
// sending a minimal generateContent request.
func (p *GeminiProvider) TestConnection(ctx context.Context) error {
	_, err := p.callGenerate(ctx, p.model, "", "Reply with exactly the word: pong", false)
	return err
}

func (p *GeminiProvider) getModel(model string) string {
	if model == "" {
		return p.model
	}
	return model
}
