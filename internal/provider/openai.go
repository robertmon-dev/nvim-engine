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

type OpenAIProvider struct {
	APIKeys []string
	Model   string
	URL     string
	current uint64
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

func (o *OpenAIProvider) IsReady() bool {
	return len(o.APIKeys) > 0 && o.URL != ""
}

func (p *OpenAIProvider) getNextKey() string {
	if len(p.APIKeys) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&p.current, 1) - 1
	return p.APIKeys[idx%uint64(len(p.APIKeys))]
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
	key := o.getNextKey()
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, OpenAI, req, func(res openaiResponse) string {
		if len(res.Choices) > 0 {
			return res.Choices[0].Message.Content
		}
		return ""
	})
}
