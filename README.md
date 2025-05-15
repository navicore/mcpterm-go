# MCPTerm

A terminal-based chat application built with Go. Features vi-like navigation, text formatting, and multiple chat backends.

## Features

- TUI chat application with vi-like motion support
- Multiple backend support (AWS Bedrock)
- Tool integration for filesystem and development operations

## Installation

TODO: Add installation instructions

## Usage

```bash
# Basic usage
mcpterm

# With AWS Bedrock backend
mcpterm --backend aws-bedrock --model us.anthropic.claude-3-7-sonnet-20250219-v1:0 --aws-region us-east-1

# Using mock mode for testing
mcpterm --mock
```

## Shell Completion

MCPTerm supports command completion for bash, zsh, fish, and PowerShell.

### ZSH Completion

#### Quick Setup (For Development)

To quickly enable ZSH completion for your current terminal session:

```zsh
# Load completions for current session only
source <(mcpterm completion zsh)
```

This method is ideal during development as it doesn't modify any files and only affects your current terminal session.

#### Permanent Installation

To permanently enable ZSH completion, follow these steps:

1. Make sure shell completion is enabled in your zsh:

   ```zsh
   echo "autoload -U compinit; compinit" >> ~/.zshrc
   ```

2. Generate the completion script and add it to your fpath:

   ```zsh
   # Create the completion directory if it doesn't exist
   mkdir -p ~/.zsh/completion
   
   # Generate and save the completion script
   mcpterm completion zsh > ~/.zsh/completion/_mcpterm
   ```

3. Add the completion directory to your fpath (add to ~/.zshrc):

   ```zsh
   echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshrc
   ```

4. Restart your shell or source your .zshrc:

   ```zsh
   source ~/.zshrc
   ```

Now you can use tab completion with the `mcpterm` command!

### Bash Completion

```bash
# For current session
source <(mcpterm completion bash)

# Install permanently (Linux)
mcpterm completion bash > /etc/bash_completion.d/mcpterm

# Install permanently (macOS with Homebrew)
mcpterm completion bash > $(brew --prefix)/etc/bash_completion.d/mcpterm
```

## Configuration

TODO: Add configuration details

## License

TODO: Add license information
