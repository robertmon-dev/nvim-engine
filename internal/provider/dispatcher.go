package provider

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider/p_error"
)

type Dispatcher struct {
	Providers []Provider
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
		Providers: active,
	}
}

func (r *Dispatcher) order() []Provider {
	n := len(r.Providers)
	if n == 0 {
		return nil
	}

	start := int((atomic.AddUint64(&r.current, 1) - 1) % uint64(n))
	rotated := make([]Provider, 0, n)
	for i := 0; i < n; i++ {
		rotated = append(rotated, r.Providers[(start+i)%n])
	}

	return rotated
}

func attempt(providers []Provider, call func(Provider) (string, error)) (string, error) {
	if len(providers) == 0 {
		return "", fmt.Errorf("%w: none are configured", p_error.ErrAllProvidersFailed)
	}

	var errs []error
	for _, p := range providers {
		res, err := call(p)
		if err == nil {
			return res, nil
		}
		errs = append(errs, err)

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			break
		}
	}

	return "", fmt.Errorf("%w: %w", p_error.ErrAllProvidersFailed, errors.Join(errs...))
}

func (r *Dispatcher) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return attempt(r.order(), func(p Provider) (string, error) {
		return p.Generate(ctx, systemPrompt, userPrompt)
	})
}

func (r *Dispatcher) GenerateChat(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	return attempt(r.order(), func(p Provider) (string, error) {
		return p.GenerateChat(ctx, systemPrompt, messages)
	})
}

func (r *Dispatcher) IsReady() bool {
	return len(r.Providers) > 0
}
