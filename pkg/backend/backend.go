package backend

import (
	"context"
	"encoding/json"
)

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`     // "user", "assistant", "system", etc.
	Content string `json:"content"`  // Message content
}

// ChatRequest contains the parameters for a chat completion request
type ChatRequest struct {
	Messages    []Message          // The conversation history
	MaxTokens   int                // Maximum tokens to generate
	Temperature float64            // Temperature for sampling (0.0-1.0)
	TopP        float64            // Top-p sampling parameter
	Options     map[string]any     // Backend-specific options
}

// ToolUse represents a tool call from the model
type ToolUse struct {
	Name  string          // Name of the tool to use
	Input json.RawMessage // Raw JSON input to the tool
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Name   string          // Name of the tool that was used
	Result json.RawMessage // Raw JSON result from the tool
}

// ChatResponse contains the response from a chat completion
type ChatResponse struct {
	Content      string             // The generated text
	FinishReason string             // Reason why generation stopped ("stop", "length", "tool_use", etc.)
	Usage        map[string]int     // Token usage statistics
	Error        error              // Any error that occurred
	ToolUse      *ToolUse           // Tool use request from the model, if any
	ToolResults  []ToolResult       // Results from previous tool usage
}

// BackendType represents the type of chat backend
type BackendType string

const (
	BackendAWSBedrock BackendType = "aws-bedrock"
	BackendOpenAI     BackendType = "openai"
	BackendLocal      BackendType = "local"
	BackendMock       BackendType = "mock"
)

// Backend is the interface that all chat backends must implement
type Backend interface {
	// Name returns the name of the backend
	Name() string
	
	// Type returns the type of the backend
	Type() BackendType
	
	// ModelID returns the model identifier
	ModelID() string
	
	// SendMessage sends a message to the backend and returns the response
	SendMessage(ctx context.Context, req ChatRequest) (ChatResponse, error)
	
	// Close closes any resources held by the backend
	Close() error
}

// Config represents the configuration for a chat backend
type Config struct {
	Type         BackendType       // The backend type
	ModelID      string            // The model ID/Name
	MaxTokens    int               // Default max tokens
	Temperature  float64           // Default temperature
	Options      map[string]any    // Backend-specific options
}

// Factory creates a new backend based on the provided configuration
type Factory func(config Config) (Backend, error)

// registry of backend factories
var backendFactories = make(map[BackendType]Factory)

// RegisterBackend registers a backend factory for a specific backend type
func RegisterBackend(backendType BackendType, factory Factory) {
	backendFactories[backendType] = factory
}

// NewBackend creates a new backend based on the provided configuration
func NewBackend(config Config) (Backend, error) {
	factory, ok := backendFactories[config.Type]
	if !ok {
		return nil, &BackendError{
			Code:    ErrCodeUnsupportedBackend,
			Message: "unsupported backend type: " + string(config.Type),
		}
	}
	
	return factory(config)
}