package mcpterm

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Command line flags
	configFile      string
	backendType     string
	modelID         string
	awsRegion       string
	awsProfile      string
	temperature     float64
	maxTokens       int
	contextSize     int
	systemPrompt    string
	mockMode        bool
	showTokenUsage  bool
	debugMode       bool
	enableTools     bool
)

var rootCmd = &cobra.Command{
	Use:   "mcpterm",
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

	// UI flags
	rootCmd.PersistentFlags().BoolVar(&showTokenUsage, "show-tokens", false, "Show token usage statistics")

	// App flags
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode")

	// Tools flags
	rootCmd.PersistentFlags().BoolVar(&enableTools, "enable-tools", true, "Enable system tools for the LLM (find, file_read, directory_list)")
}
