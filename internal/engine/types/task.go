package types

type Identifiable interface {
	GetID() string
}

type Task struct {
	ID      string `msgpack:"id"`
	Action  string `msgpack:"action"`
	Payload string `msgpack:"payload"`
}

func (t Task) GetID() string { return t.ID }

type Message struct {
	Role    string `msgpack:"role"`
	Content string `msgpack:"content"`
}

type ChatTask struct {
	ID       string    `msgpack:"id"`
	Prompt   string    `msgpack:"prompt"`
	Messages []Message `msgpack:"messages"`
}

func (t ChatTask) GetID() string { return t.ID }

type Result struct {
	ID    string   `msgpack:"id"`
	Data  []string `msgpack:"data"`
	Error string   `msgpack:"error"`
}
