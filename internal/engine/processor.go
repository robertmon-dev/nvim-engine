package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider"

	"github.com/alitto/pond"
)

const requestTimeout = 60 * time.Second

const optionSeparator = "===OPTION==="

type Processor struct {
	Pool       *pond.WorkerPool
	Dispatcher provider.Provider
	MaxRetries int
}

type ProcessorInterface interface {
	Process(task types.Task) ([]string, error)
	ProcessChat(task types.ChatTask) (string, error)
	GetPool() *pond.WorkerPool
	Submit(f func())
}

func (p *Processor) GetPool() *pond.WorkerPool {
	return p.Pool
}

func (p *Processor) Submit(f func()) {
	p.Pool.Submit(f)
}

func NewProcessor(workers, capacity, maxRetries int, dispatcher provider.Provider) *Processor {
	return &Processor{
		Pool:       pond.New(workers, capacity),
		Dispatcher: dispatcher,
		MaxRetries: max(1, maxRetries),
	}
}

func retry[T any](p *Processor, what string, call func(context.Context) (T, error), valid func(T) error) (T, error) {
	var zero T

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	if !p.Dispatcher.IsReady() {
		return zero, errors.New("no API keys or local providers configured in dispatcher")
	}

	attempts := max(1, p.MaxRetries)
	errs := make([]error, 0, attempts)

	for i := 0; i < attempts; i++ {
		res, err := call(ctx)
		if err == nil {
			if err = valid(res); err == nil {
				return res, nil
			}
		}

		errs = append(errs, fmt.Errorf("[attempt %d/%d]: %w", i+1, attempts, err))

		if ctx.Err() != nil {
			break
		}
	}

	return zero, fmt.Errorf("%s failed after %d attempt(s):\n%w", what, len(errs), errors.Join(errs...))
}

func (p *Processor) Process(task types.Task) ([]string, error) {
	return retry(
		p, "commit generation",
		func(ctx context.Context) ([]string, error) {
			raw, err := p.Dispatcher.Generate(ctx, SystemPrompt, task.Payload)
			if err != nil {
				return nil, err
			}

			return p.parseOptions(raw), nil
		},
		func(options []string) error {
			if len(options) == 0 {
				return errors.New("returned empty response or invalid options")
			}

			return nil
		},
	)
}

func (p *Processor) ProcessChat(task types.ChatTask) (string, error) {
	messages := make([]types.Message, 0, len(task.Messages)+1)
	messages = append(messages, task.Messages...)
	messages = append(messages, types.Message{
		Role:    "user",
		Content: task.Prompt,
	})

	return retry(
		p, "chat",
		func(ctx context.Context) (string, error) {
			return p.Dispatcher.GenerateChat(ctx, ChatSystemPrompt, messages)
		},
		func(res string) error {
			if strings.TrimSpace(res) == "" {
				return errors.New("returned empty response")
			}

			return nil
		},
	)
}

func (p *Processor) parseOptions(raw string) []string {
	if !strings.Contains(raw, optionSeparator) {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			return []string{trimmed}
		}

		return nil
	}

	var options []string
	for _, part := range strings.Split(raw, optionSeparator) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			options = append(options, trimmed)
		}
	}

	return options
}

func (p *Processor) Shutdown(timeout time.Duration) {
	log := logger.Get()
	done := make(chan struct{})

	go func() {
		p.Pool.StopAndWait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All workers finished. Bifröst is not passable. See ya!")
	case <-time.After(timeout):
		log.Warn().Msgf("Timeout waiting for workers (%s). Forcing shutdown to prevent zombie process!", timeout)
	}
}
