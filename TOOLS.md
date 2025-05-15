# MCPTerm Tool Support

MCPTerm includes powerful tools that Claude can use to help you interact with your local system and services. Tools are organized by categories to support different use cases.

## Tool Categories

### Filesystem Tools

These tools allow Claude to interact with your local macOS filesystem in a safe, read-only manner.

#### 1. `find` - File Search Tool

The find tool allows Claude to search for files and directories on your system.

**Capabilities:**
- Search in specified directories
- Filter by filename pattern (e.g., `*.go`)
- Filter by file type (regular files or directories)
- Limit search depth
- Search for files containing specific text
- Filter by file size and modification time

**Example prompts:**
- "Find all Go files in my home directory"
- "Search for configuration files that contain the word 'debug'"
- "List all directories under the current project"

#### 2. `file_read` - File Reading Tool

The file_read tool allows Claude to read the contents of files.

**Capabilities:**
- Read entire files
- Read specific line ranges with offset and limit parameters
- Handle various file types including text and code files

**Example prompts:**
- "Read my ~/.bashrc file"
- "Show me the first 50 lines of main.go"
- "What's in my TODO.md file?"

#### 3. `directory_list` - Directory Listing Tool

The directory_list tool allows Claude to see the contents of directories.

**Capabilities:**
- List files and directories in a specified path
- Filter results by filename pattern
- Get file metadata (size, is directory)

**Example prompts:**
- "List the files in my Downloads folder"
- "Show me all the markdown files in the current project"
- "What's in my ~/Documents directory?"

### Development Tools

These tools allow Claude to assist with local development tasks beyond just reading files.

#### Available Tools:
- `shell` - Execute specific shell commands on macOS with timeout protection
- `file_write` - Create or modify files on macOS (create, overwrite, append)
- `patch` - Create and apply patches to files on macOS (preferred method for file modifications)
- `diff` - Compare two files or strings and show differences

#### Planned Tools:
- `git` - Perform basic git operations

#### Example: Using file_write to create a new file

```
YOU: Can you create a simple "Hello World" Python script for me?

CLAUDE: I'd be happy to create a simple "Hello World" Python script for you. 
Let me write that to a file.

I need to use the 'file_write' tool to help answer your question. (Tool 'file_write' executed successfully)

I've created a Python script called "hello_world.py" with the following content:

```python
#!/usr/bin/env python3

def main():
    print("Hello, World!")
    print("Welcome to Python programming!")

if __name__ == "__main__":
    main()
```

You can run this script with:

```bash
python hello_world.py
```

or make it executable with:

```bash
chmod +x hello_world.py
./hello_world.py
```
```

#### Example: Using diff to compare files

```
YOU: Can you compare the differences between app.js and app.js.backup?

CLAUDE: I'll compare the two files for you.

I need to use the 'diff' tool to help answer your question. (Tool 'diff' executed successfully)

Here are the differences between app.js and app.js.backup:

```diff
--- app.js
+++ app.js
@@ -10,7 +10,7 @@
 const port = process.env.PORT || 3000;
 
 // Configure database connection
-const dbConfig = require('./config/database');
+const dbConfig = require('./config/database.prod');
 
 // Initialize middleware
 app.use(express.json());
@@ -22,6 +22,9 @@
 app.use('/api/users', require('./routes/users'));
 app.use('/api/auth', require('./routes/auth'));
 
+// Add analytics middleware
+app.use(require('./middleware/analytics'));
+
 // Error handling middleware
 app.use((err, req, res, next) => {
   console.error(err.stack);
```

The main changes are:
1. The database configuration is now using 'database.prod.js' instead of 'database.js'
2. An analytics middleware has been added to the current app.js
3. All other code remains identical between the two files

Would you like me to explain any specific part of these changes?
```

#### Example: Using patch to modify a file

```
YOU: The ReadMe.md file in my project needs updating to include installation instructions. Can you help?

CLAUDE: I'll help you update the README.md to include installation instructions. Let me first check the current content of the file.

I need to use the 'file_read' tool to help answer your question. (Tool 'file_read' executed successfully)

Now I'll create a patch to add installation instructions to your README.md.

I need to use the 'patch' tool to help answer your question. (Tool 'patch' executed successfully)

I've created a patch for your README.md file that adds installation instructions. Here's what will change:

```diff
@@ -6,6 +6,17 @@
 
 This project provides a simple REST API for managing tasks.
 
+## Installation
+
+Follow these steps to install and set up the project:
+
+```bash
+# Clone the repository
+git clone https://github.com/username/project.git
+cd project
+
+# Install dependencies
+npm install
+```
+
 ## Usage
 
 Start the server:
```

Would you like me to apply this patch to your README.md file?

YOU: Yes, please apply the patch.

CLAUDE: I'll apply the patch to your README.md file.

I need to use the 'patch' tool to help answer your question. (Tool 'patch' executed successfully)

I've successfully applied the patch to your README.md file. The installation instructions have been added.

The updated README.md now includes:
1. A new "Installation" section
2. Instructions for cloning the repository
3. Steps to install the dependencies

Is there anything else you'd like to update in your README.md file?
```

### Customer Support Tools (Planned)

These tools will allow Claude to access and manage SaaS application data for customer support scenarios.

#### Planned Tools:
- `user_find` - Search for users based on criteria
- `user_read` - View detailed user information
- `user_update` - Update user settings or details
- `tenant_read` - View tenant or organization information
- `ticket_search` - Find support tickets
- `ticket_update` - Update ticket status or details

## Enabling/Disabling Tools

Tools are enabled by default, but you can disable them all or by category:

```bash
# Disable all tools
./mcp --enable-tools=false

# Enable all tools (default)
./mcp --enable-tools=true

# Enable only specific tool categories (planned feature)
./mcp --enable-tool-categories=filesystem,development
```

## Security Considerations

- **Access Level:** Tools have access only to resources that your user account can access.
- **Read-Only Default:** The default filesystem tools are read-only and cannot modify your system.
- **Tool Usage Visibility:** You can see when Claude is using a tool - the application displays a message showing which tool is being used.
- **Usage Limits:** Tool execution is limited to 10 tool calls per message to prevent excessive resource usage.
- **Category Permissions:** Different tool categories can have different permission levels (planned feature).

## Example Conversations

**Example 1: Finding and reading files**

```
YOU: I need to find all Go test files in this project and tell me what they do.

CLAUDE: I need to use the 'find' tool to help answer your question. (Tool 'find' executed successfully)

Let me search for all Go test files and analyze them:

1. First, I found these test files:
   - pkg/chat/chat_test.go

Let me read the test file to understand what it does:

I need to use the 'file_read' tool to help answer your question. (Tool 'file_read' executed successfully)

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

CLAUDE: I need to use the 'directory_list' tool to help answer your question. (Tool 'directory_list' executed successfully)

# MCPTerm Project Structure

The MCPTerm project is organized as follows:

## Root Level
- `cmd/`: Command-line interface code
  - `mcp/`: Main application entry point
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

## Implementation Architecture

The tool system has been implemented with the following components:

- `pkg/tools/tools.go`: Core tool implementations
- `pkg/chat/tools.go`: Integration with the chat service
- Updates to configuration and command-line flags

### Adding New Tool Categories

To support the hierarchical tool structure:

1. **Tool Registration**: Each tool belongs to a category and registers itself with the tool manager
2. **Permission Control**: Categories can have different permission settings
3. **Dynamic Loading**: Tool categories can be enabled/disabled separately via configuration
4. **API Consistency**: All tools follow a consistent interface pattern regardless of category

The implementation ensures a smooth conversation flow even when tools are used, maintaining context across multiple tool uses.
