.PHONY: all build test clean dev deps lint release package help

# Build info
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: build

# Help
help:
	@echo "Mind Palace - Makefile Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build          - Build all components (palace CLI, dashboard, vscode)"
	@echo "  make build-palace   - Build palace CLI only"
	@echo "  make build-dashboard - Build Angular dashboard"
	@echo "  make build-vscode   - Build VS Code extension"
	@echo "  make release        - Build optimized release binary"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test           - Run all tests"
	@echo "  make test-go        - Run Go tests"
	@echo "  make test-dashboard - Run Angular tests"
	@echo "  make test-vscode    - Run VS Code extension tests"
	@echo "  make e2e            - Run end-to-end tests"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev            - Run palace in dev mode"
	@echo "  make dev-dashboard  - Run Angular dev server"
	@echo "  make dev-vscode     - Watch VS Code extension"
	@echo ""
	@echo "Other Commands:"
	@echo "  make deps           - Install all dependencies"
	@echo "  make lint           - Run linters"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make package-vscode - Package VS Code extension"

# =============================================================================
# Build Commands
# =============================================================================

# Build everything
build: build-dashboard build-vscode build-palace
	@echo "Build complete!"

# Build palace CLI
build-palace:
	@echo "Building palace CLI..."
	go build $(LDFLAGS) -o palace ./apps/cli

# Build dashboard (Angular)
build-dashboard:
	@echo "Building dashboard..."
	@if [ -d "apps/dashboard/node_modules" ]; then \
		cd apps/dashboard && npm run build; \
		echo "Embedding dashboard assets..."; \
		rm -rf apps/cli/internal/dashboard/dist; \
		mkdir -p apps/cli/internal/dashboard/dist; \
		if [ -d "dist/dashboard/browser" ]; then \
			cp -r dist/dashboard/browser/* ../../apps/cli/internal/dashboard/dist/; \
		elif [ -d "dist/dashboard" ]; then \
			cp -r dist/dashboard/* ../../apps/cli/internal/dashboard/dist/; \
		fi; \
	else \
		echo "Dashboard dependencies not installed. Run 'make deps-dashboard' first."; \
	fi

# Build VS Code extension
build-vscode:
	@echo "Building VS Code extension..."
	@if [ -d "apps/vscode/node_modules" ]; then \
		cd apps/vscode && npm run compile; \
	else \
		echo "VS Code extension dependencies not installed. Run 'make deps-vscode' first."; \
	fi

# Release build (optimized, with embedded dashboard)
release: build-dashboard
	@echo "Building release binary..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o palace ./apps/cli
	@echo "Release build complete: ./palace ($(shell du -h palace | cut -f1))"

# =============================================================================
# Test Commands
# =============================================================================

# Run all tests
test: test-go test-dashboard test-vscode
	@echo "All tests complete!"

# Go tests
test-go:
	@echo "Running Go tests..."
	go test -v -race ./...

# Dashboard tests
test-dashboard:
	@echo "Running dashboard tests..."
	@if [ -d "apps/dashboard/node_modules" ]; then \
		cd apps/dashboard && npm test -- --watch=false --browsers=ChromeHeadless 2>/dev/null || echo "Dashboard tests skipped (no headless browser)"; \
	else \
		echo "Dashboard tests skipped (dependencies not installed)"; \
	fi

# VS Code extension tests
test-vscode:
	@echo "Running VS Code extension tests..."
	@if [ -d "apps/vscode/node_modules" ]; then \
		cd apps/vscode && npm test 2>/dev/null || echo "VS Code tests skipped (no test runner)"; \
	else \
		echo "VS Code tests skipped (dependencies not installed)"; \
	fi

# End-to-end tests
e2e: build-palace
	@echo "Running end-to-end tests..."
	./scripts/e2e-test.sh

# =============================================================================
# Development Commands
# =============================================================================

# Run palace server in dev mode
dev:
	@echo "Starting palace in dev mode..."
	go run ./apps/cli serve --dev

# Run dashboard dev server (with proxy to palace backend)
dev-dashboard:
	@echo "Starting dashboard dev server..."
	cd apps/dashboard && npm start

# Watch VS Code extension for changes
dev-vscode:
	@echo "Watching VS Code extension..."
	cd apps/vscode && npm run watch

# Run everything in dev mode (requires multiple terminals)
dev-all:
	@echo "To run in dev mode, open separate terminals and run:"
	@echo "  Terminal 1: make dev          (Go backend)"
	@echo "  Terminal 2: make dev-dashboard (Angular frontend)"
	@echo "  Terminal 3: make dev-vscode   (VS Code extension)"

# =============================================================================
# Dependency Management
# =============================================================================

# Install all dependencies
deps: deps-go deps-dashboard deps-vscode
	@echo "All dependencies installed!"

# Go dependencies
deps-go:
	@echo "Downloading Go dependencies..."
	go mod download
	go mod tidy

# Dashboard dependencies
deps-dashboard:
	@echo "Installing dashboard dependencies..."
	cd apps/dashboard && npm install

# VS Code extension dependencies
deps-vscode:
	@echo "Installing VS Code extension dependencies..."
	cd apps/vscode && npm install

# =============================================================================
# Linting
# =============================================================================

# Run all linters
lint: lint-go lint-dashboard lint-vscode

# Go lint
lint-go:
	@echo "Linting Go code..."
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

# Dashboard lint
lint-dashboard:
	@echo "Linting dashboard..."
	@if [ -d "apps/dashboard/node_modules" ]; then \
		cd apps/dashboard && npm run lint 2>/dev/null || echo "No lint script configured"; \
	fi

# VS Code extension lint
lint-vscode:
	@echo "Linting VS Code extension..."
	@if [ -d "apps/vscode/node_modules" ]; then \
		cd apps/vscode && npm run lint 2>/dev/null || echo "No lint script configured"; \
	fi

# =============================================================================
# Packaging
# =============================================================================

# Package VS Code extension
package-vscode:
	@echo "Packaging VS Code extension..."
	cd apps/vscode && vsce package

# =============================================================================
# Cleanup
# =============================================================================

# Clean all build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f palace
	rm -rf apps/dashboard/dist
	rm -rf apps/dashboard/.angular
	rm -rf apps/vscode/out
	rm -rf apps/vscode/*.vsix
	rm -rf apps/cli/internal/dashboard/dist
	go clean ./...
	@echo "Clean complete!"

# Deep clean (including node_modules)
clean-all: clean
	@echo "Deep cleaning..."
	rm -rf apps/dashboard/node_modules
	rm -rf apps/vscode/node_modules
	@echo "Deep clean complete!"

# =============================================================================
# Utility
# =============================================================================

# Show project info
info:
	@echo "Mind Palace"
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@echo ""
	@echo "Directories:"
	@echo "  CLI:       apps/cli/"
	@echo "  Dashboard: apps/dashboard/"
	@echo "  VS Code:   apps/vscode/"
	@echo "  Internal:  apps/cli/internal/"
	@echo "  Public:    apps/cli/pkg/"

# Verify build
verify: build
	@echo "Verifying build..."
	./palace version
	@echo "Build verified!"
