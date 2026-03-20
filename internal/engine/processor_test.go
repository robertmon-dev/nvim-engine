package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider"
	"nvim-engine/mocks"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []string
	}{
		{
			name:     "Single commit without separator",
			raw:      "feat: add new button",
			expected: []string{"feat: add new button"},
		},
		{
			name:     "Three options with separator",
			raw:      "feat: option 1\n===OPTION===\nfix: option 2\n===OPTION===\nchore: option 3",
			expected: []string{"feat: option 1", "fix: option 2", "chore: option 3"},
		},
		{
			name:     "Empty options (bad AI)",
			raw:      "===OPTION===\n===OPTION===",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptions(tt.raw)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseOptions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessorEmptyResponseFailover(t *testing.T) {
	emptyMock := &mocks.MockProvider{
		GenerateFunc: func(ctx context.Context, system, user string) (string, error) {
			return " \n ===OPTION=== \n ", nil
		},
	}

	successMock := &mocks.MockProvider{
		GenerateFunc: func(ctx context.Context, system, user string) (string, error) {
			return "feat: valid option", nil
		},
	}

	providers := map[provider.ID]provider.Provider{
		provider.Gemini:    emptyMock,
		provider.Anthropic: successMock,
	}

	proc := NewProcessor(1, 10, providers)
	task := types.Task{ID: "test-3", Action: "commit", Payload: "diff"}

	result, err := proc.Process(task)
	if err != nil {
		t.Fatalf("Expected success via failover, got error: %v", err)
	}

	if len(result) == 0 || result[0] != "feat: valid option" {
		t.Errorf("Bad result. Got: %v", result)
	}
}

func TestProcessorAllFailed(t *testing.T) {
	failingMock := &mocks.MockProvider{
		GenerateFunc: func(ctx context.Context, system, user string) (string, error) {
			return "", errors.New("TOTAL OUTAGE")
		},
	}

	providers := map[provider.ID]provider.Provider{
		provider.Gemini:    failingMock,
		provider.Anthropic: failingMock,
		provider.OpenAI:    failingMock,
		provider.Ollama:    failingMock,
	}

	proc := NewProcessor(1, 10, providers)
	task := types.Task{ID: "test-2", Action: "commit", Payload: "diff..."}

	_, err := proc.Process(task)
	if err == nil {
		t.Fatal("Expected error (all models failed), but processor returned success!")
	}
}

func TestProcessorChatFailoverAndMessageBuilder(t *testing.T) {
	failingMock := &mocks.MockProvider{
		GenerateChatFunc: func(ctx context.Context, sys string, messages []types.Message) (string, error) {
			return "", errors.New("API OFFLINE")
		},
	}

	successMock := &mocks.MockProvider{
		GenerateChatFunc: func(ctx context.Context, sys string, messages []types.Message) (string, error) {
			if len(messages) != 2 {
				t.Errorf("Expected 2 messages in payload, got %d", len(messages))
			}
			if messages[1].Content != "new prompt" {
				t.Errorf("Expected last message content to be 'new prompt', got %s", messages[1].Content)
			}
			return "here is your code", nil
		},
	}

	providers := map[provider.ID]provider.Provider{
		provider.Ollama: failingMock,
		provider.Gemini: successMock,
	}

	proc := NewProcessor(1, 10, providers)
	task := types.ChatTask{
		ID:     "chat-test-1",
		Prompt: "new prompt",
		Messages: []types.Message{
			{Role: "user", Content: "old context"},
		},
	}

	result, err := proc.ProcessChat(task)
	if err != nil {
		t.Fatalf("Expected success via failover, got error: %v", err)
	}

	if result != "here is your code" {
		t.Errorf("Bad result. Got: %v", result)
	}
}

func TestProcessorChatAllFailed(t *testing.T) {
	failingMock := &mocks.MockProvider{
		GenerateChatFunc: func(ctx context.Context, sys string, messages []types.Message) (string, error) {
			return "", errors.New("TOTAL OUTAGE")
		},
	}

	providers := map[provider.ID]provider.Provider{
		provider.Ollama:    failingMock,
		provider.Gemini:    failingMock,
		provider.Anthropic: failingMock,
		provider.OpenAI:    failingMock,
	}

	proc := NewProcessor(1, 10, providers)
	task := types.ChatTask{
		ID:       "chat-test-2",
		Prompt:   "hello?",
		Messages: []types.Message{},
	}

	_, err := proc.ProcessChat(task)
	if err == nil {
		t.Fatal("Expected error (all models failed in chat), but processor returned success!")
	}

	if !strings.Contains(err.Error(), "chat failed for all providers") {
		t.Errorf("Expected specific error prefix, got: %v", err)
	}
}
