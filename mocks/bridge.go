package mocks

import "sync"

type BridgeCall struct {
	Method string
	Args   []any
}

type MockBridge struct {
	mu    sync.Mutex
	Calls []BridgeCall
}

func (m *MockBridge) Notify(method string, args ...any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, BridgeCall{Method: method, Args: args})
	return nil
}

func (m *MockBridge) GetCalls() []BridgeCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Calls
}
