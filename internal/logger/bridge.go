package logger

import (
	"errors"
	"fmt"
	"io"
	"regexp"
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

var luaIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func NewNvimBridge(w io.Writer) *NvimBridge {
	return &NvimBridge{w: w}
}

func AttachBridge(b *NvimBridge) {
	log = log.Hook(&NvimLogHook{bridge: b})
}

func (b *NvimBridge) Notify(method string, args ...any) error {
	if b == nil || b.w == nil {
		return errors.New("bridge not initialized")
	}

	if !luaIdent.MatchString(method) {
		return fmt.Errorf("refusing to call %q: not a Lua identifier", method)
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
	switch {
	case level >= zerolog.ErrorLevel:
		lvlStr = "ERROR"
	case level == zerolog.WarnLevel:
		lvlStr = "WARN"
	}

	_ = h.bridge.Notify("NvimEngineLog", msg, lvlStr, "Bifröst")
}
