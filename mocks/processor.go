package mocks

import (
	"nvim-engine/internal/engine/types"

	"github.com/alitto/pond"
)

type MockProcessor struct {
	Pool        *pond.WorkerPool
	ProcessFunc func(task types.Task) ([]string, error)
	ChatFunc    func(task types.ChatTask) (string, error)
}

func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		Pool: pond.New(1, 1),
	}
}

func (m *MockProcessor) Process(t types.Task) ([]string, error) {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(t)
	}
	return []string{"mocked"}, nil
}

func (m *MockProcessor) ProcessChat(t types.ChatTask) (string, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(t)
	}
	return "mocked chat", nil
}

func (m *MockProcessor) GetPool() *pond.WorkerPool {
	return m.Pool
}

func (m *MockProcessor) Submit(f func()) {
	m.Pool.Submit(f)
}
