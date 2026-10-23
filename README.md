# nvim-engine

**nvim-engine** is an asynchronous AI engine for the Neovim editor, written in Go. It was designed for speed, reliability, and the complete elimination of UI freezes during communication with LLM models.

## Key Features

* **Multi-Provider Failover:** Supports several providers (Gemini, Anthropic, OpenAI, Ollama). A request starts at a rotating provider and, if that one returns an error (e.g. *Rate Limit 429*), walks the rest of the ring until one answers. Only when every provider has refused is the task reported as failed.
* **Asynchronous Processing:** Uses a worker pool (`pond`), so tasks are queued and processed in the background without affecting Neovim's responsiveness.
* **RPC Communication:** Uses the MessagePack-RPC protocol (notifications only) for fast data exchange between Go and Lua.
* **Optimized Binary:** Compiled with `-s -w` and `-trimpath`, stripped of symbols, debug information and local development paths (~7 MB).
* **Commit Generation:** Built-in logic for generating commit messages compliant with the *Conventional Commits* standard.

## Installation

### Requirements
* [Go](https://go.dev/) (version 1.25 or newer)
* [Neovim](https://neovim.io/) (0.9+)
* An API key for at least one provider, or a running Ollama

### Build and Installation

```bash
git clone https://github.com/robertmon-dev/nvim-engine.git
cd nvim-engine

make install
```

The binary will be installed to: `~/.config/nvim/bin/nvim-ai-engine`.

## Configuration

Everything is read from the environment, once, at startup. See
[`.env.example`](.env.example) for the full list with defaults; the engine
refuses to start when no provider is usable.

### Providers

```bash
# Comma-separated. Several keys for one provider are rotated per request.
export GEMINI_API_KEYS="<your keys>"
export OPENAI_API_KEYS="<your keys>"
export ANTHROPIC_API_KEYS="<your keys>"

export GEMINI_MODEL="gemini-3.8-flash"
export OPENAI_MODEL="gpt-6-astra"
export ANTHROPIC_MODEL="claude-sonnet-5-5"

# Optional: which providers to try first. Unlisted ones are appended in the
# default order (gemini, anthropic, openai, ollama).
export PROVIDER_ORDER="anthropic,openai"
```

Providers without a key (or, for Ollama, without both `OLLAMA_MODEL` and
`OLLAMA_URL`) are skipped, so a partial setup is fine — the engine simply
fails over between whatever is configured.

(Ollama has strange latency locally, I would rather suggest not using its models.)

### Engine and logging

| Variable | Default | Meaning |
|---|---|---|
| `AI_ENGINE_WORKERS` | `10` | Worker pool size |
| `AI_ENGINE_CAPACITY` | `20` | Queue capacity |
| `AI_ENGINE_MAX_RETRIES` | `2` | How many times a task re-runs the whole provider ring |
| `LOG_LEVEL` | `debug` | `trace` also logs full request and response bodies |
| `LOG_PATH` | `$TMPDIR/nvim-engine.log` | Log file, created with mode `0600` |

Anything unparseable or `<= 0` falls back to the default rather than producing
a zero-sized pool. If the log file cannot be opened, the engine warns on stderr
and falls back to `$TMPDIR`, then to stderr itself — it never silently stops
logging. `stdout` is the RPC channel and is never logged to.

## Project Structure

```text
├── cmd/engine/main.go           # startup, signal handling, shutdown
├── internal
│   ├── config                   # environment -> Config, validation
│   ├── engine
│   │   ├── controller.go        # RPC listen loop and dispatch
│   │   ├── processor.go         # worker pool, retries, option parsing
│   │   ├── procedures.go        # RPC method and callback names
│   │   ├── prompts.go           # system prompts
│   │   ├── middleware           # recovery, measurement
│   │   └── types                # RPC and task types
│   ├── logger
│   │   ├── logger.go            # zerolog setup
│   │   └── bridge.go            # MessagePack-RPC writer + log hook
│   └── provider
│       ├── provider.go          # shared HTTP plumbing, provider wiring
│       ├── dispatcher.go        # failover across providers
│       ├── keyring.go           # API key rotation
│       ├── gemini.go / anthropic.go / openai.go / ollama.go
│       └── p_error              # provider error classification
├── mocks                        # hand-written test doubles
├── tests
│   ├── integration              # pytest driving a real Neovim
│   └── lua                      # Lua bridge used by the integration tests
└── Makefile
```

## Development and Testing

* **Run tests:** `make test` (runs with Go's race detector)
* **Run linter:** `make lint` (requires `golangci-lint`)
* **Coverage report:** `make cover`
* **Integration tests:** `make test-integration` (starts a real Neovim via `pynvim`; needs a provider key)

Unit tests use hand-written mocks and `httptest` servers, so the suite runs
without an internet connection or API quota. The integration suite does talk to
a provider and expects `OPENAI_API_KEYS` (or whatever `PROVIDER_ORDER` names).

## Support

The engine sends Info-level and worse back to Neovim via RPC, so `:messages`
usually has the headline. The full log is at `$LOG_PATH`, or
`$TMPDIR/nvim-engine.log` when that is unset:

```bash
tail -f "${TMPDIR:-/tmp}/nvim-engine.log"
```

Set `LOG_LEVEL=trace` to include full request and response bodies. Note that
this writes your prompts and diffs to disk; the file is mode `0600`.

---
