package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	
	"mcpterm-go/pkg/backend"
	"mcpterm-go/pkg/chat"
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
	
	// AWS options
	AWS AWSConfig `json:"aws"`
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
			BackendType:       "mock",  // Default to mock backend
			ModelID:           "mock",
			SystemPrompt:      "",      // Use the default from chat service
			ContextWindowSize: 20,
			MaxTokens:         1000,
			Temperature:       0.7,
			TopP:              0.9,
			AWS: AWSConfig{
				Region:  "",  // Use default from AWS config
				Profile: "",  // Use default profile
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
func (c *Config) GetChatOptions() chat.ChatOptions {
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
	
	return chat.ChatOptions{
		InitialSystemPrompt: systemPrompt,
		BackendType:         backendType,
		ModelID:             c.Chat.ModelID,
		ContextWindowSize:   c.Chat.ContextWindowSize,
		MaxTokens:           c.Chat.MaxTokens,
		Temperature:         c.Chat.Temperature,
		BackendOptions:      backendOptions,
	}
}