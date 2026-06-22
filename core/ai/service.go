package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// aiConfigKey is the UserProps key under which the AI config (as JSON) is stored per-user.
const aiConfigKey = "ai_config"

// Service manages AI providers and per-user configuration.
// Configuration is persisted in the user properties table (UserProps),
// so it survives server restarts and is specific to each user.
type Service struct {
	ds        model.DataStore
	mu        sync.Mutex
	providers map[string]LLMProvider // cached providers keyed by userId
}

// NewService creates a new AI service backed by the given DataStore.
func NewService(ds model.DataStore) *Service {
	return &Service{
		ds:        ds,
		providers: make(map[string]LLMProvider),
	}
}

// GetConfig returns the stored configuration for the user.
// Returns a zero-value Config (with empty Provider) if none is configured.
func (s *Service) GetConfig(ctx context.Context, userId string) (Config, error) {
	raw, err := s.ds.UserProps(ctx).DefaultGet(userId, aiConfigKey, "")
	if err != nil {
		return Config{}, err
	}
	if raw == "" {
		return Config{DefaultLang: "en"}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{DefaultLang: "en"}, err
	}
	return cfg, nil
}

// UpdateConfig persists the configuration for the user and refreshes the cached provider.
// The API key is stored in plaintext in the DB (same as other Navidrome secrets);
// it is never returned to the client by GetConfig (masked as ********).
// If the client sends "********" as the API key, the previously stored key is kept.
func (s *Service) UpdateConfig(ctx context.Context, userId string, config Config) error {
	// If the key is the mask placeholder, keep the previously stored key.
	if config.APIKey == "********" {
		if old, err := s.GetConfig(ctx, userId); err == nil {
			config.APIKey = old.APIKey
		}
	}

	// Validate the provider can be instantiated before saving.
	if _, err := s.createProvider(config); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := s.ds.UserProps(ctx).Put(userId, aiConfigKey, string(data)); err != nil {
		return err
	}

	// Invalidate the cached provider for this user.
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.providers, userId)
	log.Info(ctx, "AI configuration updated", "user", userId, "provider", config.Provider)
	return nil
}

// getProvider returns the provider for the user, building and caching it if needed.
func (s *Service) getProvider(ctx context.Context, userId string) (LLMProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.providers[userId]; ok && p != nil {
		return p, nil
	}

	cfg, err := s.GetConfig(ctx, userId)
	if err != nil {
		return nil, err
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	p, err := s.createProvider(cfg)
	if err != nil {
		return nil, err
	}
	s.providers[userId] = p
	return p, nil
}

// Translate translates text using the user's configured provider.
func (s *Service) Translate(ctx context.Context, userId string, req *TranslateRequest) (*TranslateResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	return p.Translate(ctx, req)
}

// Analyze analyzes a track using the user's configured provider.
func (s *Service) Analyze(ctx context.Context, userId string, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	return p.Analyze(ctx, req)
}

// Decode analyzes the meaning of a song using the user's configured provider.
func (s *Service) Decode(ctx context.Context, userId string, req *DecodeRequest) (*DecodeResponse, error) {
	p, err := s.getProvider(ctx, userId)
	if err != nil {
		return nil, err
	}
	return p.Decode(ctx, req)
}

// IsConfigured returns whether the user has a provider configured.
func (s *Service) IsConfigured(ctx context.Context, userId string) bool {
	cfg, err := s.GetConfig(ctx, userId)
	if err != nil {
		return false
	}
	return cfg.Provider != "" && (cfg.APIKey != "" || cfg.Provider == "ollama")
}

// createProvider creates a provider instance based on config.
func (s *Service) createProvider(config Config) (LLMProvider, error) {
	switch config.Provider {
	case "openai", "localai", "openrouter":
		return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
	case "ollama":
		return NewOllamaProvider(config.APIEndpoint, config.Model)
	case "anthropic":
		// Anthropic is OpenAI-compatible through its /v1 endpoint for this use case.
		return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// GetSupportedProviders returns a list of supported provider names.
func GetSupportedProviders() []string {
	return []string{"openai", "anthropic", "ollama", "localai", "openrouter"}
}
