package provider

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider/p_error"
)

type stubProvider struct {
	name  string
	err   error
	ready bool

	mu    sync.Mutex
	calls int
}

func (s *stubProvider) record() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
}

func (s *stubProvider) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *stubProvider) Generate(context.Context, string, string) (string, error) {
	s.record()
	if s.err != nil {
		return "", s.err
	}
	return s.name, nil
}

func (s *stubProvider) GenerateChat(context.Context, string, []types.Message) (string, error) {
	s.record()
	if s.err != nil {
		return "", s.err
	}
	return s.name, nil
}

func (s *stubProvider) IsReady() bool { return s.ready }

func rateLimited(name string) error {
	return p_error.FromResponse(name, http.StatusTooManyRequests, []byte(`{"error":{"message":"slow down"}}`))
}

func TestDispatcherFailsOverOnRateLimit(t *testing.T) {
	first := &stubProvider{name: "first", err: rateLimited("first"), ready: true}
	second := &stubProvider{name: "second", ready: true}

	d := NewDispatcher(first, second)

	got, err := d.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("expected failover to succeed, got %v", err)
	}
	if got != "second" {
		t.Errorf("answer came from %q, want %q", got, "second")
	}
	if first.Calls() != 1 || second.Calls() != 1 {
		t.Errorf("calls: first=%d second=%d, want 1 and 1", first.Calls(), second.Calls())
	}
}

func TestDispatcherChatFailsOver(t *testing.T) {
	first := &stubProvider{name: "first", err: rateLimited("first"), ready: true}
	second := &stubProvider{name: "second", ready: true}

	got, err := NewDispatcher(first, second).GenerateChat(context.Background(), "sys", nil)
	if err != nil || got != "second" {
		t.Fatalf("GenerateChat() = %q, %v; want \"second\", nil", got, err)
	}
}

func TestDispatcherAllFailedWrapsSentinel(t *testing.T) {
	a := &stubProvider{name: "a", err: rateLimited("a"), ready: true}
	b := &stubProvider{name: "b", err: errors.New("boom"), ready: true}

	_, err := NewDispatcher(a, b).Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected an error when every provider fails")
	}
	if !errors.Is(err, p_error.ErrAllProvidersFailed) {
		t.Errorf("error %v does not match ErrAllProvidersFailed", err)
	}
	if a.Calls() != 1 || b.Calls() != 1 {
		t.Errorf("every provider should be tried once: a=%d b=%d", a.Calls(), b.Calls())
	}
}

func TestDispatcherSkipsNotReady(t *testing.T) {
	notReady := &stubProvider{name: "cold", ready: false}
	ready := &stubProvider{name: "warm", ready: true}

	d := NewDispatcher(notReady, ready)
	if len(d.Providers) != 1 {
		t.Fatalf("len(Providers) = %d, want 1", len(d.Providers))
	}

	if _, err := d.Generate(context.Background(), "sys", "user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notReady.Calls() != 0 {
		t.Error("a provider that is not ready must never be called")
	}
}

func TestDispatcherEmptyIsNotReady(t *testing.T) {
	d := NewDispatcher()
	if d.IsReady() {
		t.Error("an empty dispatcher must not report ready")
	}

	_, err := d.Generate(context.Background(), "sys", "user")
	if !errors.Is(err, p_error.ErrAllProvidersFailed) {
		t.Errorf("error = %v, want ErrAllProvidersFailed", err)
	}
}

func TestDispatcherStopsOnCancelledContext(t *testing.T) {
	first := &stubProvider{name: "first", err: context.Canceled, ready: true}
	second := &stubProvider{name: "second", ready: true}

	_, err := NewDispatcher(first, second).Generate(context.Background(), "sys", "user")
	if !errors.Is(err, p_error.ErrAllProvidersFailed) {
		t.Errorf("error = %v, want ErrAllProvidersFailed", err)
	}
	if second.Calls() != 0 {
		t.Error("the ring must stop at a cancelled context")
	}
}

func TestDispatcherRotatesStartingProvider(t *testing.T) {
	a := &stubProvider{name: "a", ready: true}
	b := &stubProvider{name: "b", ready: true}

	d := NewDispatcher(a, b)

	first, _ := d.Generate(context.Background(), "sys", "user")
	second, _ := d.Generate(context.Background(), "sys", "user")

	if first == second {
		t.Errorf("both requests went to %q; the starting provider should rotate", first)
	}
}
