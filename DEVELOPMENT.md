# Development Guide

This document provides information for developers working on the MCPTerm project.

## Building and Running

We provide two methods for managing development tasks: a Makefile and a Go-based task runner.

### Using Makefile

```bash
make          # Build the application
make run      # Build and run the application
make test     # Run tests
make lint     # Run linters
make fmt      # Format code
make clean    # Clean build artifacts
make all      # Format, build, test, and lint
make tools    # Install development tools
```

### Using Go Task Runner

```bash
go run task.go           # Build the application
go run task.go run       # Build and run the application
go run task.go test      # Run tests
go run task.go lint      # Run linters
go run task.go fmt       # Format code
go run task.go clean     # Clean build artifacts
go run task.go all       # Format, build, test, and lint
go run task.go tools     # Install development tools
```

## Project Structure

```
mcpterm-go/
├── cmd/                  # Command-line application code
│   └── mcpterm/          # Main application commands
│       ├── root.go       # Root cobra command
│       └── tui.go        # TUI initialization
├── pkg/                  # Library packages
│   ├── chat/             # Chat service implementation
│   │   ├── chat.go       # Chat service interfaces and implementation
│   │   └── chat_test.go  # Unit tests for chat service
│   └── ui/               # User interface components
│       └── tui.go        # TUI components and logic
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── main.go               # Main entry point
├── Makefile              # Make targets for development
├── task.go               # Go-based task runner
└── tools.go              # Tool dependencies
```

## Development Workflow

1. Make changes to code
2. Format code: `go run task.go fmt`
3. Run tests: `go run task.go test`
4. Run lint checks: `go run task.go lint`
5. Build and run the application: `go run task.go run`

## Adding Features

When adding new features:

1. Update appropriate package in `pkg/`
2. Add tests for new functionality
3. Update UI components if needed
4. Run all checks with `go run task.go all`

## Testing

We use Go's standard testing package. Run tests with:

```bash
go run task.go test
```

## Linting

We use `golangci-lint` for code linting. Install it with:

```bash
go run task.go tools
```

Then run the linter with:

```bash
go run task.go lint
```