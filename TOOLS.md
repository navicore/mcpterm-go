# MCPTerm Tool Support

MCPTerm now includes powerful system tools that Claude can use to help you interact with your local filesystem. These tools allow Claude to search for files, read file contents, and list directory contents.

## Available Tools

### 1. `find` - File Search Tool

The find tool allows Claude to search for files and directories on your system.

**Capabilities:**
- Search in specified directories
- Filter by filename pattern (e.g., `*.go`)
- Filter by file type (regular files or directories)
- Limit search depth
- Search for files containing specific text

**Example prompts:**
- "Find all Go files in my home directory"
- "Search for configuration files that contain the word 'debug'"
- "List all directories under the current project"

### 2. `file_read` - File Reading Tool

The file_read tool allows Claude to read the contents of files.

**Capabilities:**
- Read entire files
- Read specific line ranges with offset and limit parameters
- Handle various file types including text and code files

**Example prompts:**
- "Read my ~/.bashrc file"
- "Show me the first 50 lines of main.go"
- "What's in my TODO.md file?"

### 3. `directory_list` - Directory Listing Tool

The directory_list tool allows Claude to see the contents of directories.

**Capabilities:**
- List files and directories in a specified path
- Filter results by filename pattern
- Get file metadata (size, is directory)

**Example prompts:**
- "List the files in my Downloads folder"
- "Show me all the markdown files in the current project"
- "What's in my ~/Documents directory?"

## Enabling/Disabling Tools

Tools are enabled by default, but you can disable them if needed:

```bash
# Disable tools
./mcpterm --enable-tools=false

# Enable tools (default)
./mcpterm --enable-tools=true
```

## Security Considerations

- Tools have access to your local filesystem, so they can read any files that your user account has permission to access.
- Tools cannot write to your filesystem (they are read-only).
- You can see when Claude is using a tool - the application will display a message like "I need to use the 'find' tool to help answer your question."
- Tool execution is limited to 10 tool calls per message to prevent excessive resource usage.

## Example Conversations

**Example 1: Finding and reading files**

```
YOU: I need to find all Go test files in this project and tell me what they do.

CLAUDE: I need to use the 'find' tool to help answer your question.
(Tool 'find' executed successfully)

Let me search for all Go test files and analyze them:

1. First, I found these test files:
   - pkg/chat/chat_test.go

Let me read the test file to understand what it does:

I need to use the 'file_read' tool to help answer your question.
(Tool 'file_read' executed successfully)

The pkg/chat/chat_test.go file contains unit tests for the chat package, specifically testing:

1. The SimpleMessage functionality
2. Message history management
3. Proper response generation based on user input

The tests verify that the chat service correctly:
- Stores messages in history
- Generates appropriate responses based on input content
- Handles commands like "help", "bye", etc.

This is the main test file for the chat functionality which ensures the core messaging features work correctly.
```

**Example 2: Exploring directory structure**

```
YOU: What's the structure of this project? 

CLAUDE: I need to use the 'directory_list' tool to help answer your question.
(Tool 'directory_list' executed successfully)

# MCPTerm Project Structure

The MCPTerm project is organized as follows:

## Root Level
- `cmd/`: Command-line interface code
  - `mcpterm/`: Main application entry point
- `pkg/`: Core packages
  - `backend/`: LLM backend implementations
  - `chat/`: Chat service implementation
  - `config/`: Configuration handling
  - `tools/`: System tool implementations
  - `ui/`: Terminal UI components
- `main.go`: Application entry point
- `go.mod` & `go.sum`: Go module definition and dependencies
- Documentation: `README.md`, `LICENSE`, etc.

The project follows a clean modular architecture with separation of concerns:
- Backend module handles different LLM providers
- Chat module manages conversations
- UI module handles the terminal interface
- Tools module provides system tool capabilities

This is a well-structured Go application following modern best practices.
```

## Implementation Details

The tool system has been implemented with the following components:

- `pkg/tools/tools.go`: Core tool implementations
- `pkg/chat/tools.go`: Integration with the chat service
- Updates to configuration and command-line flags

The implementation ensures a smooth conversation flow even when tools are used, maintaining context across multiple tool uses.