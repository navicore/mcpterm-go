package chat

import (
	"context"
	"fmt"
	"sync"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// Message represents a chat message
type Message struct {
	Sender  string // Who sent the message ("user", "assistant", "system")
	Content string // Message content
	IsUser  bool   // Whether the message is from the user
}

// ChatOptions contains options for the chat service
type ChatOptions struct {
	InitialSystemPrompt  string
	BackendType          backend.BackendType
	ModelID              string
	ContextWindowSize    int
	MaxTokens            int
	Temperature          float64
	BackendOptions       map[string]any
}

// DefaultChatOptions returns the default chat options
func DefaultChatOptions() ChatOptions {
	return ChatOptions{
		InitialSystemPrompt: defaultSystemPrompt,
		BackendType:        backend.BackendMock,
		ModelID:            "mock",
		ContextWindowSize:  20,
		MaxTokens:          1000,
		Temperature:        0.7,
		BackendOptions:     make(map[string]any),
	}
}

// ChatService provides conversational capabilities
type ChatService struct {
	backend        backend.Backend
	messages       []Message
	options        ChatOptions
	systemPrompt   string
	conversationMu sync.RWMutex
}

// NewChatService creates a new chat service
func NewChatService(opts ChatOptions) (*ChatService, error) {
	// Create the backend
	backendConfig := backend.Config{
		Type:        opts.BackendType,
		ModelID:     opts.ModelID,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		Options:     opts.BackendOptions,
	}
	
	b, err := backend.NewBackend(backendConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat backend: %w", err)
	}
	
	return &ChatService{
		backend:      b,
		messages:     []Message{},
		options:      opts,
		systemPrompt: opts.InitialSystemPrompt,
	}, nil
}

// SendMessage sends a message to the chat service
func (s *ChatService) SendMessage(content string) (Message, error) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	
	// Add user message to history
	userMsg := Message{
		Sender:  "user",
		Content: content,
		IsUser:  true,
	}
	s.messages = append(s.messages, userMsg)
	
	// Prepare messages for the backend
	backendMessages := s.prepareBackendMessages()
	
	// Create chat request
	req := backend.ChatRequest{
		Messages:    backendMessages,
		MaxTokens:   s.options.MaxTokens,
		Temperature: s.options.Temperature,
	}
	
	// Send to backend
	ctx := context.Background()
	resp, err := s.backend.SendMessage(ctx, req)
	if err != nil {
		return Message{}, fmt.Errorf("backend error: %w", err)
	}
	
	// Create response message
	respMsg := Message{
		Sender:  "assistant",
		Content: resp.Content,
		IsUser:  false,
	}
	
	// Add to history
	s.messages = append(s.messages, respMsg)
	
	return respMsg, nil
}

// GetHistory returns the chat history
func (s *ChatService) GetHistory() []Message {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()
	
	// Return a copy of the messages to prevent race conditions
	history := make([]Message, len(s.messages))
	copy(history, s.messages)
	
	return history
}

// Clear clears the chat history
func (s *ChatService) Clear() {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	
	s.messages = []Message{}
}

// updateSystemPrompt updates the system prompt
func (s *ChatService) UpdateSystemPrompt(prompt string) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	
	s.systemPrompt = prompt
}

// prepareBackendMessages prepares the messages for the backend
func (s *ChatService) prepareBackendMessages() []backend.Message {
	// Start with the system prompt
	result := []backend.Message{
		{
			Role:    "system",
			Content: s.systemPrompt,
		},
	}
	
	// Calculate how many messages we can include
	// For now, we'll use a simple approach of taking the last N messages
	messageLimit := s.options.ContextWindowSize
	if messageLimit <= 0 {
		messageLimit = 20 // Default to 20 messages if not specified
	}
	
	// Get the messages to include
	messagesToInclude := s.messages
	if len(messagesToInclude) > messageLimit {
		messagesToInclude = messagesToInclude[len(messagesToInclude)-messageLimit:]
	}
	
	// Add the messages
	for _, msg := range messagesToInclude {
		role := "user"
		if !msg.IsUser {
			role = "assistant"
		}
		
		result = append(result, backend.Message{
			Role:    role,
			Content: msg.Content,
		})
	}
	
	return result
}

// GetBackendInfo returns information about the backend
func (s *ChatService) GetBackendInfo() (string, string) {
	return s.backend.Name(), s.backend.ModelID()
}

// Close closes the chat service and releases resources
func (s *ChatService) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}

// Default system prompt
const defaultSystemPrompt = `You are Claude, a helpful AI assistant. 
Respond concisely and provide accurate, factual information. 
Format your responses using Markdown when appropriate to improve readability.
Use code blocks with proper syntax highlighting when including code.`