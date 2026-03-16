package mocks

import (
	"context"
)

type MockProvider struct {
	GenerateFunc func(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

func (m *MockProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, systemPrompt, userPrompt)
	}
	return "mocked response", nil
}

func (m *MockProvider) IsReady() bool {
	return true
}
