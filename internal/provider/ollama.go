package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
		return "", errors.New("ollama provider is not properly configured")
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
		return "", fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sendRequest[ollamaResponse](req)
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf("ollama api error: %s", resp.Error)
	}

	return resp.Message.Content, nil
}

func (p *OllamaProvider) IsReady() bool {
	return p.Model != "" && p.URL != ""
}
