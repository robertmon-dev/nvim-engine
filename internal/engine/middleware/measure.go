package middleware

import (
	"time"

	"nvim-engine/internal/engine/types"

	"github.com/rs/zerolog/log"
)

func WithMeasure(next types.TaskHandler) types.TaskHandler {
	return func(msg types.RPCNotification) {
		start := time.Now()

		next(msg)

		duration := time.Since(start)

		log.Info().
			Str("method", msg.Method).
			Dur("duration_ms", duration).
			Msg("RPC task processed")
	}
}
