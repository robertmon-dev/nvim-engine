package engine

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider/p_error"
	"nvim-engine/mocks"

	"github.com/vmihailenco/msgpack/v5"
)

type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Bytes()
}

func TestController_Dispatch_SubmitTask_Detailed(t *testing.T) {
	const (
		cmdLog    = "nvim_log"
		cmdResult = "ai_result"
	)

	tests := []struct {
		name             string
		mockErr          error
		expectedLog      string
		expectedLogLevel string
	}{
		{
			name:             "Success path",
			mockErr:          nil,
			expectedLog:      "Processing task",
			expectedLogLevel: "info",
		},
		{
			name: "Provider Rate Limit Error",
			mockErr: &p_error.ProviderError{
				Code:     p_error.ErrRateLimit,
				Provider: "gemini",
				Message:  "too many requests",
			},
			expectedLog:      "Gemini: You've hit the API rate limit",
			expectedLogLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mProc := mocks.NewMockProcessor()
			mProc.ProcessFunc = func(task types.Task) ([]string, error) {
				return []string{"res"}, tt.mockErr
			}

			var sBuf safeBuffer

			bridge := logger.NewNvimBridge(&sBuf)

			ctrl := &Controller{
				Proc:     mProc,
				Bridge:   bridge,
				Handlers: make(map[RPCMethod]types.TaskHandler),
			}
			ctrl.RegisterHandlers()

			taskBytes, _ := msgpack.Marshal(types.Task{ID: "test-123", Action: "commit"})
			msg := types.RPCNotification{
				Type:   2,
				Method: string(MethodSubmitTask),
				Args:   []msgpack.RawMessage{taskBytes},
			}

			ctrl.Dispatch(msg)

			mProc.GetPool().StopAndWait()

			data := sBuf.Bytes()
			if len(data) == 0 {
				t.Fatal("Buffer is empty! Controller did not write anything to Bridge.")
			}

			dec := msgpack.NewDecoder(bytes.NewReader(data))
			var capturedLogs []string
			var resultSent bool

			for {
				var raw []any
				err := dec.Decode(&raw)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Logf("Warning: decode error: %v", err)
					break
				}

				method := raw[1].(string)
				args := raw[2].([]any)

				if method == cmdLog {
					capturedLogs = append(capturedLogs, args[0].(string))
				} else if method == cmdResult {
					resultSent = true
				}
			}

			foundExpectedLog := false
			for _, l := range capturedLogs {
				if strings.Contains(l, tt.expectedLog) {
					foundExpectedLog = true
					break
				}
			}

			if !foundExpectedLog {
				t.Errorf("Expected log containing %q not found.\nCaptured logs: %v", tt.expectedLog, capturedLogs)
			}
			if !resultSent {
				t.Error("AI result message was never sent back via bridge")
			}
		})
	}
}
