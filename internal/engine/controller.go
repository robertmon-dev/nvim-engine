package engine

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"nvim-engine/internal/engine/middleware"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider/p_error"

	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

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
		if len(msg.Args) == 0 {
			return
		}

		var task T
		if err := msgpack.Unmarshal(msg.Args[0], &task); err != nil {
			c.NotifyTele("Decoding error: "+err.Error(), LogLevelError)
			return
		}

		c.Proc.Submit(func() {
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

	if strings.Contains(err.Error(), "all attempted providers failed") {
		c.NotifyTele("All AI providers failed. Check :messages for details.", LogLevelError)
		return
	}

	c.NotifyTele("Engine error: "+err.Error(), LogLevelError)
}

func (c *Controller) RegisterHandlers() {
	stack := []middleware.Middleware{
		middleware.WithRecovery,
		middleware.WithLogging,
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

	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("CRITICAL: Recovered from panic in main listener loop!")
			sigChan <- syscall.SIGTERM
		}
	}()

	for {
		var msg types.RPCNotification
		err := dec.Decode(&msg)

		if err == io.EOF {
			log.Info().Msg("Neovim closed STDIN (EOF). Initiating shutdown...")
			sigChan <- syscall.SIGTERM
			break
		}

		if err != nil {
			log.Error().Err(err).Msg("Failed to decode MessagePack payload")
			continue
		}

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

	if handler, ok := c.Handlers[method]; ok {
		handler(msg)
	} else {
		log.Debug().Str("method", msg.Method).Msg("Unknown RPC method")
	}
}

func (c *Controller) NotifyTele(msg string, level LogLevel) {
	if err := c.Bridge.Notify(string(CallbackNvimLog), msg, string(level), EngineName); err != nil {
		log.Error().Err(err).Msg("failed to send log to nvim")
	}
}
