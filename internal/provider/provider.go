package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nvim-engine/internal/config"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider/p_error"
)

type ID string

const (
	Gemini    ID = "gemini"
	Anthropic ID = "anthropic"
	OpenAI    ID = "openai"
	Ollama    ID = "ollama"
)

type Provider interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error)
	IsReady() bool
}

func InitFromConfig(cfg *config.Config) (*Dispatcher, error) {
	allProviders := map[ID]Provider{
		Gemini: &GeminiProvider{
			APIKeys: cfg.Providers.GeminiAPIKeys,
			Model:   cfg.Providers.GeminiModel,
			URL:     cfg.Providers.GeminiURL,
		},
		Anthropic: &AnthropicProvider{
			APIKeys: cfg.Providers.AnthropicAPIKeys,
			Model:   cfg.Providers.AnthropicModel,
			URL:     cfg.Providers.AnthropicURL,
		},
		OpenAI: &OpenAIProvider{
			APIKeys: cfg.Providers.OpenAIAPIKeys,
			Model:   cfg.Providers.OpenAIModel,
			URL:     cfg.Providers.OpenAIURL,
		},
		Ollama: &OllamaProvider{
			Model: cfg.Providers.OllamaModel,
			URL:   cfg.Providers.OllamaURL,
		},
	}

	var candidates []Provider
	added := make(map[ID]bool)

	for _, pName := range cfg.Providers.Order {
		id := ID(strings.ToLower(pName))
		if p, exists := allProviders[id]; exists {
			candidates = append(candidates, p)
			added[id] = true
		}
	}

	defaultOrder := []ID{Gemini, Anthropic, OpenAI, Ollama}
	for _, id := range defaultOrder {
		if !added[id] {
			candidates = append(candidates, allProviders[id])
		}
	}

	var activeProviders []Provider
	for _, p := range candidates {
		if p.IsReady() {
			activeProviders = append(activeProviders, p)
		}
	}

	if len(activeProviders) == 0 {
		return nil, &p_error.ProviderError{
			Code:     p_error.ErrConfig,
			Provider: "System",
			Message:  p_error.FriendlyMessages[p_error.ErrConfig],
		}
	}

	return NewDispatcher(activeProviders...), nil
}

func sendRequest[T any](req *http.Request) (T, []byte, int, error) {
	var target T
	log := logger.Get()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return target, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return target, nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Trace().
		Int("status", resp.StatusCode).
		Str("body", string(bodyBytes)).
		Msg("Received API response")

	return target, bodyBytes, resp.StatusCode, nil
}

func performRequest[T any](
	ctx context.Context,
	providerID ID,
	req *http.Request,
	extractor func(T) string,
) (string, error) {
	_, body, status, err := sendRequest[T](req)
	if err != nil {
		return "", err
	}

	if status < 200 || status >= 300 {
		return "", p_error.FromResponse(string(providerID), status, body)
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode %s response: %w", providerID, err)
	}

	content := extractor(result)
	if content == "" {
		return "", p_error.FromResponse(string(providerID), status, body)
	}

	return content, nil
}
