package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nvim-engine/internal/config"

	"github.com/rs/zerolog"
)

var (
	log  zerolog.Logger
	once sync.Once
)

const DefaultLogName = "nvim-engine.log"

func Get() *zerolog.Logger {
	once.Do(func() {
		cfg := config.Get()

		out, path := openLog(cfg.Logger.Path)

		level, err := zerolog.ParseLevel(cfg.Logger.Level)
		if err != nil {
			level = zerolog.DebugLevel
		}
		zerolog.SetGlobalLevel(level)

		log = zerolog.New(zerolog.ConsoleWriter{
			Out:        out,
			TimeFormat: time.TimeOnly,
			NoColor:    true,
		}).With().
			Timestamp().
			Caller().
			Logger()

		log.Info().
			Str("path", path).
			Str("level", level.String()).
			Msg("Zerolog initialized from config")
	})

	return &log
}

func openLog(configured string) (io.Writer, string) {
	candidates := []string{configured, filepath.Join(os.TempDir(), DefaultLogName)}

	var failures []string
	for _, path := range candidates {
		if path == "" {
			continue
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err == nil {
			return file, path
		}
		failures = append(failures, err.Error())
	}

	for _, msg := range failures {
		fmt.Fprintln(os.Stderr, "nvim-engine: cannot open log file:", msg)
	}
	fmt.Fprintln(os.Stderr, "nvim-engine: falling back to stderr for logs")

	return os.Stderr, "stderr"
}
