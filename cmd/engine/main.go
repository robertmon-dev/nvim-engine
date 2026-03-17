package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

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
	if err := cfg.Validate(); err != nil {
		log.Warn().Msg(err.Error())
	}

	providers := provider.InitFromConfig(cfg)
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

	go ctrl.Listen(dec, sigChan)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Shutdown signal received. Wrapping up...")

	proc.Shutdown(5 * time.Second)
}
