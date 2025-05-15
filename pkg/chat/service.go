package chat

import (
	"context"
	"fmt"
	"sync"

	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/tools"
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// ServiceMessage is used internally by ChatService - use Message from chat.go for the interface
type ServiceMessage Message

// ChatOptions contains options for the chat service
type ChatOptions struct {
	InitialSystemPrompt   string
	BackendType           backend.BackendType
	ModelID               string
	ContextWindowSize     int
	MaxTokens             int
	Temperature           float64
	BackendOptions        map[string]any
	EnableTools           bool     // Whether to enable tool support
	EnabledToolCategories []string // List of enabled tool categories
}

// DefaultChatOptions returns the default chat options
func DefaultChatOptions() ChatOptions {
	return ChatOptions{
		InitialSystemPrompt:   GetDefaultSystemPrompt(),
		BackendType:           backend.BackendMock,
		ModelID:               "mock",
		ContextWindowSize:     20,
		MaxTokens:             1000,
		Temperature:           0.7,
		BackendOptions:        make(map[string]any),
		EnableTools:           true,                   // Tools enabled by default
		EnabledToolCategories: []string{"filesystem"}, // Only filesystem tools by default
	}
}

// ChatService provides conversational capabilities
type ChatService struct {
	backend        backend.Backend
	messages       []Message
	options        ChatOptions
	systemPrompt   string
	conversationMu sync.RWMutex
	toolManager    *tools.ToolManager
	toolsEnabled   bool
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

	// Create tool manager
	toolManager, err := tools.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tool manager: %w", err)
	}

	// Set tool availability based on options
	toolManager.EnableTools(opts.EnableTools)

	// Enable specific categories if provided
	if len(opts.EnabledToolCategories) > 0 {
		if err := toolManager.EnableCategoriesByIDs(opts.EnabledToolCategories); err != nil {
			return nil, fmt.Errorf("failed to enable tool categories: %w", err)
		}
	}

	return &ChatService{
		backend:      b,
		messages:     []Message{},
		options:      opts,
		systemPrompt: opts.InitialSystemPrompt,
		toolManager:  toolManager,
		toolsEnabled: opts.EnableTools,
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

	// Process as a conversation with potential tool use
	return s.processChatWithTools()
}

// processChatWithTools handles the full chat flow with potential tool usage
func (s *ChatService) processChatWithTools() (Message, error) {
	var toolResults []backend.ToolResult
	maxToolCalls := 10 // Prevent infinite tool usage loops

	for i := 0; i < maxToolCalls; i++ {
		// Prepare messages for the backend
		backendMessages := s.prepareBackendMessages()

		// Create chat request with tools if enabled
		req := backend.ChatRequest{
			Messages:    backendMessages,
			MaxTokens:   s.options.MaxTokens,
			Temperature: s.options.Temperature,
			Options:     make(map[string]any),
		}

		// Add tools if enabled
		if s.toolsEnabled && s.toolManager != nil && s.toolManager.IsToolsEnabled() {
			req.Options["tools"] = s.toolManager.GetTools()

			// Add tool results if we have any
			if len(toolResults) > 0 {
				req.Options["tool_results"] = toolResults
			}
		}

		// Send to backend
		ctx := context.Background()
		resp, err := s.backend.SendMessage(ctx, req)
		if err != nil {
			return Message{}, fmt.Errorf("backend error: %w", err)
		}

		// If the model requested a tool
		if resp.ToolUse != nil && resp.FinishReason == "tool_use" {
			// Execute the tool first before adding any messages
			result, err := s.toolManager.HandleToolUse((*core.ToolUse)(resp.ToolUse))
			if err != nil {
				// Add error message to history
				errorMsg := Message{
					Sender:  "system",
					Content: fmt.Sprintf("Error executing tool: %v", err),
					IsUser:  false,
				}
				s.messages = append(s.messages, errorMsg)

				// Return error to user
				return errorMsg, nil
			}

			// Add a single combined message about tool usage with more details
			toolMsg := Message{
				Sender: "assistant",
				Content: fmt.Sprintf("Using the '%s' tool to help answer your question. Tool request details: %s",
					resp.ToolUse.Name,
					string(resp.ToolUse.Input)),
				IsUser: false,
			}
			s.messages = append(s.messages, toolMsg)

			// Store tool result for next request
			toolResults = append(toolResults, *result)

			// Add a debug message showing the tool result with formatting
			debugMsg := Message{
				Sender:  "system",
				Content: fmt.Sprintf("Debug - Tool '%s' result: ```json\n%s\n```", result.Name, string(result.Result)),
				IsUser:  false,
			}
			s.messages = append(s.messages, debugMsg)

			// Continue to next iteration to send the tool result
			continue
		}

		// No tool use, we have a final response
		respMsg := Message{
			Sender:  "assistant",
			Content: resp.Content,
			IsUser:  false,
		}

		// Add to history
		s.messages = append(s.messages, respMsg)

		// Return the final response
		return respMsg, nil
	}

	// If we reached max tool calls, inform the user
	errorMsg := Message{
		Sender:  "system",
		Content: "Exceeded maximum number of tool calls. The operation was halted.",
		IsUser:  false,
	}
	s.messages = append(s.messages, errorMsg)

	return errorMsg, nil
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
func (s *ChatService) Clear() error {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.messages = []Message{}
	return nil
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

// EnableTools enables or disables the use of tools
func (s *ChatService) EnableTools(enabled bool) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.toolsEnabled = enabled
	if s.toolManager != nil {
		s.toolManager.EnableTools(enabled)
	}
}

// IsToolsEnabled returns whether tools are enabled
func (s *ChatService) IsToolsEnabled() bool {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()

	return s.toolsEnabled
}

// Close closes the chat service and releases resources
func (s *ChatService) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}

// Default system prompt for backward compatibility
// This will be kept in sync with DefaultSystemPrompt in prompt.go
const defaultSystemPrompt = `You are Claude, a helpful AI assistant in a terminal environment.

Respond concisely and provide accurate, factual information.
Format your responses using Markdown when appropriate to improve readability.
Use code blocks with proper syntax highlighting when including code.

You have access to the following capabilities:
- Reading files from the filesystem
- Creating and modifying files and directories
- Running shell commands to help the user
- Finding files and searching within them

Always ensure you provide the most helpful and accurate responses possible.
When asked to run commands or modify files, explain what you're doing clearly.

Remember that you are running in a terminal interface with vi-like navigation,
so format your responses appropriately for this context.`
