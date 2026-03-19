package logger

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
)

type SafeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (s *SafeBuffer) Read(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.Read(p)
}

func (s *SafeBuffer) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.Write(p)
}

func (s *SafeBuffer) ReadByte() (byte, error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.ReadByte()
}

func (s *SafeBuffer) WriteByte(c byte) error {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.WriteByte(c)
}

func TestNvimBridge_Notify_Format(t *testing.T) {
	var buf SafeBuffer
	bridge := NewNvimBridge(&buf)

	method := "test_method"
	err := bridge.Notify(method, "hello", 42)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	dec := msgpack.NewDecoder(&buf)
	var result []any
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("Failed to decode MessagePack: %v", err)
	}

	if result[0].(int8) != 2 {
		t.Errorf("Expected message type 2 (Notification), got %v", result[0])
	}
	if result[1].(string) != "nvim_exec_lua" {
		t.Errorf("Expected method 'nvim_exec_lua', got %v", result[1])
	}

	execArgs := result[2].([]any)
	expectedLua := fmt.Sprintf("return _G['%s'](...)", method)
	if execArgs[0].(string) != expectedLua {
		t.Errorf("Expected Lua code %q, got %q", expectedLua, execArgs[0])
	}

	innerArgs := execArgs[1].([]any)
	if len(innerArgs) != 2 || innerArgs[0].(string) != "hello" || innerArgs[1].(int8) != 42 {
		t.Errorf("Invalid inner arguments: %v", innerArgs)
	}
}

func TestNvimLogHook_Run(t *testing.T) {
	var buf SafeBuffer
	bridge := NewNvimBridge(&buf)
	hook := &NvimLogHook{bridge: bridge}

	hook.Run(nil, zerolog.DebugLevel, "this is hidden")
	hook.Run(nil, zerolog.InfoLevel, "this is info")

	time.Sleep(50 * time.Millisecond)

	dec := msgpack.NewDecoder(&buf)
	var result []any
	err := dec.Decode(&result)
	if err != nil {
		t.Fatalf("Expected log in buffer, but buffer is empty: %v", err)
	}

	execArgs := result[2].([]any)
	innerArgs := execArgs[1].([]any)

	if innerArgs[0].(string) != "this is info" {
		t.Errorf("Expected message 'this is info', got: %v", innerArgs[0])
	}
	if innerArgs[1].(string) != "INFO" {
		t.Errorf("Expected level 'INFO', got: %v", innerArgs[1])
	}
}

func TestNvimBridge_Notify_Concurrency(t *testing.T) {
	var buf SafeBuffer
	bridge := NewNvimBridge(&buf)

	var wg sync.WaitGroup
	workers := 100

	for i := range workers {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			_ = bridge.Notify("spam", val)
		}(i)
	}

	wg.Wait()

	dec := msgpack.NewDecoder(&buf)
	count := 0
	for {
		var dummy []any
		err := dec.Decode(&dummy)
		if err != nil {
			break
		}
		count++
	}

	if count != workers {
		t.Errorf("Expected %d packed messages, read %d. Mutex failed!", workers, count)
	}
}
