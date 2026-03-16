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
	@echo "=> Building optimized $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "=> Build complete: $(BUILD_DIR)/$(APP_NAME)"

.PHONY: install
install: clean build
	@echo "=> Installing production binary to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(APP_NAME) $(INSTALL_DIR)/
	@echo "=> Done! Binary size: $$(du -h $(INSTALL_DIR)/$(APP_NAME) | cut -f1)"

.PHONY: run
run: build
	@echo "=> Running $(APP_NAME) in manual mode..."
	@echo "=> Note: Engine expects MessagePack-RPC input. Press Ctrl+C to stop."
	@./$(BUILD_DIR)/$(APP_NAME)

# ==============================================================================
# Testing & Linting
# ==============================================================================

.PHONY: test
test:
	@echo "=> Running tests..."
	@go test -v ./...

.PHONY: cover
cover:
	@echo "=> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

.PHONY: lint
lint:
	@echo "=> Running linter..."
	@golangci-lint run ./...

.PHONY: lint-clean
lint-clean:
	@echo "=> Cleaning linter cache..."
	@golangci-lint cache clean

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean:
	@echo "=> Cleaning up..."
	@rm -rf $(BUILD_DIR) coverage.out
	@echo "=> Clean complete."
