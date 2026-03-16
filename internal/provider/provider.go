package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return target, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return target, fmt.Errorf("api error: status code %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return target, err
	}

	return target, nil
}
