package context

import (
	"context"
	"time"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// MessageType represents the type of a message in the context
type MessageType string

const (
	// MessageTypeUser represents a message from the user
	MessageTypeUser MessageType = "user"

	// MessageTypeAssistant represents a message from the assistant
	MessageTypeAssistant MessageType = "assistant"

	// MessageTypeSystem represents a system message
	MessageTypeSystem MessageType = "system"

	// MessageTypeSummary represents a summary of previous messages
	MessageTypeSummary MessageType = "summary"
)

// ImportanceLevel represents the importance level of a message
type ImportanceLevel int

const (
	// ImportanceLow represents low importance messages
	ImportanceLow ImportanceLevel = iota + 1
	// ImportanceMedium represents medium importance messages
	ImportanceMedium
	// ImportanceHigh represents high importance messages
	ImportanceHigh
	// ImportanceCritical represents critical messages that must be preserved
	ImportanceCritical
)

// EnhancedMessage represents a message with additional metadata for context management
type EnhancedMessage struct {
	ID           string          // Unique identifier for the message
	Role         string          // Who sent the message (user, assistant, system)
	Content      string          // Message content
	Type         MessageType     // Type of message
	CreatedAt    time.Time       // When the message was created
	TokenCount   int             // Number of tokens in the message
	Importance   ImportanceLevel // Importance level of the message
	Topics       []string        // Topics covered in this message
	Tags         []string        // User or system defined tags
	References   []string        // IDs of related messages
	Embeddings   []float32       // Vector embedding of the message content (for semantic search)
	IsCompressed bool            // Whether this message is a compressed version
}

// Summary represents a compressed summary of multiple messages
type Summary struct {
	ID             string       // Unique identifier for the summary
	Content        string       // Summarized content
	CreatedAt      time.Time    // When the summary was created
	TokenCount     int          // Number of tokens in the summary
	Topics         []string     // Topics covered in this summary
	Tags           []string     // User or system defined tags
	SourceMessages []string     // IDs of messages that were summarized
	TimeSpan       [2]time.Time // Start and end time of the summarized conversation
}

// ContextSelection represents a selection of messages and summaries for a specific request
type ContextSelection struct {
	SystemMessage *EnhancedMessage  // System instructions/prompt
	Messages      []EnhancedMessage // Selected messages
	Summaries     []Summary         // Selected summaries
	TotalTokens   int               // Total tokens used by this selection
	MaxTokens     int               // Maximum allowed tokens for this selection
}

// TokenCounter provides token counting functionality for messages
type TokenCounter interface {
	// CountTokens returns the number of tokens in the given text
	CountTokens(text string) (int, error)

	// CountMessageTokens returns the number of tokens in the given message
	CountMessageTokens(message EnhancedMessage) (int, error)

	// CountSummaryTokens returns the number of tokens in a summary
	CountSummaryTokens(summary Summary) (int, error)

	// EstimateContextTokens estimates the total tokens that would be used when sending
	// these messages to the specified model
	EstimateContextTokens(messages []EnhancedMessage, model string) (int, error)
}

// ContextManagerConfig contains configuration options for the context manager
type ContextManagerConfig struct {
	// MaxHistoryMessages is the maximum number of messages to keep in history
	MaxHistoryMessages int

	// MaxContextTokens is the maximum number of tokens to include in context
	MaxContextTokens int

	// SystemPrompt is the system prompt to use
	SystemPrompt string

	// SummarizerModelID is the ID of the model to use for summarization
	SummarizerModelID string

	// PrimaryModelID is the ID of the primary model
	PrimaryModelID string

	// SummarizationThreshold is the number of messages after which to trigger summarization
	SummarizationThreshold int

	// EnableHierarchicalContext enables hierarchical context with summaries
	EnableHierarchicalContext bool

	// EnablePersistence enables persisting context to disk
	EnablePersistence bool

	// PersistencePath is the path to store persisted context
	PersistencePath string

	// TokenBudgetAllocation defines how to allocate the token budget
	// Keys are categories like "recent", "summaries", "system", values are percentages
	TokenBudgetAllocation map[string]float64
}

// ContextManager provides context management for conversations
type ContextManager interface {
	// AddMessage adds a new message to the context
	AddMessage(message EnhancedMessage) error

	// GetContextForPrompt returns a selection of messages and summaries for a prompt
	GetContextForPrompt(maxTokens int) (ContextSelection, error)

	// GetFullHistory returns the full history of messages
	GetFullHistory() []EnhancedMessage

	// GetSummaries returns all summaries
	GetSummaries() []Summary

	// CreateSummary creates a summary of the specified messages
	CreateSummary(ctx context.Context, messageIDs []string) (*Summary, error)

	// Clear clears the context
	Clear() error

	// SaveToDisk persists the context to disk
	SaveToDisk() error

	// LoadFromDisk loads the context from disk
	LoadFromDisk() error

	// SetImportance sets the importance level of a message
	SetImportance(messageID string, importance ImportanceLevel) error

	// AddTopicToMessage adds a topic to a message
	AddTopicToMessage(messageID string, topic string) error

	// AddTagToMessage adds a tag to a message
	AddTagToMessage(messageID string, tag string) error

	// Search searches the context for messages matching the query
	Search(query string, maxResults int) ([]EnhancedMessage, error)

	// PrepareBackendMessages converts the context selection to backend messages
	PrepareBackendMessages(selection ContextSelection) ([]backend.Message, error)
}
