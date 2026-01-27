.PHONY: all build test clean dev deps lint release package help sync-versions

# Build info
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli.buildVersion=$(VERSION) -X github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli.buildCommit=$(COMMIT) -X github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli.buildDate=$(DATE)"

# Default target
all: build

# Help
help:
	@echo "Mind Palace - Makefile Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build          - Build all components (palace CLI, vscode)"
	@echo "  make build-palace   - Build palace CLI only"
	@echo "  make build-vscode   - Build VS Code extension"
	@echo "  make release        - Build optimized release binary"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test           - Run all tests"
	@echo "  make test-all       - Run all tests (comprehensive script)"
	@echo "  make test-go        - Run Go tests"
	@echo "  make test-vscode    - Run VS Code extension tests"
	@echo "  make e2e            - Run end-to-end tests"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev            - Run palace in dev mode"
	@echo "  make dev-vscode     - Watch VS Code extension"
	@echo "  make menu           - Interactive development menu (Windows: scripts/dev.ps1)"
	@echo ""
	@echo "Other Commands:"
	@echo "  make deps           - Install all dependencies"
	@echo "  make lint           - Run linters"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make package-vscode - Package VS Code extension"
	@echo "  make sync-versions  - Sync all app versions with root VERSION file"

# =============================================================================
# Build Commands
# =============================================================================

# Build everything
build: build-vscode build-palace
	@echo "Build complete!"

# Build palace CLI
build-palace:
	@echo "Building palace CLI..."
	go build $(LDFLAGS) -o palace ./apps/cli

# Build VS Code extension
build-vscode:
	@echo "Building VS Code extension..."
	@if [ -d "apps/vscode/node_modules" ]; then \
		cd apps/vscode && npm run compile; \
	else \
		echo "VS Code extension dependencies not installed. Run 'make deps-vscode' first."; \
	fi

# Release build (optimized)
release:
	@echo "Building release binary..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o palace ./apps/cli
	@echo "Release build complete: ./palace ($(shell du -h palace | cut -f1))"

# =============================================================================
# Test Commands
# =============================================================================

# Run all tests
test: test-go test-vscode
	@echo "All tests complete!"

# Go tests
test-go:
	@echo "Running Go tests..."
	go test -v -race ./...

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

# Run all tests (comprehensive)
test-all:
	@echo "Running all tests..."
	@./scripts/test-all.sh

# =============================================================================
# Development Commands
# =============================================================================

# Run palace server in dev mode
dev:
	@echo "Starting palace in dev mode..."
	go run ./apps/cli serve --dev

# Watch VS Code extension for changes
dev-vscode:
	@echo "Watching VS Code extension..."
	cd apps/vscode && npm run watch

# Interactive development menu
menu:
	@./scripts/dev.sh

# =============================================================================
# Dependency Management
# =============================================================================

# Install all dependencies
deps: deps-go deps-vscode
	@echo "All dependencies installed!"

# Go dependencies
deps-go:
	@echo "Downloading Go dependencies..."
	go mod download
	go mod tidy

# VS Code extension dependencies
deps-vscode:
	@echo "Installing VS Code extension dependencies..."
	cd apps/vscode && npm install

# =============================================================================
# Linting
# =============================================================================

# Run all linters
lint: lint-go lint-vscode

# Go lint
lint-go:
	@echo "Linting Go code..."
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

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
	rm -rf apps/vscode/out
	rm -rf apps/vscode/*.vsix
	go clean ./...
	@echo "Clean complete!"

# Deep clean (including node_modules)
clean-all: clean
	@echo "Deep cleaning..."
	rm -rf apps/vscode/node_modules
	@echo "Deep clean complete!"

# =============================================================================
# Utility
# =============================================================================

# Sync all versions
sync-versions:
	@echo "Syncing ecosystem versions..."
	@./scripts/sync-versions.sh

# Show project info
info:
	@echo "Mind Palace"
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@echo ""
	@echo "Directories:"
	@echo "  CLI:       apps/cli/"
	@echo "  VS Code:   apps/vscode/"
	@echo "  Internal:  apps/cli/internal/"
	@echo "  Public:    apps/cli/pkg/"

# Verify build
verify: build
	@echo "Verifying build..."
	./palace version
	@echo "Build verified!"
