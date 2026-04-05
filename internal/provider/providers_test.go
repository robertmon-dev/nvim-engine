package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"nvim-engine/internal/config"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider/p_error"
)

func TestAnthropicProvider_Generate(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		responseBody    any
		expectedResult  string
		expectedErrCode p_error.ErrorCode
	}{
		{
			name:   "Success response",
			status: http.StatusOK,
			responseBody: anthropicResponse{
				Content: []struct {
					Text string `json:"text"`
				}{{Text: "hello from claude"}},
			},
			expectedResult: "hello from claude",
		},
		{
			name:   "Overloaded error (529)",
			status: 529,
			responseBody: map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "overloaded_error",
					"message": "Anthropic is overloaded",
				},
			},
			expectedErrCode: p_error.ErrInternal,
		},
		{
			name:   "Authentication error (401)",
			status: http.StatusUnauthorized,
			responseBody: map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "authentication_error",
					"message": "invalid x-api-key",
				},
			},
			expectedErrCode: p_error.ErrUnauthorized,
		},
		{
			name:   "Empty content error",
			status: http.StatusOK,
			responseBody: anthropicResponse{
				Content: nil,
			},
			expectedErrCode: p_error.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-api-key") != "test-anthropic-key" {
					t.Errorf("Missing or invalid x-api-key")
				}
				if r.Header.Get("anthropic-version") == "" {
					t.Errorf("Missing anthropic-version header")
				}

				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			prov := &AnthropicProvider{
				APIKeys: []string{"test-anthropic-key"},
				Model:   "claude-3-sonnet",
				URL:     server.URL,
			}

			result, err := prov.Generate(context.Background(), "sys", "user")

			if tt.expectedErrCode != "" {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				pErr, ok := err.(*p_error.ProviderError)
				if !ok {
					t.Fatalf("Expected *p_error.ProviderError, got %T", err)
				}
				if pErr.Code != tt.expectedErrCode {
					t.Errorf("Expected code %v, got %v", tt.expectedErrCode, pErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected success, got error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

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
		APIKeys: []string{"test-openai-key"},
		Model:   "gpt-4o",
		URL:     server.URL,
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
	tests := []struct {
		name            string
		status          int
		responseBody    any
		expectedResult  string
		expectedErrCode p_error.ErrorCode
	}{
		{
			name:   "Success response",
			status: http.StatusOK,
			responseBody: geminiResponse{
				Candidates: []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				}{
					{
						Content: struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						}{
							Parts: []struct {
								Text string `json:"text"`
							}{{Text: "hello from mock"}},
						},
					},
				},
			},
			expectedResult: "hello from mock",
		},
		{
			name:   "Rate limit error",
			status: http.StatusTooManyRequests,
			responseBody: map[string]any{
				"error": map[string]any{"message": "Quota exceeded", "status": "RESOURCE_EXHAUSTED"},
			},
			expectedErrCode: p_error.ErrRateLimit,
		},
		{
			name:            "Safety block (empty candidates)",
			status:          http.StatusOK,
			responseBody:    geminiResponse{Candidates: nil},
			expectedErrCode: p_error.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			prov := &GeminiProvider{
				APIKeys: []string{"test-key"},
				Model:   "test-model",
				URL:     server.URL,
			}

			result, err := prov.Generate(context.Background(), "sys", "user")

			if tt.expectedErrCode != "" {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				pErr, ok := err.(*p_error.ProviderError)
				if !ok {
					t.Fatalf("Expected *p_errors.ProviderError, got %T", err)
				}
				if pErr.Code != tt.expectedErrCode {
					t.Errorf("Expected error code %v, got %v", tt.expectedErrCode, pErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected success, got error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestOllamaProvider_Generate(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		responseBody    any
		expectedResult  string
		expectedErrCode p_error.ErrorCode
	}{
		{
			name:   "Success response",
			status: http.StatusOK,
			responseBody: ollamaResponse{
				Model: "llama3",
				Message: ollamaMessage{
					Role:    "assistant",
					Content: "hello from mock ollama",
				},
			},
			expectedResult: "hello from mock ollama",
		},
		{
			name:   "Model not found (404)",
			status: http.StatusNotFound,
			responseBody: ollamaResponse{
				Error: "model 'llama3' not found",
			},
			expectedErrCode: p_error.ErrInvalidRequest,
		},
		{
			name:   "Internal Server Error (500)",
			status: http.StatusInternalServerError,
			responseBody: ollamaResponse{
				Error: "server copped out",
			},
			expectedErrCode: p_error.ErrInternal,
		},
		{
			name:   "Error within 200 OK",
			status: http.StatusOK,
			responseBody: ollamaResponse{
				Error: "out of memory",
			},
			expectedErrCode: p_error.ErrUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			prov := &OllamaProvider{
				Model: "llama3",
				URL:   server.URL,
			}

			result, err := prov.Generate(context.Background(), "sys", "user")

			if tt.expectedErrCode != "" {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				pErr, ok := err.(*p_error.ProviderError)
				if !ok {
					t.Fatalf("Expected *p_errors.ProviderError, got %T", err)
				}
				if pErr.Code != tt.expectedErrCode {
					t.Errorf("Expected code %v, got %v", tt.expectedErrCode, pErr.Code)
				}
				if pErr.Provider != string(Ollama) {
					t.Errorf("Expected provider %s, got %s", Ollama, pErr.Provider)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected success, got error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestProvider_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prov := &OpenAIProvider{
		APIKeys: []string{"dummy-key-to-pass-isready"},
		URL:     server.URL,
	}

	_, err := prov.Generate(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("Expected an HTTP 500 error, but got nil")
	}
}

func TestAnthropicProvider_GenerateChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{
			Content: []struct {
				Text string `json:"text"`
			}{{Text: "chat response from claude"}},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &AnthropicProvider{
		APIKeys: []string{"test-key"},
		Model:   "claude-3",
		URL:     server.URL,
	}

	messages := []types.Message{
		{Role: "system", Content: "should be ignored in array"},
		{Role: "user", Content: "hello"},
	}

	result, err := prov.GenerateChat(context.Background(), "top level system", messages)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "chat response from claude" {
		t.Errorf("Expected 'chat response from claude', got: %v", result)
	}
}

func TestOpenAIProvider_GenerateChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{}
		choice := struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{}
		choice.Message.Content = "chat response from openai"
		resp.Choices = append(resp.Choices, choice)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &OpenAIProvider{
		APIKeys: []string{"test-openai-key"},
		Model:   "gpt-4o",
		URL:     server.URL,
	}

	messages := []types.Message{{Role: "user", Content: "hi"}}

	result, err := prov.GenerateChat(context.Background(), "system prompt", messages)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "chat response from openai" {
		t.Errorf("Expected 'chat response from openai', got: %v", result)
	}
}

func TestGeminiProvider_GenerateChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			}{
				{
					Content: struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					}{
						Parts: []struct {
							Text string `json:"text"`
						}{{Text: "chat response from gemini"}},
					},
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &GeminiProvider{
		APIKeys: []string{"test-key"},
		Model:   "gemini-1.5",
		URL:     server.URL,
	}

	messages := []types.Message{{Role: "user", Content: "hey"}}

	result, err := prov.GenerateChat(context.Background(), "sys prompt", messages)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "chat response from gemini" {
		t.Errorf("Expected 'chat response from gemini', got: %v", result)
	}
}

func TestOllamaProvider_GenerateChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{
			Model: "llama3",
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "chat response from ollama",
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	prov := &OllamaProvider{
		Model: "llama3",
		URL:   server.URL,
	}

	messages := []types.Message{{Role: "user", Content: "ping"}}

	result, err := prov.GenerateChat(context.Background(), "sys", messages)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "chat response from ollama" {
		t.Errorf("Expected 'chat response from ollama', got: %v", result)
	}
}

func TestInitFromConfig_Order(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Order: []string{"OLLAMA", "OpEnAi", "fake_provider"},

			OllamaModel: "llama3",
			OllamaURL:   "http://localhost:11434",

			OpenAIAPIKeys: []string{"test-key"},
			OpenAIModel:   "gpt-4o",
			OpenAIURL:     "https://api.openai.com",

			GeminiAPIKeys: []string{"test-key"},
			GeminiModel:   "gemini-1.5",
			GeminiURL:     "https://google.com",

			AnthropicModel: "claude-3",
			AnthropicURL:   "https://anthropic.com",
		},
	}

	dispatcher, _ := InitFromConfig(cfg)

	activeProviders := dispatcher.Providers

	if len(activeProviders) != 3 {
		t.Fatalf("Expected exactly 3 active providers, got %d", len(activeProviders))
	}

	expectedOrder := []string{"*provider.OllamaProvider", "*provider.OpenAIProvider", "*provider.GeminiProvider"}

	for i, p := range activeProviders {
		actualType := fmt.Sprintf("%T", p)
		if actualType != expectedOrder[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], actualType)
		}
	}
}
