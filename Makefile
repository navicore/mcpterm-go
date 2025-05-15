.PHONY: build run run-claude test lint clean all default

# Binary name
BINARY_NAME=mcp

# Project details
PROJECT_NAME=mcpterm-go
MAIN_PACKAGE=.

# Build settings
GO=$(shell which go)
GOFLAGS=-v
BUILD_DIR=./build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
default: build

# Build the application
build:
	@echo "Building $(PROJECT_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run the application
run: build
	@echo "Running $(PROJECT_NAME)..."
	./$(BINARY_NAME)

# Run the application with AWS Bedrock Claude 3.7 Sonnet
run-claude: build
	@echo "Running $(PROJECT_NAME) with AWS Bedrock Claude 3.7 Sonnet..."
	@echo "Note: Make sure AWS credentials are configured properly"
	./$(BINARY_NAME) --backend aws-bedrock --model us.anthropic.claude-3-7-sonnet-20250219-v1:0 --aws-region us-west-2

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run linting using golangci-lint if available, otherwise use go vet
lint:
	@if command -v golangci-lint > /dev/null; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "Running go vet (golangci-lint not found)..."; \
		$(GO) vet ./...; \
		echo "For more comprehensive linting, please install golangci-lint"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Check if code is correctly formatted
fmt-check:
	@echo "Checking if code is formatted..."
	@if [ -n "$$($(GO) fmt ./...)" ]; then \
		echo "Code is not formatted, run 'make fmt'"; \
		exit 1; \
	else \
		echo "Code is properly formatted"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Install development tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build, test, and lint
all: fmt build test lint

# Help command
help:
	@echo "Available commands:"
	@echo "  make              - Build the application"
	@echo "  make build        - Build the application"
	@echo "  make run          - Build and run the application"
	@echo "  make run-claude   - Run with AWS Bedrock Claude 3.7 Sonnet"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make fmt          - Format code"
	@echo "  make fmt-check    - Check if code is formatted"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make all          - Format, build, test, and lint"
	@echo "  make tools        - Install development tools"
	@echo "  make help         - Display this help message"
