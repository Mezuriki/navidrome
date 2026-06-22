package ai

import (
	"context"
	"fmt"
	"sync"
)

// Service manages AI providers and configuration
type Service struct {
	mu        sync.RWMutex
	providers map[string]LLMProvider
	config    Config
}

// NewService creates a new AI service
func NewService() *Service {
	return &Service{
		providers: make(map[string]LLMProvider),
		config: Config{
			Provider:    "openai",
			DefaultLang: "en",
		},
	}
}

// UpdateConfig updates the AI service configuration
func (s *Service) UpdateConfig(config Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	// Clear existing providers
	for k := range s.providers {
		delete(s.providers, k)
	}

	// Initialize the configured provider
	provider, err := s.createProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	s.providers[config.Provider] = provider
	return nil
}

// GetConfig returns the current configuration
func (s *Service) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// Translate translates text using the configured provider
func (s *Service) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResponse, error) {
	provider, err := s.getProvider()
	if err != nil {
		return nil, err
	}

	return provider.Translate(ctx, req)
}

// Analyze analyzes a track using the configured provider
func (s *Service) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	provider, err := s.getProvider()
	if err != nil {
		return nil, err
	}

	return provider.Analyze(ctx, req)
}

// Decode analyzes the meaning of a song using the configured provider
func (s *Service) Decode(ctx context.Context, req *DecodeRequest) (*DecodeResponse, error) {
	provider, err := s.getProvider()
	if err != nil {
		return nil, err
	}

	return provider.Decode(ctx, req)
}

// getProvider returns the active provider
func (s *Service) getProvider() (LLMProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config.Provider == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	provider, exists := s.providers[s.config.Provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not initialized", s.config.Provider)
	}

	return provider, nil
}

// createProvider creates a provider instance based on config
func (s *Service) createProvider(config Config) (LLMProvider, error) {
	switch config.Provider {
	case "openai", "localai", "openrouter":
		return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
	case "ollama":
		return NewOllamaProvider(config.APIEndpoint, config.Model)
	case "anthropic":
		// For now, use OpenAI-compatible interface
		return NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// IsConfigured returns true if an AI provider is properly configured
func (s *Service) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.Provider != "" && s.config.APIKey != ""
}

// GetSupportedProviders returns a list of supported provider names
func GetSupportedProviders() []string {
	return []string{"openai", "anthropic", "ollama", "localai", "openrouter"}
}
