package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/chat"
)

// Config represents the application configuration
type Config struct {
	// Chat settings
	Chat ChatConfig `json:"chat"`

	// UI settings
	UI UIConfig `json:"ui"`

	// App settings
	App AppConfig `json:"app"`
}

// ChatConfig represents chat-related configuration
type ChatConfig struct {
	// Backend type (aws-bedrock, openai, local, mock)
	BackendType string `json:"backend_type"`

	// Model ID
	ModelID string `json:"model_id"`

	// Default system prompt
	SystemPrompt string `json:"system_prompt"`

	// Maximum number of messages to include in the context
	ContextWindowSize int `json:"context_window_size"`

	// Maximum number of tokens in the response
	MaxTokens int `json:"max_tokens"`

	// Temperature for sampling (0.0-1.0)
	Temperature float64 `json:"temperature"`

	// Top-P sampling parameter
	TopP float64 `json:"top_p"`

	// Enable system tools for the model
	EnableTools bool `json:"enable_tools"`

	// List of enabled tool categories
	EnabledToolCategories []string `json:"enabled_tool_categories"`

	// Context Management options
	ContextManagement ContextManagementConfig `json:"context_management"`

	// AWS options
	AWS AWSConfig `json:"aws"`
}

// ContextManagementConfig contains options for advanced context management
type ContextManagementConfig struct {
	// Enable advanced context management
	Enabled bool `json:"enabled"`

	// Primary model for regular interactions (e.g., Claude 3.7 Sonnet)
	PrimaryModelID string `json:"primary_model_id"`

	// Summarizer model for generating context summaries (e.g., Claude 3.5 Haiku)
	SummarizerModelID string `json:"summarizer_model_id"`

	// Maximum context tokens
	MaxContextTokens int `json:"max_context_tokens"`

	// Enable hierarchical context structure
	EnableHierarchical bool `json:"enable_hierarchical"`

	// Enable context persistence to disk
	EnablePersistence bool `json:"enable_persistence"`

	// Path to store persisted context
	PersistencePath string `json:"persistence_path"`
}

// AWSConfig contains AWS-specific configuration
type AWSConfig struct {
	// AWS region
	Region string `json:"region"`

	// AWS profile
	Profile string `json:"profile"`
}

// UIConfig represents UI-related configuration
type UIConfig struct {
	// Show timestamps in the chat
	ShowTimestamps bool `json:"show_timestamps"`

	// Show typing indicators
	ShowTypingIndicator bool `json:"show_typing_indicator"`

	// Show token usage
	ShowTokenUsage bool `json:"show_token_usage"`

	// Syntax highlighting theme
	SyntaxTheme string `json:"syntax_theme"`
}

// AppConfig represents application-level configuration
type AppConfig struct {
	// Debug mode
	Debug bool `json:"debug"`

	// Log file
	LogFile string `json:"log_file"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Chat: ChatConfig{
			BackendType:           "mock", // Default to mock backend
			ModelID:               "mock",
			SystemPrompt:          "", // Use the default from chat service
			ContextWindowSize:     20,
			MaxTokens:             1000,
			Temperature:           0.7,
			TopP:                  0.9,
			EnableTools:           true,                   // Enable system tools by default
			EnabledToolCategories: []string{"filesystem"}, // Enable only filesystem tools by default
			ContextManagement: ContextManagementConfig{
				Enabled:            false, // Disabled by default
				PrimaryModelID:     "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
				SummarizerModelID:  "anthropic.claude-3-haiku-20240307-v1:0",
				MaxContextTokens:   100000,
				EnableHierarchical: true,
				EnablePersistence:  false,
				PersistencePath:    "",
			},
			AWS: AWSConfig{
				Region:  "", // Use default from AWS config
				Profile: "", // Use default profile
			},
		},
		UI: UIConfig{
			ShowTimestamps:      false,
			ShowTypingIndicator: true,
			ShowTokenUsage:      false,
			SyntaxTheme:         "dracula",
		},
		App: AppConfig{
			Debug:   false,
			LogFile: "",
		},
	}
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		// If no path specified, use default location
		home, err := os.UserHomeDir()
		if err != nil {
			return config, fmt.Errorf("failed to get user home directory: %w", err)
		}

		configPath = filepath.Join(home, ".config", "mcpterm", "config.json")
	}

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, create directory and save default config
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return config, fmt.Errorf("failed to create config directory: %w", err)
		}

		if err := SaveConfig(config, configPath); err != nil {
			return config, fmt.Errorf("failed to save default config: %w", err)
		}

		return config, nil
	}

	// Read and parse the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to the specified file
func SaveConfig(config Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetChatOptions converts the configuration to chat options
func (c *Config) GetChatOptions() interface{} {
	backendType := backend.BackendMock
	switch c.Chat.BackendType {
	case "aws-bedrock", "bedrock":
		backendType = backend.BackendAWSBedrock
	case "openai":
		backendType = backend.BackendOpenAI
	case "local":
		backendType = backend.BackendLocal
	}

	// Extract backend-specific options
	backendOptions := make(map[string]any)

	// Add AWS options
	if backendType == backend.BackendAWSBedrock {
		if c.Chat.AWS.Region != "" {
			backendOptions["region"] = c.Chat.AWS.Region
		}
		if c.Chat.AWS.Profile != "" {
			backendOptions["profile"] = c.Chat.AWS.Profile
		}
	}

	systemPrompt := c.Chat.SystemPrompt
	if systemPrompt == "" {
		// Use default
		opts := chat.DefaultChatOptions()
		systemPrompt = opts.InitialSystemPrompt
	}

	// Base chat options
	baseChatOptions := chat.ChatOptions{
		InitialSystemPrompt:   systemPrompt,
		BackendType:           backendType,
		ModelID:               c.Chat.ModelID,
		ContextWindowSize:     c.Chat.ContextWindowSize,
		MaxTokens:             c.Chat.MaxTokens,
		Temperature:           c.Chat.Temperature,
		BackendOptions:        backendOptions,
		EnableTools:           c.Chat.EnableTools,
		EnabledToolCategories: c.Chat.EnabledToolCategories,
	}

	// If context management is enabled, return ContextChatOptions
	if c.Chat.ContextManagement.Enabled {
		// Get default context options
		contextOpts := chat.DefaultContextChatOptions()

		// Override with config values
		contextOpts.ChatOptions = baseChatOptions
		contextOpts.PrimaryModelID = c.Chat.ContextManagement.PrimaryModelID
		contextOpts.SummarizerModelID = c.Chat.ContextManagement.SummarizerModelID

		// Set context manager config
		contextOpts.ContextManagerConfig.MaxContextTokens = c.Chat.ContextManagement.MaxContextTokens
		contextOpts.ContextManagerConfig.SystemPrompt = systemPrompt

		// Set hierarchical context config
		contextOpts.HierarchicalConfig.LongTermPersistence = c.Chat.ContextManagement.EnablePersistence

		// Set persistence path if persistence is enabled
		if c.Chat.ContextManagement.EnablePersistence && c.Chat.ContextManagement.PersistencePath != "" {
			fmt.Printf("DEBUG CONFIG: Setting persistence path to %s\n", c.Chat.ContextManagement.PersistencePath)
			contextOpts.HierarchicalConfig.LongTermFilePath = c.Chat.ContextManagement.PersistencePath

			// Also set it in the context manager config for good measure
			contextOpts.ContextManagerConfig.PersistencePath = c.Chat.ContextManagement.PersistencePath

			// Set the EnablePersistence flag in the context manager config too
			contextOpts.ContextManagerConfig.EnablePersistence = true
		} else {
			fmt.Printf("DEBUG CONFIG: Not setting persistence path. EnablePersistence=%v, Path='%s'\n",
				c.Chat.ContextManagement.EnablePersistence, c.Chat.ContextManagement.PersistencePath)
		}

		return contextOpts
	}

	// Return standard chat options
	return baseChatOptions
}

// GetStandardChatOptions returns standard chat options regardless of context management settings
func (c *Config) GetStandardChatOptions() chat.ChatOptions {
	chatOptions, ok := c.GetChatOptions().(chat.ChatOptions)
	if !ok {
		// If we got context options, extract the base options
		contextOpts, ok := c.GetChatOptions().(chat.ContextChatOptions)
		if ok {
			return contextOpts.ChatOptions
		}

		// Fall back to default if something went wrong
		return chat.DefaultChatOptions()
	}

	return chatOptions
}
