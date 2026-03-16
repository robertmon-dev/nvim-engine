package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	payload := map[string]interface{}{
		"model":      a.Model,
		"max_tokens": 1024,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": user},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", a.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	result, err := sendRequest[anthropicResponse](req)
	if err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("anthropic returned empty content")
	}

	return result.Content[0].Text, nil
}

func (o *AnthropicProvider) IsReady() bool {
	return o.APIKey != "" && o.URL != ""
}
