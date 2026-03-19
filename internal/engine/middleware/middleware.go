package middleware

import "nvim-engine/internal/engine/types"

type Middleware func(types.TaskHandler) types.TaskHandler

func Chain(h types.TaskHandler, mws ...Middleware) types.TaskHandler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}

	return h
}
