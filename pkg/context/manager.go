package context

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// DefaultContextManagerConfig returns the default context manager configuration
func DefaultContextManagerConfig() ContextManagerConfig {
	return ContextManagerConfig{
		MaxHistoryMessages:        1000,
		MaxContextTokens:          100000,
		SummarizationThreshold:    20,
		EnableHierarchicalContext: true,
		EnablePersistence:         false,
		PersistencePath:           "",
		TokenBudgetAllocation: map[string]float64{
			"system":  0.10, // 10% for system prompt
			"recent":  0.60, // 60% for recent messages
			"summary": 0.25, // 25% for summaries
			"reserve": 0.05, // 5% reserve
		},
	}
}

// StandardContextManager is the standard implementation of the ContextManager interface
type StandardContextManager struct {
	config            ContextManagerConfig
	messages          []EnhancedMessage
	summaries         []Summary
	tokenCounter      TokenCounter
	primaryBackend    backend.Backend
	summarizerBackend backend.Backend
	systemPrompt      string
	mu                sync.RWMutex
}

// NewContextManager creates a new context manager with the given configuration
func NewContextManager(config ContextManagerConfig, tokenCounter TokenCounter) (*StandardContextManager, error) {
	if tokenCounter == nil {
		return nil, errors.New("token counter is required")
	}

	return &StandardContextManager{
		config:       config,
		messages:     make([]EnhancedMessage, 0, 100),
		summaries:    make([]Summary, 0, 10),
		tokenCounter: tokenCounter,
		systemPrompt: config.SystemPrompt,
		mu:           sync.RWMutex{},
	}, nil
}

// SetBackends sets the primary and summarizer backends
func (m *StandardContextManager) SetBackends(primary, summarizer backend.Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.primaryBackend = primary
	m.summarizerBackend = summarizer
}

// AddMessage adds a new message to the context
func (m *StandardContextManager) AddMessage(message EnhancedMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate an ID if not provided
	if message.ID == "" {
		message.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}

	// Set creation time if not set
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	// Count tokens if not already counted
	if message.TokenCount == 0 {
		count, err := m.tokenCounter.CountMessageTokens(message)
		if err != nil {
			return fmt.Errorf("failed to count tokens: %w", err)
		}
		message.TokenCount = count
	}

	// Add message to history
	m.messages = append(m.messages, message)

	// Check if we need to summarize older messages
	if m.config.EnableHierarchicalContext &&
		len(m.messages) >= m.config.SummarizationThreshold {
		// This would trigger async summarization in a goroutine
		// But we'll leave this for later implementation
	}

	// If we've exceeded the max history size, trim the oldest messages
	// after ensuring they've been summarized
	if len(m.messages) > m.config.MaxHistoryMessages {
		// For now, just trim - summarization will be implemented later
		m.messages = m.messages[len(m.messages)-m.config.MaxHistoryMessages:]
	}

	return nil
}

// AddSummary adds a summary to the context
func (m *StandardContextManager) AddSummary(summary Summary) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate an ID if not provided
	if summary.ID == "" {
		summary.ID = fmt.Sprintf("summary_%d", time.Now().UnixNano())
	}

	// Set creation time if not set
	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = time.Now()
	}

	// Count tokens if not already counted
	if summary.TokenCount == 0 {
		count, err := m.tokenCounter.CountSummaryTokens(summary)
		if err != nil {
			return fmt.Errorf("failed to count summary tokens: %w", err)
		}
		summary.TokenCount = count
	}

	// Add to summaries
	m.summaries = append(m.summaries, summary)

	return nil
}

// GetContextForPrompt returns a selection of messages and summaries for a prompt
func (m *StandardContextManager) GetContextForPrompt(maxTokens int) (ContextSelection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no max tokens specified, use the config value
	if maxTokens <= 0 {
		maxTokens = m.config.MaxContextTokens
	}

	// Start with an empty selection
	selection := ContextSelection{
		MaxTokens: maxTokens,
	}

	// Add system message if available
	if m.systemPrompt != "" {
		sysMsg := EnhancedMessage{
			ID:         "system_prompt",
			Role:       "system",
			Content:    m.systemPrompt,
			Type:       MessageTypeSystem,
			CreatedAt:  time.Now(),
			Importance: ImportanceCritical,
		}

		// Count tokens for system message
		count, err := m.tokenCounter.CountMessageTokens(sysMsg)
		if err != nil {
			return selection, fmt.Errorf("failed to count system message tokens: %w", err)
		}
		sysMsg.TokenCount = count

		selection.SystemMessage = &sysMsg
		selection.TotalTokens += count
	}

	// Calculate budget for different components based on allocation
	budgetMap := make(map[string]int)
	remainingTokens := maxTokens - selection.TotalTokens

	for category, percentage := range m.config.TokenBudgetAllocation {
		if category == "system" {
			// System budget already used
			continue
		}
		budget := int(float64(maxTokens) * percentage)
		budgetMap[category] = budget
	}

	// If hierarchical context is enabled, allocate tokens for summaries
	if m.config.EnableHierarchicalContext && len(m.summaries) > 0 {
		summaryBudget := budgetMap["summary"]
		selectedSummaries := selectSummaries(m.summaries, summaryBudget)
		selection.Summaries = selectedSummaries

		// Calculate token usage
		summaryTokens := 0
		for _, s := range selectedSummaries {
			summaryTokens += s.TokenCount
		}

		selection.TotalTokens += summaryTokens
		remainingTokens -= summaryTokens
	}

	// Allocate remaining tokens to recent messages, prioritizing by importance
	// and recency
	recentBudget := budgetMap["recent"]
	if remainingTokens < recentBudget {
		recentBudget = remainingTokens
	}

	selectedMessages := selectMessages(m.messages, recentBudget)
	selection.Messages = selectedMessages

	// Calculate final token count
	messageTokens := 0
	for _, msg := range selectedMessages {
		messageTokens += msg.TokenCount
	}

	selection.TotalTokens += messageTokens

	return selection, nil
}

// GetFullHistory returns the full history of messages
func (m *StandardContextManager) GetFullHistory() []EnhancedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	history := make([]EnhancedMessage, len(m.messages))
	copy(history, m.messages)

	return history
}

// GetSummaries returns all summaries
func (m *StandardContextManager) GetSummaries() []Summary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	summaries := make([]Summary, len(m.summaries))
	copy(summaries, m.summaries)

	return summaries
}

// CreateSummary creates a summary of the specified messages
// This is a placeholder implementation - the real one will use the summarizer model
func (m *StandardContextManager) CreateSummary(ctx context.Context, messageIDs []string) (*Summary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify we have a summarizer backend
	if m.summarizerBackend == nil {
		return nil, errors.New("summarizer backend not configured")
	}

	// Find the messages to summarize
	var messagesToSummarize []EnhancedMessage
	for _, msg := range m.messages {
		for _, id := range messageIDs {
			if msg.ID == id {
				messagesToSummarize = append(messagesToSummarize, msg)
				break
			}
		}
	}

	if len(messagesToSummarize) == 0 {
		return nil, errors.New("no messages found to summarize")
	}

	// Sort messages by time
	sort.Slice(messagesToSummarize, func(i, j int) bool {
		return messagesToSummarize[i].CreatedAt.Before(messagesToSummarize[j].CreatedAt)
	})

	// This is where we would call the summarizer model
	// For now, we'll just create a placeholder summary

	// Calculate time span
	startTime := messagesToSummarize[0].CreatedAt
	endTime := messagesToSummarize[len(messagesToSummarize)-1].CreatedAt

	// Extract topics (combine all topics from all messages)
	topicMap := make(map[string]bool)
	for _, msg := range messagesToSummarize {
		for _, topic := range msg.Topics {
			topicMap[topic] = true
		}
	}

	topics := make([]string, 0, len(topicMap))
	for topic := range topicMap {
		topics = append(topics, topic)
	}

	// Create placeholder summary content
	// In a real implementation, this would be generated by the summarizer model
	content := fmt.Sprintf("Summary of %d messages from %s to %s",
		len(messagesToSummarize),
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339))

	summary := &Summary{
		ID:             fmt.Sprintf("summary_%d", time.Now().UnixNano()),
		Content:        content,
		CreatedAt:      time.Now(),
		Topics:         topics,
		SourceMessages: messageIDs,
		TimeSpan:       [2]time.Time{startTime, endTime},
	}

	// Count tokens
	summaryMsg := EnhancedMessage{Content: content}
	count, err := m.tokenCounter.CountMessageTokens(summaryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to count summary tokens: %w", err)
	}

	summary.TokenCount = count

	// Add to summaries
	m.summaries = append(m.summaries, *summary)

	return summary, nil
}

// Clear clears the context
func (m *StandardContextManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]EnhancedMessage, 0, 100)
	m.summaries = make([]Summary, 0, 10)

	return nil
}

// SaveToDisk persists the context to disk
func (m *StandardContextManager) SaveToDisk() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.config.EnablePersistence || m.config.PersistencePath == "" {
		return errors.New("persistence not enabled or path not set")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.config.PersistencePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create data to persist
	data := struct {
		Messages  []EnhancedMessage `json:"messages"`
		Summaries []Summary         `json:"summaries"`
		SavedAt   time.Time         `json:"saved_at"`
	}{
		Messages:  m.messages,
		Summaries: m.summaries,
		SavedAt:   time.Now(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.config.PersistencePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write context data: %w", err)
	}

	return nil
}

// LoadFromDisk loads the context from disk
func (m *StandardContextManager) LoadFromDisk() error {
	if !m.config.EnablePersistence || m.config.PersistencePath == "" {
		return errors.New("persistence not enabled or path not set")
	}

	// Check if file exists
	if _, err := os.Stat(m.config.PersistencePath); os.IsNotExist(err) {
		return fmt.Errorf("context file does not exist: %w", err)
	}

	// Read file
	jsonData, err := os.ReadFile(m.config.PersistencePath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	// Unmarshal data
	var data struct {
		Messages  []EnhancedMessage `json:"messages"`
		Summaries []Summary         `json:"summaries"`
		SavedAt   time.Time         `json:"saved_at"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	// Update context data
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = data.Messages
	m.summaries = data.Summaries

	return nil
}

// SetImportance sets the importance level of a message
func (m *StandardContextManager) SetImportance(messageID string, importance ImportanceLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, msg := range m.messages {
		if msg.ID == messageID {
			m.messages[i].Importance = importance
			return nil
		}
	}

	return fmt.Errorf("message with ID %s not found", messageID)
}

// AddTopicToMessage adds a topic to a message
func (m *StandardContextManager) AddTopicToMessage(messageID string, topic string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, msg := range m.messages {
		if msg.ID == messageID {
			// Check if topic already exists
			for _, t := range msg.Topics {
				if t == topic {
					return nil // Topic already exists
				}
			}

			// Add topic
			m.messages[i].Topics = append(m.messages[i].Topics, topic)
			return nil
		}
	}

	return fmt.Errorf("message with ID %s not found", messageID)
}

// AddTagToMessage adds a tag to a message
func (m *StandardContextManager) AddTagToMessage(messageID string, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, msg := range m.messages {
		if msg.ID == messageID {
			// Check if tag already exists
			for _, t := range msg.Tags {
				if t == tag {
					return nil // Tag already exists
				}
			}

			// Add tag
			m.messages[i].Tags = append(m.messages[i].Tags, tag)
			return nil
		}
	}

	return fmt.Errorf("message with ID %s not found", messageID)
}

// Search searches the context for messages matching the query
// This is a simple implementation that just does string matching
// A real implementation would use vector search or other more advanced techniques
func (m *StandardContextManager) Search(query string, maxResults int) ([]EnhancedMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []EnhancedMessage

	// Simple string matching for now
	for _, msg := range m.messages {
		if len(results) >= maxResults {
			break
		}

		// Check content
		if containsIgnoreCase(msg.Content, query) {
			results = append(results, msg)
			continue
		}

		// Check topics
		for _, topic := range msg.Topics {
			if containsIgnoreCase(topic, query) {
				results = append(results, msg)
				break
			}
		}

		// Check tags
		for _, tag := range msg.Tags {
			if containsIgnoreCase(tag, query) {
				results = append(results, msg)
				break
			}
		}
	}

	return results, nil
}

// PrepareBackendMessages converts the context selection to backend messages
func (m *StandardContextManager) PrepareBackendMessages(selection ContextSelection) ([]backend.Message, error) {
	var messages []backend.Message

	// Add system message if present
	if selection.SystemMessage != nil {
		messages = append(messages, backend.Message{
			Role:    "system",
			Content: selection.SystemMessage.Content,
		})
	}

	// Add summaries as a system message if present
	if len(selection.Summaries) > 0 {
		var summaryContent string
		for i, summary := range selection.Summaries {
			if i > 0 {
				summaryContent += "\n\n"
			}
			summaryContent += fmt.Sprintf("SUMMARY (%s to %s): %s",
				summary.TimeSpan[0].Format("2006-01-02 15:04:05"),
				summary.TimeSpan[1].Format("2006-01-02 15:04:05"),
				summary.Content)
		}

		if summaryContent != "" {
			messages = append(messages, backend.Message{
				Role:    "system",
				Content: "Previous conversation summaries:\n" + summaryContent,
			})
		}
	}

	// Add regular messages
	for _, msg := range selection.Messages {
		role := "user"
		if msg.Type == MessageTypeAssistant {
			role = "assistant"
		} else if msg.Type == MessageTypeSystem {
			role = "system"
		}

		messages = append(messages, backend.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	return messages, nil
}

// Helper functions

// selectMessages selects messages to include in the context based on
// importance, recency, and token budget
func selectMessages(messages []EnhancedMessage, tokenBudget int) []EnhancedMessage {
	if len(messages) == 0 {
		return nil
	}

	// Make a copy of messages to sort
	msgCopy := make([]EnhancedMessage, len(messages))
	copy(msgCopy, messages)

	// Sort by importance (highest first) and then by recency (newest first)
	sort.Slice(msgCopy, func(i, j int) bool {
		// First compare importance
		if msgCopy[i].Importance != msgCopy[j].Importance {
			return msgCopy[i].Importance > msgCopy[j].Importance
		}
		// Then compare recency
		return msgCopy[i].CreatedAt.After(msgCopy[j].CreatedAt)
	})

	// Select messages up to the token budget
	var selected []EnhancedMessage
	usedTokens := 0

	for _, msg := range msgCopy {
		// Always include critical messages
		if msg.Importance == ImportanceCritical {
			selected = append(selected, msg)
			usedTokens += msg.TokenCount
			continue
		}

		// Stop if we're over budget
		if usedTokens+msg.TokenCount > tokenBudget {
			break
		}

		selected = append(selected, msg)
		usedTokens += msg.TokenCount
	}

	// Sort the final selection by creation time to maintain conversation flow
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].CreatedAt.Before(selected[j].CreatedAt)
	})

	return selected
}

// selectSummaries selects summaries to include in the context based on
// relevance and token budget
func selectSummaries(summaries []Summary, tokenBudget int) []Summary {
	if len(summaries) == 0 {
		return nil
	}

	// Make a copy of summaries to sort
	sumCopy := make([]Summary, len(summaries))
	copy(sumCopy, summaries)

	// Sort by recency (newest first)
	sort.Slice(sumCopy, func(i, j int) bool {
		return sumCopy[i].CreatedAt.After(sumCopy[j].CreatedAt)
	})

	// Select summaries up to the token budget
	var selected []Summary
	usedTokens := 0

	for _, sum := range sumCopy {
		// Stop if we're over budget
		if usedTokens+sum.TokenCount > tokenBudget {
			break
		}

		selected = append(selected, sum)
		usedTokens += sum.TokenCount
	}

	// Sort the final selection by time span to maintain chronological order
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].TimeSpan[0].Before(selected[j].TimeSpan[0])
	})

	return selected
}

// containsIgnoreCase checks if a string contains a substring, ignoring case
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}
