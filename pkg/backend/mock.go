package backend

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func init() {
	RegisterBackend(BackendMock, NewMockBackend)
}

// MockBackend is a simple mock implementation of the Backend interface for testing
type MockBackend struct {
	config  Config
	modelID string
}

// NewMockBackend creates a new mock backend
func NewMockBackend(config Config) (Backend, error) {
	return &MockBackend{
		config:  config,
		modelID: config.ModelID,
	}, nil
}

// Name returns the name of the backend
func (b *MockBackend) Name() string {
	return "Mock Backend"
}

// Type returns the type of the backend
func (b *MockBackend) Type() BackendType {
	return BackendMock
}

// ModelID returns the model identifier
func (b *MockBackend) ModelID() string {
	return b.modelID
}

// SendMessage simulates sending a message in the mock backend
func (b *MockBackend) SendMessage(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Simulate a brief delay
	select {
	case <-ctx.Done():
		return ChatResponse{}, ctx.Err()
	case <-time.After(500 * time.Millisecond):
		// Continue processing
	}
	
	// Get the last user message
	var lastUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}
	
	// Generate a mock response based on the user's message
	response := b.generateMockResponse(lastUserMessage)
	
	// Prepare usage statistics
	usage := make(map[string]int)
	usage["prompt_tokens"] = len(strings.Split(lastUserMessage, " "))
	usage["completion_tokens"] = len(strings.Split(response, " "))
	usage["total_tokens"] = usage["prompt_tokens"] + usage["completion_tokens"]
	
	return ChatResponse{
		Content:      response,
		FinishReason: "stop",
		Usage:        usage,
	}, nil
}

// Close closes any resources held by the backend
func (b *MockBackend) Close() error {
	return nil
}

// generateMockResponse generates a mock response based on the user's message
func (b *MockBackend) generateMockResponse(userMessage string) string {
	lowerMessage := strings.ToLower(userMessage)
	
	switch {
	case strings.Contains(lowerMessage, "hello") || strings.Contains(lowerMessage, "hi"):
		return "# Hello there!\n\nI'm a mock AI assistant. While I can't actually process your queries, I can demonstrate the interface. How can I pretend to help you today?"
		
	case strings.Contains(lowerMessage, "help"):
		return "## Help Menu\n\nIn this mock mode, I can showcase the following features:\n\n- **Markdown Formatting**: Headers, lists, code blocks, etc.\n- **Chat Interface**: Simulated conversation flow\n- **Mock Responses**: Pre-programmed responses based on keywords\n\nTo see these in action, try commands like `hello`, `features`, `test markdown`, etc."
		
	case strings.Contains(lowerMessage, "feature"):
		return "## Features of This Chat Interface\n\n- **Vi-like Navigation**: Move through conversations with vim keybindings\n- **Multiple Backends**: Support for different AI providers\n- **Markdown Support**: Rich text formatting\n- **Terminal UI**: Clean, efficient interface\n- **History Navigation**: Browse previous messages\n\nThese are simulated features in the mock backend."
		
	case strings.Contains(lowerMessage, "markdown") || strings.Contains(lowerMessage, "format"):
		return "# Markdown Examples\n\n## Text Formatting\n\n*Italic text* and **bold text** are supported.\n\n## Lists\n\n1. Numbered lists\n2. Are easy to create\n\n- Bullet points\n- Work as well\n\n## Code Blocks\n\n```go\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n> Blockquotes are also supported."
		
	default:
		return fmt.Sprintf("I'm a mock assistant, so I don't fully understand your message: \"%s\". This is just a simulated response. Try asking about 'help', 'features', or 'markdown' to see some example responses.", userMessage)
	}
}