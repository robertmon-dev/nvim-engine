package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"nvim-engine/internal/provider/p_error"
)

type OpenAIProvider struct {
	APIKey string
	Model  string
	URL    string
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiPayload struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *OpenAIProvider) Generate(ctx context.Context, system, user string) (string, error) {
	if !o.IsReady() {
		return "", p_error.NewConfigError(string(OpenAI))
	}

	payload := openaiPayload{
		Model: o.Model,
		Messages: []openaiMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", o.URL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, OpenAI, req, func(res openaiResponse) string {
		if len(res.Choices) > 0 {
			return res.Choices[0].Message.Content
		}
		return ""
	})
}

func (o *OpenAIProvider) IsReady() bool {
	return o.APIKey != "" && o.URL != ""
}
