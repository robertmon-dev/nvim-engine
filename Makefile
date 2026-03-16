# ==============================================================================
# Neovim Go-Engine Makefile
# ==============================================================================

APP_NAME := nvim-ai-engine
BUILD_DIR := bin
MAIN_PKG := ./cmd/engine
INSTALL_DIR := $(HOME)/.config/nvim/bin

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
