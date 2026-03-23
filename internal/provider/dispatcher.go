package provider

import (
	"context"
	"errors"
	"sync/atomic"

	"nvim-engine/internal/engine/types"
)

type Dispatcher struct {
	providers []Provider
	current   uint64
}

func NewDispatcher(providers ...Provider) *Dispatcher {
	var active []Provider

	for _, p := range providers {
		if p.IsReady() {
			active = append(active, p)
		}
	}

	return &Dispatcher{
		providers: active,
	}
}

func (r *Dispatcher) getNext() Provider {
	if len(r.providers) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&r.current, 1) - 1
	return r.providers[idx%uint64(len(r.providers))]
}

func (r *Dispatcher) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	p := r.getNext()
	if p == nil {
		return "", errors.New("no active AI providers available")
	}
	return p.Generate(ctx, systemPrompt, userPrompt)
}

func (r *Dispatcher) GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	p := r.getNext()
	if p == nil {
		return "", errors.New("no active AI providers available")
	}

	return p.GenerateChat(ctx, systemPrompt, messages)
}

func (r *Dispatcher) IsReady() bool {
	return len(r.providers) > 0
}
