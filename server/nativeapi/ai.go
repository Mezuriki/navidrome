package nativeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/ai"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

type AIHandler struct {
	aiService *ai.Service
}

func NewAIHandler(aiService *ai.Service) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// userFromCtx extracts the authenticated user id from the request context.
func userFromCtx(r *http.Request) (string, bool) {
	u, ok := request.UserFrom(r.Context())
	if !ok {
		return "", false
	}
	return u.ID, true
}

// Translate handles lyrics translation requests.
func (h *AIHandler) Translate(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ai.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.aiService.Translate(r.Context(), userId, &req)
	if err != nil {
		log.Error(r.Context(), "AI translation failed", "error", err)
		writeAIError(w, err)
		return
	}
	encodeJSON(w, resp)
}

// Analyze handles track analysis requests.
func (h *AIHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ai.AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.aiService.Analyze(r.Context(), userId, &req)
	if err != nil {
		log.Error(r.Context(), "AI analyze failed", "error", err)
		writeAIError(w, err)
		return
	}
	encodeJSON(w, resp)
}

// Decode handles track "decode" (meaning/mood) requests.
func (h *AIHandler) Decode(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ai.DecodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.aiService.Decode(r.Context(), userId, &req)
	if err != nil {
		log.Error(r.Context(), "AI decode failed", "error", err)
		writeAIError(w, err)
		return
	}
	encodeJSON(w, resp)
}

// GetConfig returns the current user's AI configuration (API key is masked).
func (h *AIHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, err := h.aiService.GetConfig(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, publicConfig(cfg, h.aiService.IsConfigured(r.Context(), userId)))
}

// UpdateConfig updates the user's AI configuration.
func (h *AIHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var cfg ai.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.aiService.UpdateConfig(r.Context(), userId, cfg); err != nil {
		log.Error(r.Context(), "Failed to update AI config", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	encodeJSON(w, publicConfig(cfg, true))
}

// Status returns whether AI is configured for the current user.
func (h *AIHandler) Status(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, _ := h.aiService.GetConfig(r.Context(), userId)
	encodeJSON(w, map[string]interface{}{
		"configured": h.aiService.IsConfigured(r.Context(), userId),
		"provider":   cfg.Provider,
	})
}

// FetchLyrics kicks off a background lyrics fetch for the given media file.
// The body must be {"mediaFileId": "<id>"}. Returns immediately; the UI polls
// /api/ai/lyrics/status for progress.
func (h *AIHandler) FetchLyrics(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		MediaFileID string `json:"mediaFileId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.MediaFileID == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	if err := h.aiService.FetchLyrics(r.Context(), userId, req.MediaFileID); err != nil {
		log.Error(r.Context(), "AI lyrics fetch failed to start", "error", err, "mediaFileId", req.MediaFileID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"ok": true, "mediaFileId": req.MediaFileID})
}

// LyricsStatus returns the current status of a background lyrics fetch.
// Query parameter: mediaFileId.
func (h *AIHandler) LyricsStatus(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mediaFileId := r.URL.Query().Get("mediaFileId")
	if mediaFileId == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	status, err := h.aiService.GetLyricsStatus(r.Context(), userId, mediaFileId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, status)
}

// GetStoredTranslate returns a previously persisted translation for a track.
// Query params: mediaFileId, lang (defaults to "ru").
func (h *AIHandler) GetStoredTranslate(w http.ResponseWriter, r *http.Request) {
	_, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mediaFileId := r.URL.Query().Get("mediaFileId")
	if mediaFileId == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "ru"
	}
	text, found, err := h.aiService.GetStoredResult(r.Context(), mediaFileId, ".ai.translate."+lang+".txt")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"text": text, "found": found})
}

// GetStoredDecode returns a previously persisted decode for a track.
// Query param: mediaFileId.
func (h *AIHandler) GetStoredDecode(w http.ResponseWriter, r *http.Request) {
	_, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mediaFileId := r.URL.Query().Get("mediaFileId")
	if mediaFileId == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	text, found, err := h.aiService.GetStoredResult(r.Context(), mediaFileId, ".ai.decode.md")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"text": text, "found": found})
}

// GetStoredAnalyze returns a previously persisted analysis for a track.
// Query param: mediaFileId.
func (h *AIHandler) GetStoredAnalyze(w http.ResponseWriter, r *http.Request) {
	_, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mediaFileId := r.URL.Query().Get("mediaFileId")
	if mediaFileId == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	text, found, err := h.aiService.GetStoredResult(r.Context(), mediaFileId, ".ai.analyze.txt")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"text": text, "found": found})
}

// MissingLyrics lists the media files of an artist/album that do not yet have a
// synced lyrics sidecar (.lrc). Used by external orchestrators (e.g. Mixarr) to
// decide which tracks to enrich. Query params (one of): artistId, albumId.
func (h *AIHandler) MissingLyrics(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	artistId := r.URL.Query().Get("artistId")
	albumId := r.URL.Query().Get("albumId")
	if artistId == "" && albumId == "" {
		http.Error(w, "artistId or albumId is required", http.StatusBadRequest)
		return
	}
	items, err := h.aiService.MissingLyrics(r.Context(), userId, artistId, albumId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"items": items})
}

// MissingDecode lists the media files of an artist/album that do not yet have a
// meaning-decode sidecar (.ai.decode.md). Query params (one of): artistId, albumId.
func (h *AIHandler) MissingDecode(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	artistId := r.URL.Query().Get("artistId")
	albumId := r.URL.Query().Get("albumId")
	if artistId == "" && albumId == "" {
		http.Error(w, "artistId or albumId is required", http.StatusBadRequest)
		return
	}
	items, err := h.aiService.MissingDecode(r.Context(), userId, artistId, albumId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encodeJSON(w, map[string]interface{}{"items": items})
}

// FetchOriginalLyrics fetches ORIGINAL synced lyrics from LRCLIB only (Phase A).
// Synchronous, no translation. Body: {"mediaFileId": "<id>"}.
func (h *AIHandler) FetchOriginalLyrics(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		MediaFileID string `json:"mediaFileId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.MediaFileID == "" {
		http.Error(w, "mediaFileId is required", http.StatusBadRequest)
		return
	}
	found, err := h.aiService.FetchOriginalLyrics(r.Context(), userId, req.MediaFileID)
	if err != nil {
		writeAIError(w, err)
		return
	}
	encodeJSON(w, map[string]interface{}{"ok": true, "found": found, "mediaFileId": req.MediaFileID})
}

// TranslateBatch translates up to N songs in one LLM call (Phase B).
// Body: {"items": [{mediaFileId, title, artist, lyrics}], "toLang": "ru"}.
func (h *AIHandler) TranslateBatch(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ai.TranslateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "items are required", http.StatusBadRequest)
		return
	}
	results, err := h.aiService.TranslateBatch(r.Context(), userId, &req)
	if err != nil {
		writeAIError(w, err)
		return
	}
	encodeJSON(w, map[string]interface{}{"results": results})
}

// DecodeBatch generates meaning-decode for up to N songs in one LLM call (Phase C).
// Body: {"items": [{mediaFileId, title, artist, album, lyrics}]}.
func (h *AIHandler) DecodeBatch(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ai.DecodeBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "items are required", http.StatusBadRequest)
		return
	}
	results, err := h.aiService.DecodeBatch(r.Context(), userId, &req)
	if err != nil {
		writeAIError(w, err)
		return
	}
	encodeJSON(w, map[string]interface{}{"results": results})
}

// ── Mixarr enrichment proxy endpoints ────────────────────────────────────────
// These let the navidrome Activity panel show/cancel a running Mixarr task.

var mixarrHTTPClient = &http.Client{Timeout: 15 * time.Second}

// EnrichStatus proxies a GET to Mixarr's enrich/status endpoint.
func (h *AIHandler) EnrichStatus(w http.ResponseWriter, r *http.Request) {
	mixarrURL := mixarrBaseURL(r)
	resp, err := mixarrHTTPClient.Get(mixarrURL + "/api/navidrome/enrich/status")
	if err != nil {
		encodeJSON(w, map[string]interface{}{"status": "idle"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

// EnrichCancel proxies a POST to Mixarr's enrich/cancel endpoint.
func (h *AIHandler) EnrichCancel(w http.ResponseWriter, r *http.Request) {
	mixarrURL := mixarrBaseURL(r)
	resp, err := mixarrHTTPClient.Post(mixarrURL+"/api/navidrome/enrich/cancel", "application/json", nil)
	if err != nil {
		http.Error(w, "Failed to reach Mixarr", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func mixarrBaseURL(r *http.Request) string {
	if q := r.URL.Query().Get("mixarrUrl"); q != "" {
		return strings.TrimRight(q, "/")
	}
	return "https://192.168.1.95:3443"
}

// ────────────────────────────────────────────────────────────────────────────

// Test verifies the AI provider configuration by sending a minimal probe
// request. It accepts an optional JSON body {provider, apiKey, apiEndpoint,
// model} so a key can be tested before being saved. With no body or an empty
// provider, the user's currently stored configuration is tested instead.
func (h *AIHandler) Test(w http.ResponseWriter, r *http.Request) {
	userId, ok := userFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg ai.Config
	// Body is optional; an empty/invalid body just means "test the saved config".
	_ = json.NewDecoder(r.Body).Decode(&cfg)

	if err := h.aiService.TestConfig(r.Context(), userId, cfg); err != nil {
		log.Warn(r.Context(), "AI connection test failed", "error", err)
		encodeJSON(w, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	encodeJSON(w, map[string]interface{}{"ok": true})
}

// publicConfig masks the API key before returning it to the client.
func publicConfig(cfg ai.Config, configured bool) map[string]interface{} {
	keyMasked := ""
	if cfg.APIKey != "" {
		keyMasked = "********"
	}
	return map[string]interface{}{
		"provider":        cfg.Provider,
		"apiKey":          keyMasked,
		"apiEndpoint":     cfg.APIEndpoint,
		"model":           cfg.Model,
		"defaultLanguage": cfg.DefaultLang,
		"configured":      configured,
	}
}

// encodeJSON writes a JSON response.
func encodeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeAIError maps a provider error to an HTTP response. Quota / rate-limit /
// region / transient-overload failures return 503 (Service Unavailable) with a
// retryable flag so the UI can show a friendly "try again later" message and
// orchestrators (Mixarr) can back off and retry. Everything else stays a 500.
func writeAIError(w http.ResponseWriter, err error) {
	msg := err.Error()
	low := strings.ToLower(msg)
	retryable := strings.Contains(low, "quota") ||
		strings.Contains(low, "rate limit") ||
		strings.Contains(low, "resource_exhausted") ||
		strings.Contains(low, "429") ||
		strings.Contains(low, "location is not supported") ||
		strings.Contains(low, "high demand") ||
		strings.Contains(low, "overloaded") ||
		strings.Contains(low, "try again later") ||
		strings.Contains(low, "status unavailable")

	status := http.StatusInternalServerError
	if retryable {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":     msg,
		"retryable": retryable,
	})
}
