.PHONY: build run test clean install

# Build info from root
VERSION ?= $(shell cat ../../VERSION 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X main.buildVersion=$(VERSION) -X main.buildCommit=$(COMMIT) -X main.buildDate=$(DATE)"

# Build palace CLI
build:
	@echo "Building palace CLI..."
	cd ../.. && go build $(LDFLAGS) -o palace ./cmd/palace
	@echo "Binary created at ../../palace"

# Run palace with arguments
run:
	cd ../.. && go run ./cmd/palace $(ARGS)

# Run tests
test:
	cd ../.. && go test -v ./cmd/palace/...

# Clean build artifacts
clean:
	rm -f ../../palace

# Install to GOPATH/bin
install:
	cd ../.. && go install $(LDFLAGS) ./cmd/palace

# Development run
dev:
	cd ../.. && go run ./cmd/palace serve --dev
