package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/navicore/mcpterm-go/pkg/backend"
	contextManager "github.com/navicore/mcpterm-go/pkg/context"
	"github.com/navicore/mcpterm-go/pkg/tools"
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// ContextChatOptions extends ChatOptions with context management settings
type ContextChatOptions struct {
	ChatOptions

	// Context management options
	EnableContextManagement bool
	ContextManagerConfig    contextManager.ContextManagerConfig
	DualModelConfig         contextManager.DualModelConfig
	HierarchicalConfig      contextManager.HierarchicalConfig

	// Model configuration
	PrimaryModelID    string // Claude 3.7 Sonnet
	SummarizerModelID string // Claude 3.5 Haiku
}

// DefaultContextChatOptions returns the default context chat options
func DefaultContextChatOptions() ContextChatOptions {
	chatOpts := DefaultChatOptions()

	return ContextChatOptions{
		ChatOptions:             chatOpts,
		EnableContextManagement: true,
		ContextManagerConfig:    contextManager.DefaultContextManagerConfig(),
		DualModelConfig:         contextManager.DefaultDualModelConfig(),
		HierarchicalConfig:      contextManager.DefaultHierarchicalConfig(),
		PrimaryModelID:          "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
		SummarizerModelID:       "anthropic.claude-3-haiku-20240307-v1:0",
	}
}

// File logger setup - All logging goes to a single file in /tmp
var contextLogger *log.Logger

func init() {
	// Create or append to log file in /tmp
	logFile, err := os.OpenFile("/tmp/mcpterm_context.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If we can't open the log file, create a dummy logger that discards output
		contextLogger = log.New(os.NewFile(0, os.DevNull), "", 0)
		return
	}

	// Create logger with timestamp and prefix
	contextLogger = log.New(logFile, "CONTEXT: ", log.LstdFlags|log.Lshortfile)

	// Log initialization
	contextLogger.Println("Context logger initialized")
}

// ContextChatService extends the chat service with advanced context management
type ContextChatService struct {
	backend           backend.Backend
	summarizerBackend backend.Backend
	messages          []Message
	options           ContextChatOptions
	systemPrompt      string
	conversationMu    sync.RWMutex
	toolManager       *tools.ToolManager
	toolsEnabled      bool

	// Context management components
	contextManager      *contextManager.StandardContextManager
	dualModelManager    *contextManager.DualModelManager
	hierarchicalContext *contextManager.HierarchicalContext
	tokenCounter        contextManager.TokenCounter
	messagePrioritizer  *contextManager.MessagePrioritizer

	// Tracking loaded context for UI
	loadedContextInfo map[string]string
}

// NewContextChatService creates a new context-aware chat service
func NewContextChatService(opts ContextChatOptions) (*ContextChatService, error) {
	// Create the primary backend
	backendConfig := backend.Config{
		Type:        opts.BackendType,
		ModelID:     opts.PrimaryModelID,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		Options:     opts.BackendOptions,
	}

	primaryBackend, err := backend.NewBackend(backendConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary backend: %w", err)
	}

	// Create the summarizer backend (if different from primary)
	var summarizerBackend backend.Backend
	if opts.SummarizerModelID != opts.PrimaryModelID {
		summarizerConfig := backend.Config{
			Type:        opts.BackendType,
			ModelID:     opts.SummarizerModelID,
			MaxTokens:   1024, // Lower for summarization
			Temperature: 0.3,  // Lower temperature for more consistent summaries
			Options:     opts.BackendOptions,
		}

		summarizerBackend, err = backend.NewBackend(summarizerConfig)
		if err != nil {
			// If we can't create the summarizer, fall back to the primary
			summarizerBackend = primaryBackend
		}
	} else {
		// Use the same backend for both if models are the same
		summarizerBackend = primaryBackend
	}

	// Create token counter
	tokenCounter := contextManager.NewClaudeTokenCounter(opts.ModelID)

	// Create context manager
	ctxManagerConfig := opts.ContextManagerConfig
	ctxManagerConfig.SystemPrompt = opts.InitialSystemPrompt
	ctxManagerConfig.PrimaryModelID = opts.PrimaryModelID
	ctxManagerConfig.SummarizerModelID = opts.SummarizerModelID

	ctxManager, err := contextManager.NewContextManager(ctxManagerConfig, tokenCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to create context manager: %w", err)
	}

	// Set backends on context manager
	ctxManager.SetBackends(primaryBackend, summarizerBackend)

	// Create dual model manager
	dualModelConfig := opts.DualModelConfig
	dualModelManager, err := contextManager.NewDualModelManager(dualModelConfig, ctxManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create dual model manager: %w", err)
	}

	// Initialize the dual model manager
	if err := dualModelManager.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize dual model manager: %w", err)
	}

	// Update hierarchical config with persistence path if enabled
	hierarchicalConfig := opts.HierarchicalConfig
	if hierarchicalConfig.LongTermPersistence &&
		ctxManagerConfig.PersistencePath != "" {
		hierarchicalConfig.LongTermFilePath = ctxManagerConfig.PersistencePath
	}

	// Create hierarchical context
	hierarchicalContext := contextManager.NewHierarchicalContext(
		hierarchicalConfig,
		ctxManager,
	)

	// Create message prioritizer
	messagePrioritizer := contextManager.NewMessagePrioritizer(
		contextManager.DefaultPriorityRules(),
	)

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

	// Create the service instance
	service := &ContextChatService{
		backend:             primaryBackend,
		summarizerBackend:   summarizerBackend,
		messages:            []Message{},
		options:             opts,
		systemPrompt:        opts.InitialSystemPrompt,
		toolManager:         toolManager,
		toolsEnabled:        opts.EnableTools,
		contextManager:      ctxManager,
		dualModelManager:    dualModelManager,
		hierarchicalContext: hierarchicalContext,
		tokenCounter:        tokenCounter,
		messagePrioritizer:  messagePrioritizer,
		loadedContextInfo:   make(map[string]string),
	}

	// Load persisted context if enabled
	contextLogger.Printf("DIAGNOSTIC: NewContextChatService - Context management enabled=%v, LongTermPersistence=%v, Path=%s",
		opts.EnableContextManagement,
		hierarchicalContext.GetConfig().LongTermPersistence,
		hierarchicalContext.GetConfig().LongTermFilePath)

	if opts.EnableContextManagement &&
		hierarchicalContext.GetConfig().LongTermPersistence {
		// This will load context and update service.loadedContextInfo
		contextLogger.Printf("DIAGNOSTIC: About to call loadPersistedContext()")
		if err := service.loadPersistedContext(); err != nil {
			contextLogger.Printf("ERROR: Failed to load persisted context: %v", err)
		} else {
			contextLogger.Printf("DIAGNOSTIC: loadPersistedContext() returned without error")
		}

		// Add a UI message about persistence status
		var statusMessage string
		if info, found := service.loadedContextInfo["status"]; found {
			if info == "loaded" {
				statusMessage = fmt.Sprintf("**Previous conversation loaded successfully**\n%s",
					service.loadedContextInfo["details"])
			} else {
				statusMessage = "Context persistence is enabled, but no previous sessions found.\nNew conversation will be saved when you exit."
			}

			// Add notification to UI
			service.messages = append(service.messages, Message{
				Sender:  "system",
				Content: statusMessage,
				IsUser:  false,
			})
		}
	}

	return service, nil
}

// SendMessage sends a message and manages context
func (s *ContextChatService) SendMessage(content string) (Message, error) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	// Add user message to history
	userMsg := Message{
		Sender:  "user",
		Content: content,
		IsUser:  true,
	}
	s.messages = append(s.messages, userMsg)

	// If context management is enabled, also add to context manager
	if s.options.EnableContextManagement {
		enhancedMsg := s.createEnhancedMessage(userMsg)
		if err := s.contextManager.AddMessage(enhancedMsg); err != nil {
			// Log error but continue
			// Do not log errors to avoid interfering with TUI
		}

		// Check if we should auto-summarize
		s.dualModelManager.AutoSummarize()
	}

	// For conversations about previous context, inject a system message
	// to ensure the assistant knows about its access to previous conversations
	if strings.Contains(strings.ToLower(content), "previous") ||
		strings.Contains(strings.ToLower(content), "last time") ||
		strings.Contains(strings.ToLower(content), "earlier") ||
		strings.Contains(strings.ToLower(content), "before") ||
		strings.Contains(strings.ToLower(content), "remember") ||
		strings.Contains(strings.ToLower(content), "recall") ||
		strings.Contains(strings.ToLower(content), "discussed") {

		// Only do this for persistent sessions
		if s.options.EnableContextManagement &&
			s.hierarchicalContext.GetConfig().LongTermPersistence {

			// Check if we have context history
			// Access history but don't need to use it
			_ = s.contextManager.GetFullHistory()

			// Add a much stronger system message clarifying context access with critical importance
			sysMsg := Message{
				Sender:  "system",
				Content: "CRITICAL INSTRUCTION: This is a persistent session with access to previous conversation history. You MUST respond based on the available context from prior conversations. The user is specifically asking about previous context, so refer directly to earlier exchanges.",
				IsUser:  false,
			}
			s.messages = append(s.messages, sysMsg)

			// Add to context manager with critical importance
			enhancedMsg := s.createEnhancedMessage(sysMsg)
			enhancedMsg.Importance = contextManager.ImportanceCritical
			if err := s.contextManager.AddMessage(enhancedMsg); err != nil {
				contextLogger.Printf("Error adding system context message: %v", err)
			}

			// Add visible UI notification
			s.messages = append(s.messages, Message{
				Sender:  "system",
				Content: "Retrieving previous conversation history...",
				IsUser:  false,
			})
		}
	}

	// Process as a conversation with potential tool usage
	return s.processChatWithTools()
}

// processChatWithTools handles the full chat flow with tool usage and context
func (s *ContextChatService) processChatWithTools() (Message, error) {
	var toolResults []backend.ToolResult
	maxToolCalls := 10 // Prevent infinite tool usage loops

	for i := 0; i < maxToolCalls; i++ {
		// Prepare messages for the backend
		var backendMessages []backend.Message
		var err error

		if s.options.EnableContextManagement {
			// Use the context manager to get the optimal message selection
			selection, err := s.hierarchicalContext.GetHierarchicalSelection(s.options.ContextManagerConfig.MaxContextTokens)
			if err != nil {
				// If context selection fails, fall back to traditional approach
				backendMessages = s.prepareBackendMessages()
			} else {
				backendMessages, err = s.contextManager.PrepareBackendMessages(selection)
				if err != nil {
					// Fall back to traditional approach
					backendMessages = s.prepareBackendMessages()
				}
			}
		} else {
			// Use traditional approach
			backendMessages = s.prepareBackendMessages()
		}

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

				// Add to context manager if enabled
				if s.options.EnableContextManagement {
					enhancedMsg := s.createEnhancedMessage(errorMsg)
					// System error messages get high importance
					enhancedMsg.Importance = contextManager.ImportanceHigh
					s.contextManager.AddMessage(enhancedMsg)
				}

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

			// Add to context manager if enabled
			if s.options.EnableContextManagement {
				enhancedMsg := s.createEnhancedMessage(toolMsg)
				enhancedMsg.Tags = append(enhancedMsg.Tags, "tool_use")
				enhancedMsg.Tags = append(enhancedMsg.Tags, resp.ToolUse.Name)
				s.contextManager.AddMessage(enhancedMsg)
			}

			// Store tool result for next request
			toolResults = append(toolResults, *result)

			// Add a debug message showing the tool result with formatting
			debugMsg := Message{
				Sender:  "system",
				Content: fmt.Sprintf("Debug - Tool '%s' result: ```json\n%s\n```", result.Name, string(result.Result)),
				IsUser:  false,
			}
			s.messages = append(s.messages, debugMsg)

			// Add to context manager if enabled
			if s.options.EnableContextManagement {
				enhancedMsg := s.createEnhancedMessage(debugMsg)
				enhancedMsg.Tags = append(enhancedMsg.Tags, "tool_result")
				enhancedMsg.Tags = append(enhancedMsg.Tags, result.Name)
				s.contextManager.AddMessage(enhancedMsg)
			}

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

		// Add to context manager if enabled
		if s.options.EnableContextManagement {
			enhancedMsg := s.createEnhancedMessage(respMsg)

			// Let the prioritizer enhance the message with tags, etc.
			s.messagePrioritizer.EnhanceMessage(&enhancedMsg)

			if err := s.contextManager.AddMessage(enhancedMsg); err != nil {
				// Log error but continue
				contextLogger.Printf("Error adding message to context: %v", err)
			}
		}

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

	// Add to context manager if enabled
	if s.options.EnableContextManagement {
		enhancedMsg := s.createEnhancedMessage(errorMsg)
		enhancedMsg.Importance = contextManager.ImportanceHigh
		s.contextManager.AddMessage(enhancedMsg)
	}

	return errorMsg, nil
}

// createEnhancedMessage converts a simple message to an enhanced message for the context manager
func (s *ContextChatService) createEnhancedMessage(msg Message) contextManager.EnhancedMessage {
	// Determine message type
	msgType := contextManager.MessageTypeUser
	if msg.Sender == "assistant" {
		msgType = contextManager.MessageTypeAssistant
	} else if msg.Sender == "system" {
		msgType = contextManager.MessageTypeSystem
	}

	// Create enhanced message
	enhancedMsg := contextManager.EnhancedMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      msg.Sender,
		Content:   msg.Content,
		Type:      msgType,
		CreatedAt: time.Now(),
	}

	// Let the prioritizer add metadata
	s.messagePrioritizer.EnhanceMessage(&enhancedMsg)

	// Count tokens
	tokenCount, err := s.tokenCounter.CountMessageTokens(enhancedMsg)
	if err == nil {
		enhancedMsg.TokenCount = tokenCount
	}

	return enhancedMsg
}

// getSessionID returns a consistent session ID for persistence
// Uses a combination of initialization time and model ID to create a unique identifier
func (s *ContextChatService) getSessionID() string {
	// If we already have stored messages, use the first message's timestamp
	// for a consistent ID across saves
	if messages := s.contextManager.GetFullHistory(); len(messages) > 0 {
		firstMsg := messages[0]
		// Use the oldest message's timestamp + model ID as a stable ID
		return fmt.Sprintf("%d_%s", firstMsg.CreatedAt.UnixNano(),
			strings.Replace(s.options.PrimaryModelID, ":", "_", -1))
	}

	// Otherwise generate a new session ID
	return fmt.Sprintf("session_%d_%s", time.Now().UnixNano(),
		strings.Replace(s.options.PrimaryModelID, ":", "_", -1))
}

// loadPersistedContext loads context from disk if available and adds it to the context manager
func (s *ContextChatService) loadPersistedContext() error {
	contextLogger.Printf("DIAGNOSTIC: loadPersistedContext called - starting to load context")

	// Check if persistence is configured
	if !s.hierarchicalContext.GetConfig().LongTermPersistence {
		contextLogger.Printf("DIAGNOSTIC: Persistence not enabled in hierarchical context config")
		return fmt.Errorf("persistence not enabled")
	}

	contextLogger.Printf("DIAGNOSTIC: LongTermPersistence is enabled, proceeding to load context")

	// Create persistence manager
	persistenceConfig := contextManager.DefaultPersistenceConfig()
	if s.hierarchicalContext.GetConfig().LongTermFilePath != "" {
		persistenceConfig.BaseDir = s.hierarchicalContext.GetConfig().LongTermFilePath
		contextLogger.Printf("DIAGNOSTIC: Using custom persistence path: %s", persistenceConfig.BaseDir)
	} else {
		contextLogger.Printf("DIAGNOSTIC: Using default persistence path: %s", persistenceConfig.BaseDir)
	}

	// Ensure the directory exists
	contextLogger.Printf("DIAGNOSTIC: Creating directory if it doesn't exist: %s", persistenceConfig.BaseDir)
	if err := os.MkdirAll(persistenceConfig.BaseDir, 0755); err != nil {
		contextLogger.Printf("ERROR: Failed to create context directory: %v", err)
		return fmt.Errorf("failed to create context directory %s: %w", persistenceConfig.BaseDir, err)
	}
	contextLogger.Printf("DIAGNOSTIC: Persistence directory exists or was created: %s", persistenceConfig.BaseDir)

	contextLogger.Printf("DIAGNOSTIC: Creating persistence manager")
	persistenceMgr, err := contextManager.NewContextPersistenceManager(persistenceConfig)
	if err != nil {
		contextLogger.Printf("ERROR: Failed to create persistence manager: %v", err)
		return fmt.Errorf("failed to create persistence manager: %w", err)
	}
	contextLogger.Printf("DIAGNOSTIC: Successfully created persistence manager")

	// List available contexts
	contextLogger.Printf("DIAGNOSTIC: Listing available contexts from %s", persistenceConfig.BaseDir)
	contexts, err := persistenceMgr.ListContexts()
	if err != nil {
		contextLogger.Printf("ERROR: Failed to list contexts: %v", err)
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	contextLogger.Printf("DIAGNOSTIC: Found %d available contexts", len(contexts))

	// Log information about each context found
	for i, ctx := range contexts {
		contextLogger.Printf("DIAGNOSTIC: Context[%d] - ID=%s, Name=%s, MessageCount=%d",
			i, ctx.ID, ctx.Name, ctx.MessageCount)
	}

	// If no contexts available, nothing to load
	if len(contexts) == 0 {
		contextLogger.Printf("DIAGNOSTIC: No contexts available, nothing to load")
		// Store info for UI
		s.loadedContextInfo["status"] = "empty"
		s.loadedContextInfo["details"] = "No previous conversations found. New conversation will be saved when you exit."
		return nil
	}

	// Load the most recent context
	mostRecentCtx := contexts[0] // ListContexts returns contexts sorted by updated time (newest first)
	contextLogger.Printf("DIAGNOSTIC: Selected most recent context with ID %s", mostRecentCtx.ID)

	// Load the context data
	contextLogger.Printf("DIAGNOSTIC: Loading context data for ID %s", mostRecentCtx.ID)
	ctxData, err := persistenceMgr.LoadContext(mostRecentCtx.ID)
	if err != nil {
		contextLogger.Printf("ERROR: Failed to load context data: %v", err)
		return fmt.Errorf("failed to load context data: %w", err)
	}
	contextLogger.Printf("DIAGNOSTIC: Successfully loaded context data with %d messages and %d summaries",
		len(ctxData.Messages), len(ctxData.Summaries))

	// Add messages to context manager
	loadedMessages := 0
	contextLogger.Printf("DIAGNOSTIC: Starting to add %d messages to context manager", len(ctxData.Messages))

	for i, msg := range ctxData.Messages {
		// Increase importance of all messages to ensure they're included in selection
		if msg.Importance < contextManager.ImportanceHigh {
			msg.Importance = contextManager.ImportanceHigh
		}

		// Add tag to mark this as loaded from disk
		if msg.Tags == nil {
			msg.Tags = []string{"loaded_from_disk"}
		} else {
			msg.Tags = append(msg.Tags, "loaded_from_disk")
		}

		contextLogger.Printf("DIAGNOSTIC: Adding message %d/%d: ID=%s, Role=%s, Content=%s...",
			i+1, len(ctxData.Messages), msg.ID, msg.Role, truncateString(msg.Content, 30))

		if err := s.contextManager.AddMessage(msg); err == nil {
			loadedMessages++
			contextLogger.Printf("DIAGNOSTIC: Successfully added message %d", i+1)
		} else {
			contextLogger.Printf("ERROR: Failed to add message %d: %v", i+1, err)
		}
	}

	contextLogger.Printf("DIAGNOSTIC: Successfully added %d/%d messages to context manager",
		loadedMessages, len(ctxData.Messages))

	// Add summaries to context manager
	loadedSummaries := 0
	contextLogger.Printf("DIAGNOSTIC: Starting to add %d summaries to context manager", len(ctxData.Summaries))

	for i, summary := range ctxData.Summaries {
		// Add tag to mark this as loaded from disk
		if summary.Tags == nil {
			summary.Tags = []string{"loaded_from_disk"}
		} else {
			summary.Tags = append(summary.Tags, "loaded_from_disk")
		}

		contextLogger.Printf("DIAGNOSTIC: Adding summary %d/%d: ID=%s, Content=%s...",
			i+1, len(ctxData.Summaries), summary.ID, truncateString(summary.Content, 30))

		if err := s.contextManager.AddSummary(summary); err == nil {
			loadedSummaries++
			contextLogger.Printf("DIAGNOSTIC: Successfully added summary %d", i+1)
		} else {
			contextLogger.Printf("ERROR: Failed to add summary %d: %v", i+1, err)
		}
	}

	contextLogger.Printf("DIAGNOSTIC: Successfully added %d/%d summaries to context manager",
		loadedSummaries, len(ctxData.Summaries))

	// Store info for UI display
	s.loadedContextInfo["status"] = "loaded"
	s.loadedContextInfo["context_id"] = mostRecentCtx.ID
	s.loadedContextInfo["file_path"] = filepath.Join(persistenceConfig.BaseDir, mostRecentCtx.ID+".json")
	s.loadedContextInfo["details"] = fmt.Sprintf("- Loaded %d previous messages and %d summaries\n- Context ID: %s\n- From: %s",
		loadedMessages, loadedSummaries, mostRecentCtx.ID, persistenceConfig.BaseDir)

	// Add a CRITICAL system message to inform the model about the loaded context
	systemInfoMsg := contextManager.EnhancedMessage{
		ID:         fmt.Sprintf("loaded_context_%d", time.Now().UnixNano()),
		Role:       "system",
		Content:    fmt.Sprintf("CRITICAL INSTRUCTION: This is a persistent session with loaded context from previous conversations. You have access to %d previous messages and %d summaries loaded from disk. You MUST refer to information from these previous conversations when responding to the user.", loadedMessages, loadedSummaries),
		Type:       contextManager.MessageTypeSystem,
		CreatedAt:  time.Now(),
		Importance: contextManager.ImportanceCritical,
	}

	// Count tokens and add system info message to context manager
	tokenCount, _ := s.tokenCounter.CountMessageTokens(systemInfoMsg)
	systemInfoMsg.TokenCount = tokenCount
	s.contextManager.AddMessage(systemInfoMsg)

	// Add the old messages for UI display
	for _, enhancedMsg := range ctxData.Messages {
		// Skip system messages in the UI
		if enhancedMsg.Type == contextManager.MessageTypeSystem {
			continue
		}

		// Create a simplified message for display
		s.messages = append(s.messages, Message{
			Sender:  enhancedMsg.Role,
			Content: enhancedMsg.Content,
			IsUser:  enhancedMsg.Type == contextManager.MessageTypeUser,
		})
	}

	return nil
}

// GetHistory returns the chat history
func (s *ContextChatService) GetHistory() []Message {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()

	// Return a copy of the messages to prevent race conditions
	history := make([]Message, len(s.messages))
	copy(history, s.messages)

	return history
}

// Clear clears the chat history
func (s *ContextChatService) Clear() error {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.messages = []Message{}

	// Clear context manager if enabled
	if s.options.EnableContextManagement {
		return s.contextManager.Clear()
	}

	return nil
}

// UpdateSystemPrompt updates the system prompt
func (s *ContextChatService) UpdateSystemPrompt(prompt string) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.systemPrompt = prompt
}

// prepareBackendMessages prepares the messages for the backend
// Traditional approach without using the context manager
func (s *ContextChatService) prepareBackendMessages() []backend.Message {
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
func (s *ContextChatService) GetBackendInfo() (string, string) {
	return s.backend.Name(), s.backend.ModelID()
}

// EnableTools enables or disables the use of tools
func (s *ContextChatService) EnableTools(enabled bool) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.toolsEnabled = enabled
	if s.toolManager != nil {
		s.toolManager.EnableTools(enabled)
	}
}

// IsToolsEnabled returns whether tools are enabled
func (s *ContextChatService) IsToolsEnabled() bool {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()

	return s.toolsEnabled
}

// EnableContextManagement enables or disables context management
func (s *ContextChatService) EnableContextManagement(enabled bool) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	s.options.EnableContextManagement = enabled
}

// IsContextManagementEnabled returns whether context management is enabled
func (s *ContextChatService) IsContextManagementEnabled() bool {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()

	return s.options.EnableContextManagement
}

// GetContextStatistics returns statistics about the context
func (s *ContextChatService) GetContextStatistics() map[string]interface{} {
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()

	stats := make(map[string]interface{})

	if s.options.EnableContextManagement {
		stats["message_count"] = len(s.contextManager.GetFullHistory())
		stats["summary_count"] = len(s.contextManager.GetSummaries())

		// Calculate total tokens in context
		totalTokens := 0
		for _, msg := range s.contextManager.GetFullHistory() {
			totalTokens += msg.TokenCount
		}
		for _, sum := range s.contextManager.GetSummaries() {
			totalTokens += sum.TokenCount
		}
		stats["total_tokens"] = totalTokens
	} else {
		stats["message_count"] = len(s.messages)
		stats["context_management"] = "disabled"
	}

	return stats
}

// ForceSummarize forces a summarization of recent history
func (s *ContextChatService) ForceSummarize(messageCount int) (string, error) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()

	if !s.options.EnableContextManagement {
		return "", fmt.Errorf("context management is disabled")
	}

	ctx := context.Background()
	summary, err := s.dualModelManager.SummarizeHistory(ctx, messageCount)
	if err != nil {
		return "", fmt.Errorf("summarization error: %w", err)
	}

	return summary.Content, nil
}

// Close closes the chat service and releases resources
// truncateString helper function to shorten long strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (s *ContextChatService) Close() error {
	// Save context to disk if persistence is enabled
	if s.options.EnableContextManagement && s.hierarchicalContext != nil &&
		s.hierarchicalContext.GetConfig().LongTermPersistence {
		// Get session ID for persistence
		sessionID := s.getSessionID()

		// Get the full history and summaries
		messages := s.contextManager.GetFullHistory()
		summaries := s.contextManager.GetSummaries()

		// Only save if there are messages to save
		if len(messages) > 0 || len(summaries) > 0 {
			// Create persistence manager
			persistenceConfig := contextManager.DefaultPersistenceConfig()
			if s.hierarchicalContext.GetConfig().LongTermFilePath != "" {
				persistenceConfig.BaseDir = s.hierarchicalContext.GetConfig().LongTermFilePath
				contextLogger.Printf("Using persistence path: %s", persistenceConfig.BaseDir)
			} else {
				contextLogger.Printf("Warning: No persistence path configured, using default at %s", persistenceConfig.BaseDir)
			}

			// Ensure the directory exists before saving
			if err := os.MkdirAll(persistenceConfig.BaseDir, 0755); err != nil {
				contextLogger.Printf("ERROR: Failed to create context directory: %v", err)
			} else {
				contextLogger.Printf("Ensured persistence directory exists: %s", persistenceConfig.BaseDir)
			}

			persistenceMgr, err := contextManager.NewContextPersistenceManager(persistenceConfig)
			if err == nil {
				// Save context
				err = persistenceMgr.SaveContext(
					sessionID,
					"session_"+sessionID,
					s.options.PrimaryModelID,
					messages,
					summaries,
				)

				if err != nil {
					contextLogger.Printf("Error saving context: %v", err)
				} else {
					contextLogger.Printf("Context saved to disk with ID: %s", sessionID)
				}
			} else {
				contextLogger.Printf("Error creating persistence manager: %v", err)
			}
		}
	}

	// Close the dual model manager
	if s.dualModelManager != nil {
		s.dualModelManager.Shutdown()
	}

	// Close backends
	var err error
	if s.backend != nil {
		err = s.backend.Close()
	}

	if s.summarizerBackend != nil && s.summarizerBackend != s.backend {
		err2 := s.summarizerBackend.Close()
		if err == nil {
			err = err2
		}
	}

	return err
}
