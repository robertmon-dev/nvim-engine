package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"nvim-engine/internal/provider/p_error"
)

type AnthropicProvider struct {
	APIKey string
	Model  string
	URL    string
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (a *AnthropicProvider) Generate(ctx context.Context, system, user string) (string, error) {
	if !a.IsReady() {
		return "", p_error.NewConfigError(string(Anthropic))
	}

	payload := map[string]interface{}{
		"model":      a.Model,
		"max_tokens": 1024,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": user},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, Anthropic, req, func(res anthropicResponse) string {
		if len(res.Content) > 0 {
			return res.Content[0].Text
		}
		return ""
	})
}

func (a *AnthropicProvider) IsReady() bool {
	return a.APIKey != "" && a.URL != ""
}
