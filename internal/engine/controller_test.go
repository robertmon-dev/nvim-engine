package engine

import (
	"errors"
	"strings"
	"testing"

	"nvim-engine/internal/engine/types"
	"nvim-engine/mocks"

	"github.com/vmihailenco/msgpack/v5"
)

func TestController_Dispatch_SubmitTask_Mocked(t *testing.T) {
	tests := []struct {
		name        string
		mockErr     error
		expectedLog string
	}{
		{
			name:        "Success path",
			mockErr:     nil,
			expectedLog: "Processing task",
		},
		{
			name:        "Provider error handles telemetry",
			mockErr:     errors.New("processor explosion"),
			expectedLog: "Engine error: processor explosion",
		},
		{
			name:        "All providers failed telemetry",
			mockErr:     errors.New("some context: all attempted providers failed: openai failed"),
			expectedLog: "All AI providers failed. Check :messages for details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mProc := mocks.NewMockProcessor()
			mProc.ProcessFunc = func(task types.Task) ([]string, error) {
				return []string{"res"}, tt.mockErr
			}
			mBridge := &mocks.MockBridge{}

			ctrl := &Controller{
				Proc:     mProc,
				Bridge:   mBridge,
				Handlers: make(map[RPCMethod]types.TaskHandler),
			}
			ctrl.RegisterHandlers()

			task := types.Task{ID: "test-123", Action: "commit"}
			taskBytes, _ := msgpack.Marshal(task)
			msg := types.RPCNotification{
				Type: 2, Method: string(MethodSubmitTask),
				Args: []msgpack.RawMessage{taskBytes},
			}

			ctrl.Dispatch(msg)
			mProc.GetPool().StopAndWait()

			calls := mBridge.GetCalls()

			foundLog := false
			foundResult := false

			for _, call := range calls {
				if call.Method == string(CallbackNvimLog) {
					if msg, ok := call.Args[0].(string); ok && strings.Contains(msg, tt.expectedLog) {
						foundLog = true
					}
				}
				if call.Method == string(CallbackAIResult) {
					foundResult = true
				}
			}

			if !foundLog {
				t.Errorf("Expected log containing %q not found. Actual calls: %+v", tt.expectedLog, calls)
			}
			if !foundResult {
				t.Error("AI result notification was never sent")
			}
		})
	}
}
