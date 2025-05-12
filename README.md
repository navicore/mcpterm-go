# MCPTerm

# UNDER CONSTRUCTION

# UNDER CONSTRUCTION

# UNDER CONSTRUCTION

A terminal-based chat application with vi-like motion support built in Go, featuring multiple chat backends including AWS Bedrock for Claude models.

## Overview

MCPTerm provides a terminal user interface for chat interactions with AI assistants, complete with vi-like navigation, markdown rendering, and a modular backend system that supports different AI providers.

## Features

- **Terminal UI**: Clean, responsive interface that works in any terminal
- **Full Vi Editing**: Complete vi-like editing with normal, insert and visual modes, with multi-line selection
- **Markdown Support**: Format messages with rich markdown
- **Input History**: Navigate through previous messages with up/down keys
- **Mode Indicators**: Visual indication of current mode (normal/insert)
- **Command System**: Simple command system for features and information
- **Multiple Backends**: Support for different AI providers:
  - **AWS Bedrock**: Integration with Claude models
  - **Mock Backend**: For testing without API access
- **Configuration**: Flexible configuration via file or command-line flags

## Requirements

- Go 1.21+ (though should work with Go 1.18+)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/mcpterm-go.git
cd mcpterm-go
```

2. Build the application (choose one):
```bash
# Using go directly
go build -o mcpterm

# Using Makefile
make build

# Using task runner
go run task.go build
```

3. Run the application (choose one):
```bash
# Run with default mock backend
./mcpterm

# Using AWS Bedrock for Claude
./mcpterm --backend aws-bedrock --model us.anthropic.claude-3-7-sonnet-20250219-v1:0 --aws-region us-west-2

# Using Makefile
make run

# Using task runner
go run task.go run
```

## Development

The project includes both a Makefile and a Go-based task runner to help with common development tasks:

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

## Backend Configuration

MCPTerm supports multiple backends for interacting with various AI services:

### AWS Bedrock

To use AWS Bedrock with Claude models:

```bash
./mcpterm --backend aws-bedrock --model <model-id> --aws-region <region> [--aws-profile <profile>]
```

Available Claude model IDs:
- `us.anthropic.claude-3-7-sonnet-20250219-v1:0` - Claude 3.7 Sonnet (Recommended)
- `anthropic.claude-3-sonnet-20240229-v1:0` - Claude 3 Sonnet
- `anthropic.claude-3-haiku-20240307-v1:0` - Claude 3 Haiku
- `anthropic.claude-3-opus-20240229-v1:0` - Claude 3 Opus

AWS authentication is handled via the standard AWS SDK mechanisms:
- AWS credentials file
- Environment variables
- IAM roles
- AWS profiles (specify with `--aws-profile`)

### Mock Backend

For testing without API access:

```bash
./mcpterm --mock
```

Or explicitly:

```bash
./mcpterm --backend mock
```

The mock backend provides pre-programmed responses based on keywords in your messages.

### Configuration Options

Common configuration flags:
- `--config <path>` - Path to config file (default: `$HOME/.config/mcpterm/config.json`)
- `--temperature <float>` - Temperature for sampling (default: 0.7)
- `--max-tokens <int>` - Maximum tokens in response (default: 1000)
- `--context-size <int>` - Number of messages to include in context (default: 20)
- `--system-prompt <string>` - Custom system prompt
- `--show-tokens` - Show token usage statistics

## Usage

### Basic Commands

- Type a message and press `Enter` to send
- Press `Tab` to switch focus between chat history and input
- Press `Esc` to enter normal mode for vi-like editing
- Use `j`/`k` in normal mode to navigate input history
- Press `Ctrl+c` to quit
- Press `Ctrl+h` to toggle help display

### Vi Editing Capabilities

#### Input Editing (Normal Mode)

| Key | Action |
|-----|--------|
| `h`/`l` | Move cursor left/right |
| `0`/`$` | Move to beginning/end of line |
| `w`/`b` | Move forward/backward by word |
| `i`/`a` | Enter insert mode at/after cursor |
| `A` | Insert at end of line |
| `x` | Delete character under cursor |
| `dd` | Delete entire line |
| `yy` | Yank (copy) entire line |
| `p`/`P` | Paste after/before cursor |
| `j`/`k` | Navigate history down/up |
| `v` | Enter visual mode for selection |
| `Esc` | Return to normal mode |

#### Chat Viewport Navigation and Visual Selection

| Key | Action |
|-----|--------|
| `j` | Scroll down one line |
| `k` | Scroll up one line |
| `g` | Go to top of chat history |
| `G` | Go to bottom of chat history |
| `d` | Scroll half-page down |
| `u` | Scroll half-page up |
| `v` | Enter visual mode for multi-line selection |

The viewport has a visible cursor and supports both navigation and selection:

| Key | Action in Viewport Navigation |
|-----|------------------------------|
| `h`/`l` | Move cursor left/right |
| `j`/`k` | Move cursor up/down and scroll |
| `0`/`$` | Move cursor to beginning/end of line |
| `g`/`G` | Go to top/bottom of viewport |
| `v` | Enter visual selection mode |

Visual mode in viewport allows for powerful multi-line text selection:

| Key | Action in Viewport Visual Mode |
|-----|--------------------------------|
| `h`/`l` | Extend selection left/right |
| `j`/`k` | Extend selection down/up |
| `0`/`$` | Move cursor to begin/end of line while selecting |
| `y` | Copy selected text to clipboard |
| `Esc` | Exit visual mode (maintains cursor position) |

### Chat Commands

Type these commands to see different responses:

- `help` - Show available commands
- `features` - Show application features
- `vi` or `vim` - Show navigation help
- `markdown` - Show markdown formatting examples

## Technologies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - AWS API access
- [AWS Bedrock Runtime](https://github.com/aws/aws-sdk-go-v2/tree/main/service/bedrockruntime) - AWS Bedrock API

## License

MIT
