package engine

import (
	"io"
	"os"
	"syscall"
	"time"

	"nvim-engine/internal/engine/middleware"
	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"

	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

type Controller struct {
	Proc     *Processor
	Bridge   *logger.NvimBridge
	Handlers map[RPCMethod]types.TaskHandler
}

func (c *Controller) RegisterHandlers() {
	commonMiddleware := []middleware.Middleware{
		middleware.WithRecovery,
		middleware.WithLogging,
		middleware.WithMeasure,
	}

	c.Handlers[MethodSubmitTask] = middleware.Chain(c.handleSubmitTask, commonMiddleware...)
}

func (c *Controller) handleSubmitTask(msg types.RPCNotification) {
	if len(msg.Args) == 0 {
		return
	}

	var task Task
	if err := msgpack.Unmarshal(msg.Args[0], &task); err != nil {
		c.NotifyTele("Failed processing task: "+err.Error(), LogLevelError)
		return
	}

	c.Proc.Pool.Submit(func() {
		start := time.Now()

		c.NotifyTele("Processing task: "+task.ID, LogLevelInfo)
		data, err := c.Proc.Process(task)

		res := Result{ID: task.ID, Data: data}
		if err != nil {
			res.Error = err.Error()
		}

		log.Debug().Dur("actual_work_duration", time.Since(start)).Msg("Task finished in pool")

		if err := c.Bridge.Notify(string(CallbackAIResult), res); err != nil {
			log.Error().Err(err).Msg("failed to notify nvim")
		}
	})
}

func (c *Controller) handleChat(msg types.RPCNotification) {
	if len(msg.Args) == 0 {
		return
	}

	var chatTask ChatTask
	if err := msgpack.Unmarshal(msg.Args[0], &chatTask); err != nil {
		c.NotifyTele("Chat unmarshal error: "+err.Error(), LogLevelError)
		return
	}

	c.Proc.Pool.Submit(func() {
		start := time.Now()
		c.NotifyTele("Chatting: "+chatTask.ID, LogLevelInfo)

		data, err := c.Proc.ProcessChat(chatTask)

		res := Result{ID: chatTask.ID, Data: data}
		if err != nil {
			res.Error = err.Error()
		}

		log.Debug().Dur("chat_duration", time.Since(start)).Msg("Chat task finished")

		c.Bridge.Notify(string(CallbackAIResult), res)
	})
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
