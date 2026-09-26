package middleware

import (
	"time"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
)

func WithMeasure(next types.TaskHandler) types.TaskHandler {
	return func(msg types.RPCNotification) {
		start := time.Now()

		next(msg)

		logger.Get().Debug().
			Str("method", msg.Method).
			Dur("enqueue_duration", time.Since(start)).
			Msg("RPC notification accepted")
	}
}
