package engine

import "github.com/vmihailenco/msgpack/v5"

type Identifiable interface {
	GetID() string
}

type Task struct {
	ID      string `msgpack:"id"`
	Action  string `msgpack:"action"`
	Payload string `msgpack:"payload"`
}

func (t Task) GetID() string { return t.ID }

type ChatTask struct {
	ID      string   `msgpack:"id"`
	Prompt  string   `msgpack:"prompt"`
	History []string `msgpack:"history,omitempty"`
}

func (t ChatTask) GetID() string { return t.ID }

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
