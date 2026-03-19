package logger

import (
	"io"
	"os"
	"sync"
	"time"

	"nvim-engine/internal/config"

	"github.com/rs/zerolog"
)

var (
	log  zerolog.Logger
	once sync.Once
)

func Get() *zerolog.Logger {
	once.Do(func() {
		cfg := config.Get()

		logPath := cfg.Logger.Path
		if logPath == "" {
			logPath = "/tmp/nvim-ai-engine.log"
		}

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)

		var out io.Writer
		if err != nil {
			out = io.Discard
		} else {
			out = file
		}

		consoleWriter := zerolog.ConsoleWriter{
			Out:        out,
			TimeFormat: time.TimeOnly,
			NoColor:    true,
		}

		level, err := zerolog.ParseLevel(cfg.Logger.Level)
		if err != nil {
			level = zerolog.DebugLevel
		}
		zerolog.SetGlobalLevel(level)

		log = zerolog.New(consoleWriter).With().
			Timestamp().
			Caller().
			Logger()

		log.Info().
			Str("path", cfg.Logger.Path).
			Str("level", level.String()).
			Msg("Zerolog initialized from config")
	})

	return &log
}
