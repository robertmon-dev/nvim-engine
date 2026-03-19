package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"nvim-engine/internal/config"
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
	IsReady() bool
}

func InitFromConfig(cfg *config.Config) map[ID]Provider {
	return map[ID]Provider{
		Gemini: &GeminiProvider{
			APIKey: cfg.Providers.GeminiAPIKey,
			Model:  cfg.Providers.GeminiModel,
			URL:    cfg.Providers.GeminiURL,
		},
		Anthropic: &AnthropicProvider{
			APIKey: cfg.Providers.AnthropicAPIKey,
			Model:  cfg.Providers.AnthropicModel,
			URL:    cfg.Providers.AnthropicURL,
		},
		OpenAI: &OpenAIProvider{
			APIKey: cfg.Providers.OpenAIAPIKey,
			Model:  cfg.Providers.OpenAIModel,
			URL:    cfg.Providers.OpenAIURL,
		},
		Ollama: &OllamaProvider{
			Model: cfg.Providers.OllamaModel,
			URL:   cfg.Providers.OllamaURL,
		},
	}
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
