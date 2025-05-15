# Testing ZSH Completion

Follow these steps to test the ZSH completion functionality:

## Supported Completions

The following flags support tab completion:

- `--backend`: Completes with "aws-bedrock" and "mock"
- `--model`: Completes with Claude model IDs
- `--aws-region`: Completes with AWS region names

## Quick Test During Development

1. Generate and source the completion script directly:
   ```zsh
   source <(./mcp completion zsh)
   ```

2. Test completions with:
   ```zsh
   # Test backend completion
   ./mcp --backend [TAB]
   # Should show: aws-bedrock mock

   # Test model completion
   ./mcp --model [TAB]
   # Should show the Claude model IDs
   
   # Test AWS region completion
   ./mcp --aws-region [TAB]
   # Should show AWS region IDs
   ```

## Permanent Installation

For permanent installation (as detailed in the README.md):

1. Make sure shell completion is enabled in your zsh:
   ```zsh
   echo "autoload -U compinit; compinit" >> ~/.zshrc
   ```

2. Create a completion directory and install the script:
   ```zsh
   mkdir -p ~/.zsh/completion
   ./mcp completion zsh > ~/.zsh/completion/_mcp
   ```

3. Add the completion directory to your fpath:
   ```zsh
   echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshrc
   ```

4. Restart your shell or source your .zshrc:
   ```zsh
   source ~/.zshrc
   ```

5. Test completions with:
   ```zsh
   mcp --backend [TAB]
   mcp --model [TAB]
   mcp --aws-region [TAB]
   ```

## Troubleshooting

If completions don't work:

1. Make sure you're using the latest version of the app with the built completions

2. Try generating and sourcing the completion script directly:
   ```zsh
   source <(./mcp completion zsh)
   ```

3. Verify the completion script was generated correctly:
   ```zsh
   ./mcp completion zsh | grep -A 5 -B 5 "backend\|model\|aws-region"
   ```

4. Ensure ZSH completion is properly enabled:
   ```zsh
   echo $fpath | grep -o completion
   ```

5. Re-run compinit to reload completions:
   ```zsh
   compinit
   ```
