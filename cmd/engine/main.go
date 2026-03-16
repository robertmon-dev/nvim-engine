package main

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"nvim-engine/internal/config"
	"nvim-engine/internal/engine"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider"

	"github.com/vmihailenco/msgpack/v5"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)

	log := logger.Get()
	log.Info().Msg("Booting up the Go-Engine...")

	cfg := config.Get()

	providers := map[provider.ID]provider.Provider{
		provider.Gemini: &provider.GeminiProvider{
			APIKey: cfg.Providers.GeminiAPIKey,
			Model:  cfg.Providers.GeminiModel,
			URL:    cfg.Providers.GeminiURL,
		},
		provider.Anthropic: &provider.AnthropicProvider{
			APIKey: cfg.Providers.AnthropicAPIKey,
			Model:  cfg.Providers.AnthropicModel,
			URL:    cfg.Providers.AnthropicURL,
		},
		provider.OpenAI: &provider.OpenAIProvider{
			APIKey: cfg.Providers.OpenAIAPIKey,
			Model:  cfg.Providers.OpenAIModel,
			URL:    cfg.Providers.OpenAIURL,
		},
	}

	proc := engine.NewProcessor(cfg.Engine.Workers, cfg.Engine.Capacity, providers)

	dec := msgpack.NewDecoder(os.Stdin)
	enc := msgpack.NewEncoder(os.Stdout)

	bridge := logger.NewNvimBridge(enc)
	logger.AttachBridge(bridge)

	ctrl := &engine.Controller{
		Proc:   proc,
		Bridge: bridge,
	}

	log.Info().Msg("Go Engine is armored and ready to receive MessagePack RPC")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			var msg engine.RPCNotification
			err := dec.Decode(&msg)

			if err == io.EOF {
				log.Info().Msg("Neovim closed STDIN (EOF). Initiating shutdown...")
				sigChan <- syscall.SIGTERM
				break
			}

			if err != nil {
				log.Error().Err(err).Msg("Failed to decode MessagePack payload")
				continue
			}

			log.Debug().Str("method", msg.Method).Msg("Received RPC task")
			ctrl.Dispatch(msg)
		}
	}()

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Shutdown signal received. Wrapping up...")

	proc.Pool.StopAndWait()

	log.Info().Msg("All workers finished. Go-Engine gracefully shut down. See ya!")
}
