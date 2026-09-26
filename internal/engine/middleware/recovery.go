package middleware

import (
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
)

func WithRecovery(next types.TaskHandler) types.TaskHandler {
	return func(msg types.RPCNotification) {
		defer func() {
			if r := recover(); r != nil {
				logger.Get().Error().Interface("panic", r).
					Str("method", msg.Method).
					Msg("Recovered from handler panic")
			}
		}()

		next(msg)
	}
}
