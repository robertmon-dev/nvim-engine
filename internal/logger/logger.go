package logger

import (
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

		file, err := os.OpenFile(cfg.Logger.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)

		var output os.File
		if err != nil {
			output = *os.Stderr
		} else {
			output = *file
		}

		consoleWriter := zerolog.ConsoleWriter{
			Out:        &output,
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
