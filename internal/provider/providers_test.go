package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-openai-key" {
			t.Errorf("Expected valid Authorization header, got: %v", r.Header.Get("Authorization"))
		}

		resp := openaiResponse{}
		choice := struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{}
		choice.Message.Content = "hello from mock openai"
		resp.Choices = append(resp.Choices, choice)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &OpenAIProvider{
		APIKey: "test-openai-key",
		Model:  "gpt-4o",
		URL:    server.URL,
	}

	result, err := prov.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "hello from mock openai" {
		t.Errorf("Expected 'hello from mock openai', got: %v", result)
	}
}

func TestGeminiProvider_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{}
		candidate := struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{}
		candidate.Content.Parts = append(candidate.Content.Parts, struct {
			Text string `json:"text"`
		}{Text: "hello from mock gemini"})
		resp.Candidates = append(resp.Candidates, candidate)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &GeminiProvider{
		APIKey: "test-gemini-key",
		Model:  "gemini-test",
		URL:    server.URL,
	}

	result, err := prov.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "hello from mock gemini" {
		t.Errorf("Expected 'hello from mock gemini', got: %v", result)
	}
}

func TestAnthropicProvider_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-anthropic-key" {
			t.Errorf("Expected valid x-api-key header, got: %v", r.Header.Get("x-api-key"))
		}

		resp := anthropicResponse{}
		content := struct {
			Text string `json:"text"`
		}{Text: "hello from mock claude"}
		resp.Content = append(resp.Content, content)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &AnthropicProvider{
		APIKey: "test-anthropic-key",
		Model:  "claude-test",
		URL:    server.URL,
	}

	result, err := prov.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "hello from mock claude" {
		t.Errorf("Expected 'hello from mock claude', got: %v", result)
	}
}

func TestProvider_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prov := &OpenAIProvider{
		URL: server.URL,
	}

	_, err := prov.Generate(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("Expected an HTTP 500 error, but got nil")
	}
}
