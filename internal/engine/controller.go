package engine

import (
	"nvim-engine/internal/logger"

	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

type Controller struct {
	Proc   *Processor
	Bridge *logger.NvimBridge
}

func (c *Controller) Dispatch(msg RPCNotification) {
	if msg.Type != 2 {
		return
	}

	switch msg.Method {
	case "submit_task":
		if len(msg.Args) == 0 {
			return
		}

		var task Task
		if err := msgpack.Unmarshal(msg.Args[0], &task); err != nil {
			c.NotifyTele("Failed processing task: "+err.Error(), "ERROR")
			return
		}

		c.Proc.Pool.Submit(func() {
			c.NotifyTele("Processing task: "+task.ID, "INFO")

			data, err := c.Proc.Process(task)

			res := Result{ID: task.ID, Data: data}
			if err != nil {
				res.Error = err.Error()
			}

			if err := c.Bridge.Notify("on_ai_result", res); err != nil {
				log.Error().Err(err).Msg("failed to notify nvim with ai result")
			}
		})
	}
}

func (c *Controller) NotifyTele(msg, level string) {
	if err := c.Bridge.Notify("NvimEngineLog", msg, level, "Go-Engine"); err != nil {
		log.Error().Err(err).Msg("failed to send log to nvim")
	}
}
