package mcpterm

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/chat"
	"github.com/navicore/mcpterm-go/pkg/config"
	"github.com/navicore/mcpterm-go/pkg/ui"
)

func startTUI() {
	// Load configuration
	cfg, err := loadAndMergeConfig()
	if err != nil {
		fmt.Printf("Warning: Could not load configuration: %v\nUsing defaults\n", err)
		// Use default config if loading fails
		cfg = config.DefaultConfig()
	}
	
	// Create chat service
	chatOptions := cfg.GetChatOptions()
	chatService, err := chat.NewChatService(chatOptions)
	if err != nil {
		fmt.Printf("Error initializing chat service: %v\n", err)
		os.Exit(1)
	}
	
	// Defer closing the chat service
	defer chatService.Close()
	
	// Get backend info for welcome message
	backendName, modelID := chatService.GetBackendInfo()

	// Initialize the TUI model
	m := ui.NewModel()
	m.SetChatService(chatService)
	
	// Add welcome messages with markdown formatting
	m.AddMessage(ui.Message{
		Username: "System",
		Content:  fmt.Sprintf("# Welcome to MCPTerm!\n\nA **terminal-based chat interface** with vi-like navigation.\n\nBackend: **%s**\nModel: **%s**", backendName, modelID),
		IsUser:   false,
	})
	
	m.AddMessage(ui.Message{
		Username: "System",
		Content: "## Vi Editing\n\n" +
			"This app supports vi editing modes:\n\n" +
			"Press `Esc` to enter normal mode where you can use:\n" +
			"- Movement: `h`/`l` (left/right), `0`/`$` (start/end of line), `w`/`b` (word forward/back)\n" +
			"- Editing: `i`/`a` (insert/append), `x` (delete char), `dd`/`yy` (delete/yank line)\n" +
			"- History: `j`/`k` (browse history down/up)\n" +
			"- Clipboard: `y` in visual mode copies text that can be pasted with `p`\n\n" +
			"Press `Tab` to switch focus between chatbot conversation and input field.",
		IsUser: false,
	})
	
	m.AddMessage(ui.Message{
		Username: "System",
		Content: "## Viewport Navigation\n\n" +
			"When the viewport is focused (press `Tab` to switch):\n" +
			"- Use vi motions: `j`/`k` to scroll, `g`/`G` for top/bottom\n" +
			"- `d`/`u` for half-page down/up\n" +
			"- `0`/`$` for far left/right\n\n" +
			"*All messages support markdown formatting!*\n\n" +
			"Type `help` for available commands.",
		IsUser: false,
	})
	
	// Run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

// loadAndMergeConfig loads the configuration file and merges it with command line flags
func loadAndMergeConfig() (config.Config, error) {
	// Load config from file
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return cfg, err
	}
	
	// Override with command line flags if provided
	if mockMode {
		cfg.Chat.BackendType = "mock"
		cfg.Chat.ModelID = "mock"
	} else if backendType != "" {
		cfg.Chat.BackendType = backendType
	}
	
	if modelID != "" {
		cfg.Chat.ModelID = modelID
	}
	
	// If model is specified but no backend, set appropriate backend
	if modelID != "" && backendType == "" {
		// Detect backend from model ID
		if strings.Contains(modelID, "claude") ||
		   strings.Contains(modelID, "anthropic") ||
		   strings.HasPrefix(modelID, "us.anthropic") {
			cfg.Chat.BackendType = "aws-bedrock"
		}
	}
	
	// If neither backend nor model is specified, but AWS region is, use AWS Bedrock
	if backendType == "" && modelID == "" && awsRegion != "" {
		cfg.Chat.BackendType = "aws-bedrock"
		// Use Claude 3.7 Sonnet as default model for Bedrock
		cfg.Chat.ModelID = backend.ModelClaude37Sonnet
	}
	
	// AWS specific flags
	if awsRegion != "" {
		cfg.Chat.AWS.Region = awsRegion
	}
	if awsProfile != "" {
		cfg.Chat.AWS.Profile = awsProfile
	}
	
	// Model parameters
	if temperature != 0.7 { // Check against default to see if user specified
		cfg.Chat.Temperature = temperature
	}
	if maxTokens != 1000 { // Check against default to see if user specified
		cfg.Chat.MaxTokens = maxTokens
	}
	if contextSize != 20 { // Check against default to see if user specified
		cfg.Chat.ContextWindowSize = contextSize
	}
	if systemPrompt != "" {
		cfg.Chat.SystemPrompt = systemPrompt
	}
	
	// UI flags
	if showTokenUsage {
		cfg.UI.ShowTokenUsage = true
	}
	
	// App flags
	if debugMode {
		cfg.App.Debug = true
	}

	// Tools flags
	cfg.Chat.EnableTools = enableTools
	
	return cfg, nil
}