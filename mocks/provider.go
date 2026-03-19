package mocks

import "context"

type MockProvider struct {
	GenerateFunc func(ctx context.Context, sys, user string) (string, error)
	IsReadyFunc  func() bool
}

func (m *MockProvider) Generate(ctx context.Context, sys, user string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, sys, user)
	}
	return "mocked response", nil
}

func (m *MockProvider) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}
