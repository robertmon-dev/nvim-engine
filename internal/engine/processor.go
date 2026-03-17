package engine

import (
	"context"
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
}

func NewProcessor(workers, capacity int, providers map[provider.ID]provider.Provider) *Processor {
	return &Processor{
		Pool:      pond.New(workers, capacity),
		Providers: providers,
	}
}

func (p *Processor) Process(task Task) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	order := []provider.ID{
		provider.Gemini,
		provider.Anthropic,
		provider.OpenAI,
	}

	var lastErr error
	var attemptedCount int

	for _, id := range order {
		prov, ok := p.Providers[id]
		if !ok {
			continue
		}

		if !prov.IsReady() {
			continue
		}

		attemptedCount++
		rawText, err := prov.Generate(ctx, SystemPrompt, task.Payload)

		if err == nil {
			options := parseOptions(rawText)
			if len(options) > 0 {
				return options, nil
			}
		}
		lastErr = err
	}

	if attemptedCount == 0 {
		return nil, fmt.Errorf("no API keys provided")
	}

	return nil, fmt.Errorf("all providers failed. last error: %v", lastErr)
}

func parseOptions(raw string) []string {
	if !strings.Contains(raw, "===OPTION===") {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			return []string{trimmed}
		}
		return nil
	}

	var options []string
	parts := strings.Split(raw, "===OPTION===")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
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
		log.Info().Msg("All workers finished. Go-Engine gracefully shut down. See ya!")
	case <-time.After(timeout):
		log.Warn().Msgf("Timeout waiting for workers (%s). Forcing shutdown to prevent zombie process!", timeout)
	}
}
