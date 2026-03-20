package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"nvim-engine/internal/engine/types"
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

	return o.doRequest(ctx, payload)
}

func (o *OpenAIProvider) GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	if !o.IsReady() {
		return "", p_error.NewConfigError(string(OpenAI))
	}

	openaiMessages := make([]openaiMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		openaiMessages = append(openaiMessages, openaiMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		openaiMessages = append(openaiMessages, openaiMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	payload := openaiPayload{
		Model:    o.Model,
		Messages: openaiMessages,
	}

	return o.doRequest(ctx, payload)
}

func (o *OpenAIProvider) doRequest(ctx context.Context, payload openaiPayload) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

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
