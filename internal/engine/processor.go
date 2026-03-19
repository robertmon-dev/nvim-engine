package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider"

	"github.com/alitto/pond"
)

type Processor struct {
	Pool      *pond.WorkerPool
	Providers map[provider.ID]provider.Provider
	Order     []provider.ID
}

func NewProcessor(workers, capacity int, providers map[provider.ID]provider.Provider) *Processor {
	return &Processor{
		Pool:      pond.New(workers, capacity),
		Providers: providers,
		Order: []provider.ID{
			provider.Ollama,
			provider.Gemini,
			provider.Anthropic,
			provider.OpenAI,
		},
	}
}

func (p *Processor) Process(task Task) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var errs []error
	var attemptedCount int

	for _, id := range p.Order {
		prov, ok := p.Providers[id]
		if !ok || !prov.IsReady() {
			continue
		}

		attemptedCount++
		rawText, err := prov.Generate(ctx, SystemPrompt, task.Payload)

		if err == nil {
			options := parseOptions(rawText)
			if len(options) > 0 {
				return options, nil
			}

			err = fmt.Errorf("returned empty response")
		}

		errs = append(errs, fmt.Errorf("[%s failed]: %w", id, err))
	}

	if attemptedCount == 0 {
		return nil, fmt.Errorf("no API keys or local providers configured")
	}

	return nil, fmt.Errorf("all attempted providers failed:\n%w", errors.Join(errs...))
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
		log.Info().Msg("All workers finished. Go-Engine gracefully shut down. See ya!")
	case <-time.After(timeout):
		log.Warn().Msgf("Timeout waiting for workers (%s). Forcing shutdown to prevent zombie process!", timeout)
	}
}
