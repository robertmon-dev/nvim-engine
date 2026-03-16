# nvim-engine

**nvim-engine** is a high-performance, asynchronous AI engine for the Neovim editor, written in Go. It was designed for speed, reliability, and the complete elimination of UI freezes during communication with LLM models.

## Key Features

* **Multi-Provider Failover:** Support for multiple providers (Gemini, Anthropic, OpenAI). If one of them returns an error (e.g., *Rate Limit 429*), the engine automatically switches to the next one in the list.
* **Asynchronous Processing:** Uses an advanced Worker Pool (`pond`), ensuring tasks are queued and processed in the background without affecting Neovim's responsiveness.
* **RPC Communication:** Utilizes the MessagePack-RPC protocol for ultra-fast data exchange between Go and Lua.
* **Optimized Binary:** Compilation with `-s -w` and `-trimpath` flags ensures a lightweight executable (6.6MB) stripped of unnecessary symbols, debug information, and local development paths.
* **Commit Generation:** Built-in logic for generating professional commit messages compliant with the *Conventional Commits* standard.

## Installation

### Requirements
* [Go](https://go.dev/) (version 1.25 or newer)
* [Neovim](https://neovim.io/) (0.9+)
* API keys for your selected providers (Gemini, OpenAI, Anthropic)

### Build and Installation
The project includes a robust `Makefile` to automate the build and installation process:

```bash

git clone https://github.com/robertmon-dev/nvim-engine.git
cd nvim-engine

make install
```

The binary will be installed to: `~/.config/nvim/bin/nvim-engine`.

## Configuration

### Environment Variables
The engine automatically reads API keys from your environment. You can set these in your `.zshrc`, `.bashrc`, or a `.env` file:

```bash
export GEMINI_API_KEY="<pass your keys>"
export OPENAI_API_KEY="<pass your keys>>"
export ANTHROPIC_API_KEY="<pass your keys>"

export GEMINI_MODEL="gemini-2.0-flash"
export OPENAI_MODEL="gpt-4o"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"
```

## Project Structure

```text
├── cmd
│   └── engine
│       └── main.go
├── go.mod
├── go.sum
├── internal
│   ├── config
│   │   └── config.go
│   ├── engine
│   │   ├── controller.go
│   │   ├── processor.go
│   │   ├── prompts.go
│   │   └── task.go
│   ├── logger
│   │   ├── bridge.go
│   │   └── logger.go
│   └── provider
│       ├── anthropic.go
│       ├── gemini.go
│       ├── openai.go
│       ├── provider.go
│       └── providers_test.go
├── Makefile
└─── mocks
     └── provider.go
```

## Development and Testing

The project emphasizes code quality and thread safety, utilizing Go's race detector during testing.

* **Run tests:** `make test`
* **Run linter:** `make lint` (requires `golangci-lint`)
* **Coverage report:** `make cover`

All tests utilize a `MockProvider`, allowing the suite to run without an active internet connection or consuming API quotas.

## Support

The engine sends real-time logs back to Neovim via RPC. If you encounter issues, you can check the logs by inspecting the temporary log file:
`/tmp/nvim-engine.log`

---
