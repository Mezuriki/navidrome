package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/navidrome/navidrome/core/ai"
	"github.com/navidrome/navidrome/log"
)

type AIHandler struct {
	aiService *ai.Service
}

func NewAIHandler(aiService *ai.Service) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// encodeJSON writes a JSON response
func encodeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TranslateRequest handles translation requests
func (h *AIHandler) Translate(w http.ResponseWriter, r *http.Request) {
	var req ai.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.aiService.Translate(r.Context(), &req)
	if err != nil {
		log.Error(r.Context(), "Translation failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encodeJSON(w, resp)
}

// AnalyzeRequest handles track analysis requests
func (h *AIHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	var req ai.AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.aiService.Analyze(r.Context(), &req)
	if err != nil {
		log.Error(r.Context(), "Analysis failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encodeJSON(w, resp)
}

// DecodeRequest handles track decoding requests
func (h *AIHandler) Decode(w http.ResponseWriter, r *http.Request) {
	var req ai.DecodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.aiService.Decode(r.Context(), &req)
	if err != nil {
		log.Error(r.Context(), "Decode failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encodeJSON(w, resp)
}

// GetConfig returns the current AI configuration
func (h *AIHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.aiService.GetConfig()
	// Don't expose the API key in responses
	config.APIKey = ""
	encodeJSON(w, config)
}

// UpdateConfig updates the AI configuration
func (h *AIHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config ai.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.aiService.UpdateConfig(config); err != nil {
		log.Error(r.Context(), "Failed to update AI config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info(r.Context(), "AI config updated", "provider", config.Provider)

	// Return config without API key
	config.APIKey = ""
	encodeJSON(w, config)
}

// IsConfigured returns whether AI is properly configured
func (h *AIHandler) IsConfigured(w http.ResponseWriter, r *http.Request) {
	encodeJSON(w, map[string]interface{}{
		"configured": h.aiService.IsConfigured(),
		"provider":   h.aiService.GetConfig().Provider,
	})
}
