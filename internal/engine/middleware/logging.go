package middleware

import (
	"time"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
)

func WithLogging(next types.TaskHandler) types.TaskHandler {
	return func(msg types.RPCNotification) {
		start := time.Now()
		next(msg)
		logger.Get().Debug().Dur("duration", time.Since(start)).Msg("RPC call")
	}
}
