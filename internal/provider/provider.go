package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"nvim-engine/internal/config"
	"nvim-engine/internal/logger"
)

type ID string

const (
	Gemini    ID = "gemini"
	Anthropic ID = "anthropic"
	OpenAI    ID = "openai"
)

type Provider interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	IsReady() bool
}

func sendRequest[T any](req *http.Request) (T, error) {
	var target T
	log := logger.Get()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return target, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return target, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Trace().
		Int("status", resp.StatusCode).
		Str("body", string(bodyBytes)).
		Msg("Received API response")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return target, fmt.Errorf("api error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.Unmarshal(bodyBytes, &target); err != nil {
		return target, err
	}

	return target, nil
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
	}
}
