package mocks

import (
	"context"

	"nvim-engine/internal/engine/types"
)

type MockDispatcher struct {
	GenerateFunc     func(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	GenerateChatFunc func(ctx context.Context, systemPrompt string, messages []types.Message) (string, error)
	IsReadyFunc      func() bool
}

func (m *MockDispatcher) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, systemPrompt, userPrompt)
	}
	return "", nil
}

func (m *MockDispatcher) GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	if m.GenerateChatFunc != nil {
		return m.GenerateChatFunc(ctx, systemPrompt, messages)
	}
	return "", nil
}

func (m *MockDispatcher) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}
