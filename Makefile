# ==============================================================================
# Neovim Bifrost Makefile
# ==============================================================================

APP_NAME := nvim-ai-engine
BUILD_DIR := bin
MAIN_PKG := ./cmd/engine
INSTALL_DIR := $(HOME)/.config/nvim/bin
VENV = tests/.venv
PYTHON = $(VENV)/bin/python
PIP = $(VENV)/bin/pip

$(VENV):
	@python3 -m venv $(VENV)
	@$(PIP) install -q -r tests/requirements.txt

.PHONY: all
all: build

# ==============================================================================
# Build & Install
# ==============================================================================

.PHONY: build
build:
	@echo "=> Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -trimpath -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "=> Build complete: $(BUILD_DIR)/$(APP_NAME)"

.PHONY: install
install: build
	@echo "=> Killing existing instances of $(APP_NAME)..."
	@pkill -f $(APP_NAME) || true
	@echo "=> Installing $(APP_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(APP_NAME) $(INSTALL_DIR)/
	@echo "=> Install complete. Neovim is ready to use the engine!"

# ==============================================================================
# Release
# ==============================================================================

.PHONY: release
release:
	@chmod +x scripts/release.sh
	@./scripts/release.sh

# ==============================================================================
# Testing & Linting
# ==============================================================================

.PHONY: test
test:
	@echo "=> Running tests..."
	@go test -v -race ./...

.PHONY: test-integration
test-integration: build $(VENV)
	@echo "=> Running E2E Integration Tests..."
	@$(VENV)/bin/pytest -v --timeout=15 tests/integration

.PHONY: cover
cover:
	@echo "=> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

.PHONY: lint
lint:
	@echo "=> Running linter..."
	@golangci-lint run ./...

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean:
	@echo "=> Cleaning up..."
	@rm -rf $(BUILD_DIR) dist/ coverage.out
	@echo "=> Clean complete."
