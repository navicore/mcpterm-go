package context

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// MemoryLevel represents a level in hierarchical memory
type MemoryLevel string

const (
	// LongTermMemory represents persistent, summarized memory across sessions
	LongTermMemory MemoryLevel = "long_term"

	// MediumTermMemory represents summarized memory within the current session
	MediumTermMemory MemoryLevel = "medium_term"

	// ShortTermMemory represents recent, complete messages
	ShortTermMemory MemoryLevel = "short_term"

	// ImmediateMemory represents the most recent messages and current context
	ImmediateMemory MemoryLevel = "immediate"
)

// HierarchicalConfig contains configuration for hierarchical context management
type HierarchicalConfig struct {
	// Maximum tokens for each memory level
	MaxTokensPerLevel map[MemoryLevel]int

	// Time window for each memory level
	TimeWindowPerLevel map[MemoryLevel]time.Duration

	// Number of messages after which to trigger summarization for medium-term memory
	MediumTermSummarizationThreshold int

	// Long-term memory persistence settings
	LongTermPersistence bool
	LongTermFilePath    string

	// Whether to include system messages in hierarchical context
	IncludeSystemMessages bool

	// Importance thresholds for different memory levels
	ImportanceThresholds map[MemoryLevel]ImportanceLevel
}

// DefaultHierarchicalConfig returns the default hierarchical configuration
func DefaultHierarchicalConfig() HierarchicalConfig {
	return HierarchicalConfig{
		MaxTokensPerLevel: map[MemoryLevel]int{
			LongTermMemory:   1000,
			MediumTermMemory: 2000,
			ShortTermMemory:  3000,
			ImmediateMemory:  4000,
		},
		TimeWindowPerLevel: map[MemoryLevel]time.Duration{
			LongTermMemory:   30 * 24 * time.Hour, // 30 days
			MediumTermMemory: 24 * time.Hour,      // 1 day
			ShortTermMemory:  1 * time.Hour,       // 1 hour
			ImmediateMemory:  5 * time.Minute,     // 5 minutes
		},
		MediumTermSummarizationThreshold: 20,
		LongTermPersistence:              true,
		LongTermFilePath:                 "",
		IncludeSystemMessages:            true,
		ImportanceThresholds: map[MemoryLevel]ImportanceLevel{
			LongTermMemory:   ImportanceHigh,
			MediumTermMemory: ImportanceMedium,
			ShortTermMemory:  ImportanceLow,
			ImmediateMemory:  ImportanceLow,
		},
	}
}

// HierarchicalContext manages hierarchical context structures
type HierarchicalContext struct {
	config         HierarchicalConfig
	contextManager *StandardContextManager
}

// NewHierarchicalContext creates a new hierarchical context manager
func NewHierarchicalContext(config HierarchicalConfig, contextManager *StandardContextManager) *HierarchicalContext {
	return &HierarchicalContext{
		config:         config,
		contextManager: contextManager,
	}
}

// GetConfig returns the hierarchical context configuration
func (h *HierarchicalContext) GetConfig() HierarchicalConfig {
	return h.config
}

// GetHierarchicalSelection returns a context selection organized by memory levels
func (h *HierarchicalContext) GetHierarchicalSelection(maxTokens int) (ContextSelection, error) {
	// Debug log removed to avoid interfering with TUI
	// Start with an empty selection
	selection := ContextSelection{
		MaxTokens: maxTokens,
	}

	// Add system message if available
	systemPrompt := h.contextManager.systemPrompt
	if systemPrompt != "" {
		sysMsg := EnhancedMessage{
			ID:         "system_prompt",
			Role:       "system",
			Content:    systemPrompt,
			Type:       MessageTypeSystem,
			CreatedAt:  time.Now(),
			Importance: ImportanceCritical,
		}

		// Count tokens for system message
		count, err := h.contextManager.tokenCounter.CountMessageTokens(sysMsg)
		if err != nil {
			return selection, fmt.Errorf("failed to count system message tokens: %w", err)
		}
		sysMsg.TokenCount = count

		selection.SystemMessage = &sysMsg
		selection.TotalTokens += count
	}

	// Calculate remaining tokens
	remainingTokens := maxTokens - selection.TotalTokens

	// Get all messages and summaries
	allMessages := h.contextManager.GetFullHistory()
	allSummaries := h.contextManager.GetSummaries()

	// Calculate token budget for each memory level
	budgets := h.calculateBudgets(remainingTokens)

	// Allocate messages and summaries for each level
	now := time.Now()

	// 1. Long-term memory (oldest summaries)
	longTermSummaries := h.filterSummariesByTimeWindow(allSummaries, LongTermMemory, now)
	longTermSelection := h.selectSummaries(longTermSummaries, budgets[LongTermMemory])
	selection.Summaries = append(selection.Summaries, longTermSelection...)
	selection.TotalTokens += sumTokens(longTermSelection)

	// 2. Medium-term memory (more recent summaries)
	mediumTermSummaries := h.filterSummariesByTimeWindow(allSummaries, MediumTermMemory, now)
	mediumTermSelection := h.selectSummaries(mediumTermSummaries, budgets[MediumTermMemory])
	selection.Summaries = append(selection.Summaries, mediumTermSelection...)
	selection.TotalTokens += sumTokens(mediumTermSelection)

	// 3. Short-term memory (recent messages, potentially summarized)
	shortTermMessages := h.filterMessagesByTimeWindow(allMessages, ShortTermMemory, now)
	shortTermSelection := h.selectMessages(shortTermMessages, budgets[ShortTermMemory])
	selection.Messages = append(selection.Messages, shortTermSelection...)
	selection.TotalTokens += sumMessageTokens(shortTermSelection)

	// 4. Immediate memory (most recent messages, always full)
	immediateMessages := h.filterMessagesByTimeWindow(allMessages, ImmediateMemory, now)
	immediateSelection := h.selectMessages(immediateMessages, budgets[ImmediateMemory])
	selection.Messages = append(selection.Messages, immediateSelection...)
	selection.TotalTokens += sumMessageTokens(immediateSelection)

	// Ensure messages are in chronological order
	sort.Slice(selection.Messages, func(i, j int) bool {
		return selection.Messages[i].CreatedAt.Before(selection.Messages[j].CreatedAt)
	})

	// Ensure summaries are in chronological order
	sort.Slice(selection.Summaries, func(i, j int) bool {
		return selection.Summaries[i].TimeSpan[0].Before(selection.Summaries[j].TimeSpan[0])
	})

	// If we're using persistence and have a system message, add a note about persistence
	if h.config.LongTermPersistence && selection.SystemMessage != nil &&
		len(selection.Messages) > 1 {
		// Only add if it's not already there
		if !strings.Contains(selection.SystemMessage.Content, "persistent session") {
			persistenceNote := "\n\nIMPORTANT: This is a persistent session. You have access to previous conversation history. Use this context when responding to the user."
			selection.SystemMessage.Content += persistenceNote
			// Debug logging removed to avoid interfering with TUI
		}
	}

	// Print debug info about what we're sending
	if len(selection.Messages) > 0 || len(selection.Summaries) > 0 {
		// Debug logging removed to avoid interfering with TUI
	}

	return selection, nil
}

// CheckSummarizationNeeded checks if summarization is needed for any memory level
func (h *HierarchicalContext) CheckSummarizationNeeded() (bool, MemoryLevel) {
	allMessages := h.contextManager.GetFullHistory()
	now := time.Now()

	// Check medium-term memory threshold
	mediumTermMessages := h.filterMessagesByTimeWindow(allMessages, MediumTermMemory, now)
	if len(mediumTermMessages) >= h.config.MediumTermSummarizationThreshold {
		return true, MediumTermMemory
	}

	// Add logic for other levels if needed

	return false, ""
}

// GetMessagesToSummarize returns messages that should be summarized for a given memory level
func (h *HierarchicalContext) GetMessagesToSummarize(level MemoryLevel) []EnhancedMessage {
	allMessages := h.contextManager.GetFullHistory()
	now := time.Now()

	switch level {
	case MediumTermMemory:
		// Get messages in medium-term window excluding most recent (immediate) messages
		mediumTermTime := now.Add(-h.config.TimeWindowPerLevel[MediumTermMemory])
		immediateTime := now.Add(-h.config.TimeWindowPerLevel[ImmediateMemory])

		var toSummarize []EnhancedMessage
		for _, msg := range allMessages {
			// Include messages older than immediate but within medium-term window
			if msg.CreatedAt.After(mediumTermTime) && msg.CreatedAt.Before(immediateTime) {
				toSummarize = append(toSummarize, msg)
			}
		}

		return toSummarize

	case LongTermMemory:
		// Get messages older than medium-term window but within long-term window
		longTermTime := now.Add(-h.config.TimeWindowPerLevel[LongTermMemory])
		mediumTermTime := now.Add(-h.config.TimeWindowPerLevel[MediumTermMemory])

		var toSummarize []EnhancedMessage
		for _, msg := range allMessages {
			// Include messages older than medium-term but within long-term window
			if msg.CreatedAt.After(longTermTime) && msg.CreatedAt.Before(mediumTermTime) {
				toSummarize = append(toSummarize, msg)
			}
		}

		return toSummarize
	}

	return nil
}

// Helper methods

// calculateBudgets calculates the token budget for each memory level
func (h *HierarchicalContext) calculateBudgets(totalTokens int) map[MemoryLevel]int {
	result := make(map[MemoryLevel]int)

	// Use fixed proportions for now
	// Could be made dynamic based on conversation state
	result[LongTermMemory] = int(float64(totalTokens) * 0.10)   // 10%
	result[MediumTermMemory] = int(float64(totalTokens) * 0.25) // 25%
	result[ShortTermMemory] = int(float64(totalTokens) * 0.30)  // 30%
	result[ImmediateMemory] = int(float64(totalTokens) * 0.35)  // 35%

	return result
}

// filterMessagesByTimeWindow filters messages based on time window and importance
func (h *HierarchicalContext) filterMessagesByTimeWindow(messages []EnhancedMessage, level MemoryLevel, now time.Time) []EnhancedMessage {
	window, hasWindow := h.config.TimeWindowPerLevel[level]
	if !hasWindow {
		return nil
	}

	importanceThreshold, hasThreshold := h.config.ImportanceThresholds[level]
	if !hasThreshold {
		importanceThreshold = ImportanceLow // Default
	}

	var result []EnhancedMessage
	windowStart := now.Add(-window)

	for _, msg := range messages {
		// Skip system messages if configured
		if !h.config.IncludeSystemMessages && msg.Type == MessageTypeSystem {
			continue
		}

		// Special handling for loaded messages from disk - always include them
		hasLoadedTag := false
		for _, tag := range msg.Tags {
			if tag == "loaded_from_disk" {
				hasLoadedTag = true
				break
			}
		}

		// Include if:
		// 1. Message has the loaded_from_disk tag, or
		// 2. Message is within time window, or
		// 3. Message has sufficient importance
		if hasLoadedTag || msg.CreatedAt.After(windowStart) || msg.Importance >= importanceThreshold {
			result = append(result, msg)
		}
	}

	return result
}

// filterSummariesByTimeWindow filters summaries based on time window
func (h *HierarchicalContext) filterSummariesByTimeWindow(summaries []Summary, level MemoryLevel, now time.Time) []Summary {
	window, hasWindow := h.config.TimeWindowPerLevel[level]
	if !hasWindow {
		return nil
	}

	var result []Summary
	windowStart := now.Add(-window)

	for _, summary := range summaries {
		// Special handling for loaded summaries from disk - always include them
		hasLoadedTag := false
		for _, tag := range summary.Tags {
			if tag == "loaded_from_disk" {
				hasLoadedTag = true
				break
			}
		}

		// Include if:
		// 1. Summary has the loaded_from_disk tag, or
		// 2. Summary time span overlaps with the window
		if hasLoadedTag || summary.TimeSpan[1].After(windowStart) {
			result = append(result, summary)
		}
	}

	return result
}

// selectMessages selects messages based on priority and token budget
func (h *HierarchicalContext) selectMessages(messages []EnhancedMessage, tokenBudget int) []EnhancedMessage {
	if len(messages) == 0 || tokenBudget <= 0 {
		return nil
	}

	// Sort by importance (highest first) then by recency (newest first)
	sortedMsgs := make([]EnhancedMessage, len(messages))
	copy(sortedMsgs, messages)

	sort.Slice(sortedMsgs, func(i, j int) bool {
		// First compare importance
		if sortedMsgs[i].Importance != sortedMsgs[j].Importance {
			return sortedMsgs[i].Importance > sortedMsgs[j].Importance
		}
		// Then compare recency
		return sortedMsgs[i].CreatedAt.After(sortedMsgs[j].CreatedAt)
	})

	// Select messages within token budget
	var selected []EnhancedMessage
	usedTokens := 0

	for _, msg := range sortedMsgs {
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

	// Re-sort by time for proper sequence
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].CreatedAt.Before(selected[j].CreatedAt)
	})

	return selected
}

// selectSummaries selects summaries based on relevance and token budget
func (h *HierarchicalContext) selectSummaries(summaries []Summary, tokenBudget int) []Summary {
	if len(summaries) == 0 || tokenBudget <= 0 {
		return nil
	}

	// Sort by creation time (newest first)
	sortedSummaries := make([]Summary, len(summaries))
	copy(sortedSummaries, summaries)

	sort.Slice(sortedSummaries, func(i, j int) bool {
		return sortedSummaries[i].CreatedAt.After(sortedSummaries[j].CreatedAt)
	})

	// Select summaries within token budget
	var selected []Summary
	usedTokens := 0

	for _, summary := range sortedSummaries {
		// Stop if we're over budget
		if usedTokens+summary.TokenCount > tokenBudget {
			break
		}

		selected = append(selected, summary)
		usedTokens += summary.TokenCount
	}

	// Re-sort by time span for proper sequence
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].TimeSpan[0].Before(selected[j].TimeSpan[0])
	})

	return selected
}

// Helper functions for token counting
func sumTokens(summaries []Summary) int {
	total := 0
	for _, s := range summaries {
		total += s.TokenCount
	}
	return total
}

func sumMessageTokens(messages []EnhancedMessage) int {
	total := 0
	for _, m := range messages {
		total += m.TokenCount
	}
	return total
}
