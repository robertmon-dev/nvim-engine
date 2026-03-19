package middleware

import (
	"nvim-engine/internal/engine/types"

	"github.com/rs/zerolog/log"
)

func WithRecovery(next types.TaskHandler) types.TaskHandler {
	return func(msg types.RPCNotification) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("Recovered from handler panic")
			}
		}()
		next(msg)
	}
}
