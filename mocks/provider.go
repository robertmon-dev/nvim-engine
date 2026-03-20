package mocks

import (
	"context"

	"nvim-engine/internal/engine/types"
)

type MockProvider struct {
	GenerateFunc     func(ctx context.Context, sys, user string) (string, error)
	GenerateChatFunc func(ctx context.Context, sys string, messages []types.Message) (string, error)
	IsReadyFunc      func() bool
}

func (m *MockProvider) Generate(ctx context.Context, sys, user string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, sys, user)
	}
	return "mocked response", nil
}

func (m *MockProvider) GenerateChat(ctx context.Context, sys string, messages []types.Message) (string, error) {
	if m.GenerateChatFunc != nil {
		return m.GenerateChatFunc(ctx, sys, messages)
	}
	return "mocked chat response", nil
}

func (m *MockProvider) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}
