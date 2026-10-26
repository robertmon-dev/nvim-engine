package engine

import (
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"nvim-engine/internal/engine/middleware"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider/p_error"

	"github.com/vmihailenco/msgpack/v5"
)

const maxDecodeErrors = 10

type Controller struct {
	Proc     ProcessorInterface
	Bridge   logger.NvimBridgeInterface
	Handlers map[RPCMethod]types.TaskHandler
}

func BindHandler[T types.Identifiable](
	c *Controller,
	infoPrefix string,
	logic func(T) ([]string, error),
) types.TaskHandler {
	return func(msg types.RPCNotification) {
		log := logger.Get()

		if len(msg.Args) == 0 {
			c.NotifyTele("Received a task with no payload", LogLevelError)
			return
		}

		var task T
		if err := msgpack.Unmarshal(msg.Args[0], &task); err != nil {
			c.NotifyTele("Decoding error: "+err.Error(), LogLevelError)
			return
		}

		c.Proc.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("task_id", task.GetID()).
						Msg("CRITICAL: recovered from panic while processing a task")
					c.NotifyTele("Engine panic while processing the task. See the log.", LogLevelError)
				}
			}()

			start := time.Now()
			id := task.GetID()

			c.NotifyTele(infoPrefix, LogLevelInfo)

			data, err := logic(task)

			res := types.Result{
				ID:   id,
				Data: data,
			}

			if err != nil {
				c.handleErrorTelemetry(err)
				res.Error = err.Error()
			}

			log.Debug().Dur("duration", time.Since(start)).Str("task_id", id).Msg("Task processed")

			if err := c.Bridge.Notify(string(CallbackAIResult), res); err != nil {
				log.Error().Err(err).Str("task_id", id).Msg("failed to send result back")
			}
		})
	}
}

func (c *Controller) handleErrorTelemetry(err error) {
	var pErr *p_error.ProviderError
	if errors.As(err, &pErr) {
		c.NotifyTele(pErr.Friendly(), LogLevelError)
		return
	}

	if errors.Is(err, p_error.ErrAllProvidersFailed) {
		c.NotifyTele("All AI providers failed. Check :messages for details.", LogLevelError)
		return
	}

	c.NotifyTele("Engine error: "+err.Error(), LogLevelError)
}

func (c *Controller) RegisterHandlers() {
	stack := []middleware.Middleware{
		middleware.WithRecovery,
		middleware.WithMeasure,
	}

	c.Handlers[MethodSubmitTask] = middleware.Chain(
		BindHandler(c, "Processing task", c.Proc.Process),
		stack...,
	)

	c.Handlers[MethodSubmitChat] = middleware.Chain(
		BindHandler(c, "Thinking...", func(t types.ChatTask) ([]string, error) {
			resp, err := c.Proc.ProcessChat(t)
			return []string{resp}, err
		}),
		stack...,
	)
}

func (c *Controller) Listen(dec *msgpack.Decoder, sigChan chan<- os.Signal) {
	log := logger.Get()

	shutdown := func(reason string) {
		log.Info().Str("reason", reason).Msg("Initiating shutdown...")
		select {
		case sigChan <- syscall.SIGTERM:
		default:
		}
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("CRITICAL: Recovered from panic in main listener loop!")
			shutdown("panic in listener")
		}
	}()

	decodeErrors := 0
	for {
		var msg types.RPCNotification

		err := dec.Decode(&msg)
		switch {
		case errors.Is(err, io.EOF):
			shutdown("Neovim closed STDIN (EOF)")
			return
		case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, os.ErrClosed):
			shutdown("STDIN closed mid-message")
			return
		case err != nil:
			decodeErrors++
			log.Error().Err(err).Int("consecutive", decodeErrors).
				Msg("Failed to decode MessagePack payload; the stream may be desynchronised")

			if decodeErrors >= maxDecodeErrors {
				c.NotifyTele("Engine lost sync with Neovim and is shutting down. Please restart it.", LogLevelError)
				shutdown("too many consecutive decode errors")
				return
			}

			continue
		}

		decodeErrors = 0

		if msg.Method == "" {
			continue
		}

		log.Debug().Str("method", msg.Method).Msg("Received RPC task")
		c.Dispatch(msg)
	}
}

func (c *Controller) Dispatch(msg types.RPCNotification) {
	if msg.Type != 2 {
		return
	}

	method := RPCMethod(msg.Method)

	handler, ok := c.Handlers[method]
	if !ok {
		logger.Get().Debug().Str("method", msg.Method).Msg("Unknown RPC method")
		return
	}

	handler(msg)
}

func (c *Controller) NotifyTele(msg string, level LogLevel) {
	if err := c.Bridge.Notify(string(CallbackNvimLog), msg, string(level), EngineName); err != nil {
		logger.Get().Error().Err(err).Msg("failed to send log to nvim")
	}
}
