package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider/p_error"
)

type AnthropicProvider struct {
	APIKeys []string
	Model   string
	URL     string
	current uint64
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (a *AnthropicProvider) IsReady() bool {
	return len(a.APIKeys) > 0 && a.URL != ""
}

func (p *AnthropicProvider) getNextKey() string {
	if len(p.APIKeys) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&p.current, 1) - 1
	return p.APIKeys[idx%uint64(len(p.APIKeys))]
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

	return a.doRequest(ctx, payload)
}

func (a *AnthropicProvider) GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	if !a.IsReady() {
		return "", p_error.NewConfigError(string(Anthropic))
	}

	anthropicMessages := make([]map[string]string, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}

		anthropicMessages = append(anthropicMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	payload := map[string]any{
		"model":      a.Model,
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages":   anthropicMessages,
	}

	return a.doRequest(ctx, payload)
}

func (a *AnthropicProvider) doRequest(ctx context.Context, payload map[string]any) (string, error) {
	key := a.getNextKey()
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	return performRequest(Anthropic, req, func(res anthropicResponse) string {
		if len(res.Content) > 0 {
			return res.Content[0].Text
		}

		return ""
	})
}
