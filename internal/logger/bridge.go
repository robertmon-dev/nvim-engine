package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
)

type NvimBridge struct {
	mu  sync.Mutex
	enc *msgpack.Encoder
}

func NewNvimBridge(enc *msgpack.Encoder) *NvimBridge {
	return &NvimBridge{enc: enc}
}

func AttachBridge(b *NvimBridge) {
	log = log.Hook(&NvimLogHook{bridge: b})
}

func (b *NvimBridge) Notify(method string, args ...any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if args == nil {
		args = []any{}
	}

	luaCode := fmt.Sprintf("return _G['%s'](...)", method)

	return b.enc.Encode([]any{
		2,
		"nvim_exec_lua",
		[]any{luaCode, args},
	})
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
		if err := h.bridge.Notify("NvimEngineLog", msg, lvlStr, "Go-Engine"); err != nil {
			fmt.Fprintf(os.Stderr, "RPC Log Error: %v\n", err)
		}
	}()
}
