package logger

import (
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
)

type NvimBridge struct {
	mu sync.Mutex
	w  io.Writer
}

type NvimBridgeInterface interface {
	Notify(method string, args ...any) error
}

func NewNvimBridge(w io.Writer) *NvimBridge {
	return &NvimBridge{w: w}
}

func AttachBridge(b *NvimBridge) {
	log = log.Hook(&NvimLogHook{bridge: b})
}

func (b *NvimBridge) Notify(method string, args ...any) error {
	if b == nil || b.w == nil {
		return fmt.Errorf("bridge not initialized")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	luaCode := fmt.Sprintf("return _G['%s'](...)", method)

	packet := []any{
		2,
		"nvim_exec_lua",
		[]any{luaCode, args},
	}

	data, err := msgpack.Marshal(packet)
	if err != nil {
		return err
	}

	_, err = b.w.Write(data)
	return err
}

type NvimLogHook struct {
	bridge *NvimBridge
}

func (h *NvimLogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level < zerolog.InfoLevel || msg == "" || h.bridge == nil {
		return
	}

	lvlStr := "INFO"
	if level == zerolog.WarnLevel {
		lvlStr = "WARN"
	} else if level >= zerolog.ErrorLevel {
		lvlStr = "ERROR"
	}

	go func() {
		_ = h.bridge.Notify("NvimEngineLog", msg, lvlStr, "Go-Engine")
	}()
}
