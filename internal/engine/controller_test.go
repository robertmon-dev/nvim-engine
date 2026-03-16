package engine

import (
	"bytes"
	"context"
	"testing"

	"nvim-engine/internal/logger"
	"nvim-engine/internal/provider"
	"nvim-engine/mocks"

	"github.com/vmihailenco/msgpack/v5"
)

func TestController_Dispatch_SubmitTask(t *testing.T) {
	mockProv := &mocks.MockProvider{
		GenerateFunc: func(ctx context.Context, sys, user string) (string, error) {
			return "feat: tested controller logic", nil
		},
	}
	providers := map[provider.ID]provider.Provider{
		provider.Gemini: mockProv,
	}

	proc := NewProcessor(1, 10, providers)
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	bridge := logger.NewNvimBridge(enc)

	ctrl := &Controller{
		Proc:   proc,
		Bridge: bridge,
	}

	task := Task{ID: "test-123", Action: "commit", Payload: "diff"}
	taskBytes, _ := msgpack.Marshal(task)

	msg := RPCNotification{
		Type:   2,
		Method: "submit_task",
		Args:   []msgpack.RawMessage{taskBytes},
	}

	ctrl.Dispatch(msg)
	proc.Pool.StopAndWait()

	dec := msgpack.NewDecoder(&buf)

	var logMsg []any
	if err := dec.Decode(&logMsg); err != nil {
		t.Fatalf("Log msg missing: %v", err)
	}

	logExecArgs := logMsg[2].([]any)
	logInnerArgs := logExecArgs[1].([]any)

	if logInnerArgs[0] != "Processing task: test-123" {
		t.Errorf("Bad log content. Got: %v", logInnerArgs[0])
	}

	var resultMsg []any
	if err := dec.Decode(&resultMsg); err != nil {
		t.Fatalf("Result msg missing: %v", err)
	}

	resExecArgs := resultMsg[2].([]any)
	resLuaArgs := resExecArgs[1].([]any)

	var resData map[string]any

	switch m := resLuaArgs[0].(type) {
	case map[string]any:
		resData = m
	case map[any]any:
		resData = make(map[string]any)
		for k, v := range m {
			resData[k.(string)] = v
		}
	default:
		t.Fatalf("Unexpected type for resData: %T", resLuaArgs[0])
	}

	if resData["id"] != "test-123" {
		t.Errorf("Bad task ID. Expected 'test-123', got '%v'", resData["id"])
	}

	options, ok := resData["data"].([]any)
	if !ok || len(options) == 0 || options[0] != "feat: tested controller logic" {
		t.Errorf("Bad AI result. Got: %#v", resData["data"])
	}
}
