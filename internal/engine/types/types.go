package types

import "github.com/vmihailenco/msgpack/v5"

type TaskHandler func(msg RPCNotification)

type RPCNotification struct {
	Type   int
	Method string
	Args   []msgpack.RawMessage
}
