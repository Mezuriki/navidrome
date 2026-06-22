package nativeapi

import (
	"encoding/json"
	"net/http"

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
