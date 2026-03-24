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

func NewProcessor(workers, capacity int, dispatcher provider.Provider) *Processor {
	return &Processor{
		Pool:       pond.New(workers, capacity),
		Dispatcher: dispatcher,
		MaxRetries: 1,
	}
}

func (p *Processor) Process(task types.Task) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if !p.Dispatcher.IsReady() {
		return nil, errors.New("no API keys or local providers configured in dispatcher")
	}

	var errs []error

	for i := 0; i < p.MaxRetries; i++ {
		rawText, err := p.Dispatcher.Generate(ctx, SystemPrompt, task.Payload)

		if err == nil {
			options := parseOptions(rawText)
			if len(options) > 0 {
				return options, nil
			}
			err = fmt.Errorf("returned empty response or invalid options")
		}

		errs = append(errs, fmt.Errorf("[Attempt %d failed]: %w", i+1, err))
	}

	return nil, fmt.Errorf("all %d attempted retries failed:\n%w", p.MaxRetries, errors.Join(errs...))
}

func (p *Processor) ProcessChat(task types.ChatTask) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if !p.Dispatcher.IsReady() {
		return "", errors.New("no API keys or local providers configured in dispatcher")
	}

	messages := make([]types.Message, 0, len(task.Messages)+1)
	messages = append(messages, task.Messages...)
	messages = append(messages, types.Message{
		Role:    "user",
		Content: task.Prompt,
	})

	var errs []error

	for i := 0; i < p.MaxRetries; i++ {
		res, err := p.Dispatcher.GenerateChat(ctx, ChatSystemPrompt, messages)

		if err == nil && strings.TrimSpace(res) != "" {
			return res, nil
		}

		if err == nil {
			err = errors.New("returned empty response")
		}

		errs = append(errs, fmt.Errorf("[Attempt %d failed]: %w", i+1, err))
	}

	return "", fmt.Errorf("chat failed after %d attempts:\n%w", p.MaxRetries, errors.Join(errs...))
}

func parseOptions(raw string) []string {
	if !strings.Contains(raw, "===OPTION===") {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			return []string{trimmed}
		}
		return nil
	}

	parts := strings.Split(raw, "===OPTION===")
	options := make([]string, 0, len(parts))

	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			options = append(options, trimmed)
		}
	}

	if len(options) == 0 {
		return nil
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
