package mcpterm

import (
	"fmt"
	"os"

	"github.com/navicore/mcpterm-go/pkg/chat"
	"github.com/spf13/cobra"
)

var (
	// Command line flags
	configFile        string
	backendType       string
	modelID           string
	awsRegion         string
	awsProfile        string
	temperature       float64
	maxTokens         int
	contextSize       int
	systemPrompt      string
	systemPromptPath  string
	showSystemPrompt  bool
	mockMode          bool
	showTokenUsage    bool
	debugMode         bool
	enableTools       bool
	enabledCategories string // Comma-separated list of tool categories to enable

	// Context management flags
	enableContextMgmt  bool
	primaryModelID     string
	summarizerModelID  string
	maxContextTokens   int
	enableHierarchical bool
	enablePersistence  bool
)

var rootCmd = &cobra.Command{
	Use:   "mcp",
	Short: "A TUI chat application with vi-like motion support",
	Long: `A terminal-based chat application built with Go.
Features vi-like navigation, text formatting, and multiple chat backends.

Available backends:
- AWS Bedrock (Claude, etc.)
- Mock (for testing)

Use the --model flag to specify the model ID, such as:
- us.anthropic.claude-3-7-sonnet-20250219-v1:0
- anthropic.claude-3-sonnet-20240229-v1:0
- anthropic.claude-3-haiku-20240307-v1:0`,
	Run: func(cmd *cobra.Command, args []string) {
		// If --show-system-prompt is specified, display the prompt and exit
		if showSystemPrompt {
			// Get the default prompt
			defaultPrompt := chat.GetDefaultSystemPrompt()

			// If a custom prompt is specified, load it
			var customPrompt string
			var promptSource string

			if systemPrompt != "" {
				// Direct prompt from command line
				customPrompt = systemPrompt
				promptSource = "from command line --system-prompt flag"
			} else if systemPromptPath != "" {
				// Prompt from file
				data, err := os.ReadFile(systemPromptPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading system prompt file: %v\n", err)
					os.Exit(1)
				}
				customPrompt = string(data)
				promptSource = fmt.Sprintf("from file '%s'", systemPromptPath)
			}

			// Display the appropriate prompt
			if customPrompt != "" {
				fmt.Printf("Custom System Prompt %s:\n\n%s\n", promptSource, customPrompt)
			} else {
				fmt.Printf("Default System Prompt:\n\n%s\n", defaultPrompt)
			}
			return
		}

		// Start the TUI with the provided configuration
		startTUI()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Configuration flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is $HOME/.config/mcpterm/config.json)")

	// Backend-related flags
	rootCmd.PersistentFlags().StringVar(&backendType, "backend", "", "Backend type (aws-bedrock, mock)")
	rootCmd.PersistentFlags().StringVar(&modelID, "model", "", "Model ID (e.g., us.anthropic.claude-3-7-sonnet-20250219-v1:0)")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "aws-region", "", "AWS region for Bedrock")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "aws-profile", "", "AWS profile for Bedrock")
	rootCmd.PersistentFlags().BoolVar(&mockMode, "mock", false, "Use mock backend (for testing)")

	// Model parameters
	rootCmd.PersistentFlags().Float64Var(&temperature, "temperature", 0.7, "Temperature for sampling (0.0-1.0)")
	rootCmd.PersistentFlags().IntVar(&maxTokens, "max-tokens", 1000, "Maximum tokens in response")
	rootCmd.PersistentFlags().IntVar(&contextSize, "context-size", 20, "Number of messages to include in context")
	rootCmd.PersistentFlags().StringVar(&systemPrompt, "system-prompt", "", "System prompt for the conversation")
	rootCmd.PersistentFlags().StringVar(&systemPromptPath, "system-prompt-path", "", "Path to a file containing a system prompt")
	rootCmd.PersistentFlags().BoolVar(&showSystemPrompt, "show-system-prompt", false, "Show the current system prompt and exit")

	// UI flags
	rootCmd.PersistentFlags().BoolVar(&showTokenUsage, "show-tokens", false, "Show token usage statistics")

	// App flags
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode")

	// Tools flags
	rootCmd.PersistentFlags().BoolVar(&enableTools, "enable-tools", true, "Enable system tools for the LLM")
	rootCmd.PersistentFlags().StringVar(&enabledCategories, "enable-tool-categories", "filesystem",
		"Comma-separated list of tool categories to enable (filesystem, development, customer_support)")

	// Context management flags
	rootCmd.PersistentFlags().BoolVar(&enableContextMgmt, "enable-context", false, "Enable advanced context management")
	rootCmd.PersistentFlags().StringVar(&primaryModelID, "primary-model", "", "Primary model ID for regular interactions")
	rootCmd.PersistentFlags().StringVar(&summarizerModelID, "summarizer-model", "", "Summarizer model ID for context summarization")
	rootCmd.PersistentFlags().IntVar(&maxContextTokens, "max-context-tokens", 100000, "Maximum tokens for context window")
	rootCmd.PersistentFlags().BoolVar(&enableHierarchical, "hierarchical-context", true, "Enable hierarchical context structure")
	rootCmd.PersistentFlags().BoolVar(&enablePersistence, "persist-context", false, "Enable context persistence to disk")

	// Register flag completions for backend and model flags
	_ = rootCmd.RegisterFlagCompletionFunc("backend", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"aws-bedrock", "mock"}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = rootCmd.RegisterFlagCompletionFunc("model", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			"anthropic.claude-3-sonnet-20240229-v1:0",
			"anthropic.claude-3-haiku-20240307-v1:0",
			"anthropic.claude-3-opus-20240229-v1:0",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// Add AWS regions completion
	_ = rootCmd.RegisterFlagCompletionFunc("aws-region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-central-1",
			"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// Enable the built-in completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
