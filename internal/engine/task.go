package engine

import "github.com/vmihailenco/msgpack/v5"

type Task struct {
	ID      string `msgpack:"id"`
	Action  string `msgpack:"action"`
	Payload string `msgpack:"payload"`
}

type ChatTask struct {
	ID      string `msgpack:"id"`
	Prompt  string `msgpack:"prompt"`
	History []any  `msgpack:"history,omitempty"`
}

type Result struct {
	ID    string   `msgpack:"id"`
	Data  []string `msgpack:"data"`
	Error string   `msgpack:"error"`
}

type RPCNotification struct {
	Type   int
	Method string
	Args   []msgpack.RawMessage
}
