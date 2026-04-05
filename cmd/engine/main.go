package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"nvim-engine/internal/config"
	"nvim-engine/internal/engine"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider"
	"nvim-engine/internal/provider/p_error"

	"github.com/vmihailenco/msgpack/v5"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)

	log := logger.Get()
	log.Info().Msg("Materializing Bifröst...")

	cfg := config.Get()
	if pErr := cfg.Validate(); pErr != nil {
		log.Fatal().Msg(pErr.Friendly())
	}

	dispatcher, err := provider.InitFromConfig(cfg)
	if err != nil {
		if pErr, ok := err.(*p_error.ProviderError); ok {
			log.Fatal().Msg(pErr.Friendly())
		}

		log.Fatal().Err(err).Msg("Failed to initialize AI providers")
	}
	proc := engine.NewProcessor(cfg.Engine.Workers, cfg.Engine.Capacity, dispatcher)

	dec := msgpack.NewDecoder(os.Stdin)

	bridge := logger.NewNvimBridge(os.Stdout)
	logger.AttachBridge(bridge)

	ctrl := &engine.Controller{
		Proc:     proc,
		Bridge:   bridge,
		Handlers: make(map[engine.RPCMethod]types.TaskHandler),
	}
	ctrl.RegisterHandlers()

	log.Info().Msg("Bifröst is set and ready to receive MessagePack RPC")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go ctrl.Listen(dec, sigChan)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Shutdown signal received. Wrapping up...")

	proc.Shutdown(5 * time.Second)
}
