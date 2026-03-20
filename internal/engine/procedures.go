package engine

type (
	RPCMethod   string
	RPCCallback string
	LogLevel    string
)

const (
	MethodSubmitTask RPCMethod = "submit_task"
	MethodSubmitChat RPCMethod = "submit_chat"

	CallbackAIResult RPCCallback = "on_ai_result"
	CallbackNvimLog  RPCCallback = "NvimEngineLog"

	LogLevelInfo  LogLevel = "INFO"
	LogLevelError LogLevel = "ERROR"
	EngineName    string   = "Bifröst"
)
