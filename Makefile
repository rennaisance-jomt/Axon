# Axon Makefile

# Build variables
BINARY_NAME=axon
VERSION=1.0.0
BUILD_DIR=./bin
GO=go
GOFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building Axon..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/axon

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/axon
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/axon
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe ./cmd/axon
	@echo "Build complete!"

# Run the binary
.PHONY: run
run: build
	@echo "Starting Axon..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with custom config
.PHONY: run-config
run-config: build
	./$(BUILD_DIR)/$(BINARY_NAME) --config config.yaml

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run human-readable tests
.PHONY: test
test:
	$(GO) run ./scripts/test_runner.go

# Run verbose tests
.PHONY: test-full
test-full:
	$(GO) test -v -race ./...

# Run tests with coverage
.PHONY: test-cover
test-cover:
	$(GO) test -cover ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...
	gofmt -w .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Development: run with live reload
.PHONY: dev
dev:
	@echo "Installing air..."
	$(GO) install github.com/air-verse/air@latest
	air

# Docker: build image
.PHONY: docker-build
docker-build:
	docker build -t axon:latest .

# Docker: run container
.PHONY: docker-run
docker-run:
	docker run -d -p 8020:8020 -v $(PWD)/data:/data axon:latest

# Docker: stop container
.PHONY: docker-stop
docker-stop:
	docker stop axon

# Help
.PHONY: help
help:
	@echo "Axon Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for all platforms"
	@echo "  run           - Build and run"
	@echo "  run-config    - Run with custom config"
	@echo "  deps          - Install dependencies"
	@echo "  test          - Run tests"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  clean         - Clean build artifacts"
	@echo "  dev           - Run with live reload"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-stop   - Stop Docker container"

