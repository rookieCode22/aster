# Aster Build System
# ==================
#
# Quick start:
#   make deps        Install build dependencies (garble)
#   make keygen      Generate encryption key fragments
#   make modules     Encrypt skills and rules into .astm files
#   make build       Build obfuscated aster-server binary
#   make build-dev   Build development binary (no obfuscation, debug symbols)
#   make test        Run all tests
#   make clean       Clean build artifacts
#
# Three deployment modes:
#   make build-server  Build for SaaS/web deployment
#   make build-box     Build for hardware appliance (linux/amd64)
#   make build-all     Build all platform binaries

.PHONY: all deps keygen modules build build-dev test clean build-server build-box build-all

# ---- Configuration ----
APP_NAME    := aster-server
BUILD_DIR   := build
MODULES_DIR := build/modules
KEYGEN_SEED ?= aster-prod-build

# Platforms
GOOS_LINUX   := linux
GOARCH_AMD64 := amd64
GOOS_WINDOWS := windows

# garble settings
GARBLE_FLAGS := -literals -tiny
GARBLE       := garble $(GARBLE_FLAGS)

# Go settings
GO      := go
GOPATH  := $(shell go env GOPATH)
GOBIN   := $(GOPATH)/bin

# ---- Default ----
all: build

# ---- Dependencies ----
deps:
	@echo "=== Installing garble ==="
	$(GO) install mvdan.cc/garble@latest
	@echo "=== Done ==="

# ---- Key Generation ----
keygen:
	@echo "=== Generating key fragments ==="
	@mkdir -p $(BUILD_DIR)
	$(GO) run cmd/keygen/main.go --seed=$(KEYGEN_SEED) 2>&1 | tee $(BUILD_DIR)/.key
	@echo "=== Key saved to $(BUILD_DIR)/.key ==="

# ---- Module Encryption ----
modules: keygen
	@echo "=== Encrypting modules ==="
	@mkdir -p $(MODULES_DIR)
	@KEY=$$(tail -1 $(BUILD_DIR)/.key); \
	$(GO) run cmd/encrypt/main.go \
		--in=skills \
		--out=$(MODULES_DIR)/aster-core-skills.astm \
		--module=aster-core \
		--version=1.0.0 \
		--type=skills \
		--key=$$KEY 2>&1
	@KEY=$$(tail -1 $(BUILD_DIR)/.key); \
	$(GO) run cmd/encrypt/main.go \
		--in=semgrep-rules \
		--out=$(MODULES_DIR)/aster-core-rules.astm \
		--module=aster-core \
		--version=1.0.0 \
		--type=rules \
		--key=$$KEY 2>&1
	@echo "=== Modules encrypted ==="

# ---- Builds ----
build-dev:
	@echo "=== Building development binary ==="
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/aster/
	@echo "=== Built $(BUILD_DIR)/$(APP_NAME) ==="

build: keygen modules
	@echo "=== Building obfuscated binary ==="
	$(GARBLE) build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/aster/
	@echo "=== Built $(BUILD_DIR)/$(APP_NAME) ==="

build-server: keygen modules
	@echo "=== Building SaaS/server binary ==="
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) $(GARBLE) build \
		-ldflags="-s -w" \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 \
		./cmd/aster/
	@echo "=== Built $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ==="

build-box: keygen modules
	@echo "=== Building appliance binary ==="
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) $(GARBLE) build \
		-ldflags="-s -w" \
		-o $(BUILD_DIR)/$(APP_NAME)-box \
		./cmd/aster/
	@echo "=== Built $(BUILD_DIR)/$(APP_NAME)-box ==="

build-all: build-server
	@echo "=== Building Windows binary ==="
	GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_AMD64) $(GARBLE) build \
		-ldflags="-s -w" \
		-o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe \
		./cmd/aster/
	@echo "=== All binaries built ==="

# ---- Testing ----
test:
	$(GO) test ./internal/vault/... -v
	$(GO) test ./internal/module/... -v

# ---- Cleanup ----
clean:
	@rm -rf $(BUILD_DIR)
	@echo "=== Cleaned ==="
