package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"nvim-engine/internal/provider/p_error"
)

type OllamaProvider struct {
	Model string
	URL   string
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

func (p *OllamaProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !p.IsReady() {
		return "", p_error.NewConfigError(string(Ollama))
	}

	reqBody := ollamaRequest{
		Model: p.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, Ollama, req, func(res ollamaResponse) string {
		if res.Error != "" {
			return ""
		}

		return res.Message.Content
	})
}

func (p *OllamaProvider) IsReady() bool {
	return p.Model != "" && p.URL != ""
}
