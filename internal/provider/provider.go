package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

var httpClient = &http.Client{Timeout: 2 * time.Minute}

var defaultOrder = []ID{Gemini, Anthropic, OpenAI, Ollama}

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
		id := ID(strings.ToLower(strings.TrimSpace(pName)))
		if p, exists := allProviders[id]; exists && !added[id] {
			candidates = append(candidates, p)
			added[id] = true
		}
	}

	for _, id := range defaultOrder {
		if !added[id] {
			candidates = append(candidates, allProviders[id])
		}
	}

	dispatcher := NewDispatcher(candidates...)
	if !dispatcher.IsReady() {
		return nil, &p_error.ProviderError{
			Code:     p_error.ErrConfig,
			Provider: "System",
			Message:  p_error.FriendlyMessages[p_error.ErrConfig],
		}
	}

	return dispatcher, nil
}

func performRequest[T any](
	ctx context.Context,
	providerID ID,
	req *http.Request,
	extractor func(T) string,
) (string, error) {
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", redactTransportError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read %s response body: %w", providerID, err)
	}

	logger.Get().Trace().
		Str("provider", string(providerID)).
		Int("status", resp.StatusCode).
		Str("body", string(body)).
		Msg("Received API response")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", p_error.FromResponse(string(providerID), resp.StatusCode, body)
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode %s response: %w", providerID, err)
	}

	content := extractor(result)
	if content == "" {
		return "", p_error.FromResponse(string(providerID), resp.StatusCode, body)
	}

	return content, nil
}

func redactTransportError(err error) error {
	var uErr *url.Error
	if !errors.As(err, &uErr) {
		return err
	}

	host := "the provider"
	if parsed, perr := url.Parse(uErr.URL); perr == nil && parsed.Host != "" {
		host = parsed.Host
	}

	return fmt.Errorf("%s request to %s failed: %w", uErr.Op, host, uErr.Err)
}
