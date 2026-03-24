package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"nvim-engine/internal/engine/types"
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
	attempts := 0

	mockDispatcher := &mocks.MockDispatcher{
		GenerateFunc: func(ctx context.Context, system, user string) (string, error) {
			attempts++
			if attempts == 1 {
				return " \n ===OPTION=== \n ", nil
			}
			return "feat: valid option", nil
		},
		IsReadyFunc: func() bool { return true },
	}

	proc := NewProcessor(1, 10, mockDispatcher)
	proc.MaxRetries = 2

	task := types.Task{ID: "test-3", Action: "commit", Payload: "diff"}

	result, err := proc.Process(task)
	if err != nil {
		t.Fatalf("Expected success via failover, got error: %v", err)
	}

	if len(result) == 0 || result[0] != "feat: valid option" {
		t.Errorf("Bad result. Got: %v", result)
	}

	if attempts != 2 {
		t.Errorf("Expected exactly 2 attempts, got %d", attempts)
	}
}

func TestProcessorAllFailed(t *testing.T) {
	attempts := 0
	failingDispatcher := &mocks.MockDispatcher{
		GenerateFunc: func(ctx context.Context, system, user string) (string, error) {
			attempts++
			return "", errors.New("TOTAL OUTAGE")
		},
		IsReadyFunc: func() bool { return true },
	}

	proc := NewProcessor(1, 10, failingDispatcher)
	proc.MaxRetries = 3

	task := types.Task{ID: "test-2", Action: "commit", Payload: "diff..."}

	_, err := proc.Process(task)
	if err == nil {
		t.Fatal("Expected error (all models failed), but processor returned success!")
	}

	if !strings.Contains(err.Error(), "all 3 attempted retries failed") {
		t.Errorf("Expected specific error prefix, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected exactly 3 attempts, got %d", attempts)
	}
}

func TestProcessorChatFailoverAndMessageBuilder(t *testing.T) {
	attempts := 0
	mockDispatcher := &mocks.MockDispatcher{
		GenerateChatFunc: func(ctx context.Context, sys string, messages []types.Message) (string, error) {
			attempts++
			if attempts == 1 {
				return "", errors.New("API OFFLINE")
			}

			if len(messages) != 2 {
				t.Errorf("Expected 2 messages in payload, got %d", len(messages))
			}
			if messages[1].Content != "new prompt" {
				t.Errorf("Expected last message content to be 'new prompt', got %s", messages[1].Content)
			}
			return "here is your code", nil
		},
		IsReadyFunc: func() bool { return true },
	}

	proc := NewProcessor(1, 10, mockDispatcher)
	proc.MaxRetries = 2 // <-- Wymuszamy 2 próby na test failovera

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
	failingDispatcher := &mocks.MockDispatcher{
		GenerateChatFunc: func(ctx context.Context, sys string, messages []types.Message) (string, error) {
			return "", errors.New("TOTAL OUTAGE")
		},
		IsReadyFunc: func() bool { return true },
	}

	proc := NewProcessor(1, 10, failingDispatcher)
	proc.MaxRetries = 2

	task := types.ChatTask{
		ID:       "chat-test-2",
		Prompt:   "hello?",
		Messages: []types.Message{},
	}

	_, err := proc.ProcessChat(task)
	if err == nil {
		t.Fatal("Expected error (all models failed in chat), but processor returned success!")
	}

	if !strings.Contains(err.Error(), "chat failed after 2 attempt") {
		t.Errorf("Expected specific error prefix, got: %v", err)
	}
}

func TestProcessorNotReadyFail(t *testing.T) {
	emptyDispatcher := &mocks.MockDispatcher{
		IsReadyFunc: func() bool { return false },
	}

	proc := NewProcessor(1, 10, emptyDispatcher)

	_, err := proc.Process(types.Task{Payload: "test"})
	if err == nil || !strings.Contains(err.Error(), "no API keys or local providers configured") {
		t.Errorf("Expected fast-fail error for unready dispatcher, got: %v", err)
	}

	_, errChat := proc.ProcessChat(types.ChatTask{Prompt: "test"})
	if errChat == nil || !strings.Contains(errChat.Error(), "no API keys or local providers configured") {
		t.Errorf("Expected fast-fail error for unready dispatcher in chat, got: %v", errChat)
	}
}
