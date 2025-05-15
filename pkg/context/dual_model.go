package context

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// File logger setup - All logging goes to a single file in /tmp
var dualModelLogger *log.Logger

func init() {
	// Create or append to log file in /tmp
	logFile, err := os.OpenFile("/tmp/mcpterm_context.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If we can't open the log file, create a dummy logger that discards output
		dualModelLogger = log.New(os.NewFile(0, os.DevNull), "", 0)
		return
	}

	// Create logger with timestamp and prefix
	dualModelLogger = log.New(logFile, "DUAL_MODEL: ", log.LstdFlags|log.Lshortfile)
}

// DualModelConfig contains configuration for the dual model manager
type DualModelConfig struct {
	// Primary model configuration (e.g., Claude 3.7 Sonnet)
	PrimaryModelID      string
	PrimaryModelOptions map[string]any

	// Summarizer model configuration (e.g., Claude 3.5 Haiku)
	SummarizerModelID      string
	SummarizerModelOptions map[string]any

	// Summarization settings
	SummarizeAfterMessages   int     // Trigger summarization after this many messages
	SummaryPrompt            string  // Prompt template for summarization
	SummarizationTemperature float64 // Temperature for summarization
	MaxSummaryTokens         int     // Maximum tokens for summary generation

	// Queue settings
	MaxQueuedSummarizations int // Maximum number of queued summarization tasks

	// Performance settings
	AsyncSummarization bool // Whether to run summarization asynchronously
}

// DefaultDualModelConfig returns the default dual model configuration
func DefaultDualModelConfig() DualModelConfig {
	return DualModelConfig{
		PrimaryModelID:           "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
		PrimaryModelOptions:      make(map[string]any),
		SummarizerModelID:        "anthropic.claude-3-haiku-20240307-v1:0",
		SummarizerModelOptions:   make(map[string]any),
		SummarizeAfterMessages:   20,
		SummaryPrompt:            defaultSummaryPrompt,
		SummarizationTemperature: 0.3, // Lower temperature for more focused summaries
		MaxSummaryTokens:         500,
		MaxQueuedSummarizations:  5,
		AsyncSummarization:       true,
	}
}

// defaultSummaryPrompt is the default prompt for summarization
const defaultSummaryPrompt = `You are a helpful AI assistant tasked with summarizing conversation context.
Your goal is to create a concise, informative summary that captures the key points,
decisions, requirements, and important context from the conversation.

Guidelines:
1. Focus on extracting the most important information
2. Preserve specific details like code snippets, error messages, and technical requirements
3. Include key user preferences and constraints mentioned
4. Maintain technical accuracy and precision
5. Identify the main topics discussed
6. Be concise but comprehensive

Here's the conversation to summarize:

{{CONVERSATION}}

Please provide a summary following these guidelines. Be specific and factual, not general.`

// SummarizationTask represents a pending summarization task
type SummarizationTask struct {
	ID         string
	MessageIDs []string
	CreatedAt  time.Time
	Priority   int // Higher numbers = higher priority
	InProgress bool
}

// DualModelManager handles the dual model architecture for context management
type DualModelManager struct {
	config             DualModelConfig
	primaryBackend     backend.Backend
	summarizerBackend  backend.Backend
	contextManager     *StandardContextManager
	summarizationQueue []*SummarizationTask
	queueMutex         sync.Mutex
	workerRunning      bool
	workerWaitGroup    sync.WaitGroup
	shutdownCh         chan struct{}
}

// NewDualModelManager creates a new dual model manager
func NewDualModelManager(config DualModelConfig, contextManager *StandardContextManager) (*DualModelManager, error) {
	if contextManager == nil {
		return nil, errors.New("context manager is required")
	}

	return &DualModelManager{
		config:             config,
		contextManager:     contextManager,
		summarizationQueue: make([]*SummarizationTask, 0, 10),
		shutdownCh:         make(chan struct{}),
	}, nil
}

// Initialize initializes the dual model manager by setting up the backends
func (m *DualModelManager) Initialize() error {
	// Create primary backend
	primaryConfig := backend.Config{
		Type:        backend.BackendAWSBedrock,
		ModelID:     m.config.PrimaryModelID,
		MaxTokens:   4096,
		Temperature: 0.7,
		Options:     m.config.PrimaryModelOptions,
	}

	var err error
	m.primaryBackend, err = backend.NewBackend(primaryConfig)
	if err != nil {
		return fmt.Errorf("failed to create primary backend: %w", err)
	}

	// Create summarizer backend
	summarizerConfig := backend.Config{
		Type:        backend.BackendAWSBedrock,
		ModelID:     m.config.SummarizerModelID,
		MaxTokens:   2048,
		Temperature: m.config.SummarizationTemperature,
		Options:     m.config.SummarizerModelOptions,
	}

	m.summarizerBackend, err = backend.NewBackend(summarizerConfig)
	if err != nil {
		return fmt.Errorf("failed to create summarizer backend: %w", err)
	}

	// Set backends on context manager
	m.contextManager.SetBackends(m.primaryBackend, m.summarizerBackend)

	// Start background worker if async summarization is enabled
	if m.config.AsyncSummarization {
		m.startBackgroundWorker()
	}

	return nil
}

// startBackgroundWorker starts the background worker for processing summarization tasks
func (m *DualModelManager) startBackgroundWorker() {
	if m.workerRunning {
		return
	}

	m.workerRunning = true
	m.workerWaitGroup.Add(1)

	go func() {
		defer m.workerWaitGroup.Done()

		for {
			select {
			case <-m.shutdownCh:
				// Shutdown signal received
				return

			case <-time.After(1 * time.Second):
				// Check for tasks every second
				task := m.getNextSummarizationTask()
				if task != nil {
					m.processSummarizationTask(task)
				}
			}
		}
	}()
}

// getNextSummarizationTask gets the next task from the queue
func (m *DualModelManager) getNextSummarizationTask() *SummarizationTask {
	m.queueMutex.Lock()
	defer m.queueMutex.Unlock()

	if len(m.summarizationQueue) == 0 {
		return nil
	}

	// Find highest priority task not in progress
	highestIdx := -1
	highestPriority := -1

	for i, task := range m.summarizationQueue {
		if !task.InProgress && task.Priority > highestPriority {
			highestPriority = task.Priority
			highestIdx = i
		}
	}

	if highestIdx >= 0 {
		// Mark as in progress
		m.summarizationQueue[highestIdx].InProgress = true
		return m.summarizationQueue[highestIdx]
	}

	return nil
}

// processSummarizationTask processes a summarization task
func (m *DualModelManager) processSummarizationTask(task *SummarizationTask) {
	ctx := context.Background()

	// Call the summarization function
	_, err := m.createSummary(ctx, task.MessageIDs)

	// Remove task from queue (regardless of success or failure)
	m.queueMutex.Lock()
	defer m.queueMutex.Unlock()

	for i, t := range m.summarizationQueue {
		if t.ID == task.ID {
			// Remove from queue
			m.summarizationQueue = append(m.summarizationQueue[:i], m.summarizationQueue[i+1:]...)
			break
		}
	}

	// Log errors but don't fail the background task
	if err != nil {
		dualModelLogger.Printf("Error creating summary: %v", err)
	}
}

// QueueSummarization queues a summarization task
func (m *DualModelManager) QueueSummarization(messageIDs []string, priority int) error {
	if len(messageIDs) == 0 {
		return errors.New("no messages to summarize")
	}

	m.queueMutex.Lock()
	defer m.queueMutex.Unlock()

	// Check if queue is full
	if len(m.summarizationQueue) >= m.config.MaxQueuedSummarizations {
		return errors.New("summarization queue is full")
	}

	// Create new task
	task := &SummarizationTask{
		ID:         fmt.Sprintf("sum_%d", time.Now().UnixNano()),
		MessageIDs: messageIDs,
		CreatedAt:  time.Now(),
		Priority:   priority,
		InProgress: false,
	}

	// Add to queue
	m.summarizationQueue = append(m.summarizationQueue, task)

	return nil
}

// createSummary creates a summary of the specified messages using the summarizer model
func (m *DualModelManager) createSummary(ctx context.Context, messageIDs []string) (*Summary, error) {
	// Get the messages to summarize
	var messagesToSummarize []EnhancedMessage
	allMessages := m.contextManager.GetFullHistory()

	// Filter messages by ID
	messageMap := make(map[string]bool)
	for _, id := range messageIDs {
		messageMap[id] = true
	}

	for _, msg := range allMessages {
		if messageMap[msg.ID] {
			messagesToSummarize = append(messagesToSummarize, msg)
		}
	}

	if len(messagesToSummarize) == 0 {
		return nil, errors.New("no messages found to summarize")
	}

	// Format the conversation text
	conversationText := formatConversationForSummary(messagesToSummarize)

	// Replace the placeholder in the prompt with the conversation
	summaryPrompt := m.config.SummaryPrompt
	summaryPrompt = strings.Replace(summaryPrompt, "{{CONVERSATION}}", conversationText, 1)

	// Create the request for the summarizer
	req := backend.ChatRequest{
		Messages: []backend.Message{
			{
				Role:    "user",
				Content: summaryPrompt,
			},
		},
		MaxTokens:   m.config.MaxSummaryTokens,
		Temperature: m.config.SummarizationTemperature,
	}

	// Send the request to the summarizer
	resp, err := m.summarizerBackend.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("summarizer error: %w", err)
	}

	// Extract topics from the summary (could be enhanced with more sophisticated topic extraction)
	topics := extractTopics(resp.Content)

	// Calculate time span
	startTime := messagesToSummarize[0].CreatedAt
	endTime := messagesToSummarize[0].CreatedAt

	for _, msg := range messagesToSummarize {
		if msg.CreatedAt.Before(startTime) {
			startTime = msg.CreatedAt
		}
		if msg.CreatedAt.After(endTime) {
			endTime = msg.CreatedAt
		}
	}

	// Create the summary object
	summary := &Summary{
		ID:             fmt.Sprintf("summary_%d", time.Now().UnixNano()),
		Content:        resp.Content,
		CreatedAt:      time.Now(),
		Topics:         topics,
		SourceMessages: messageIDs,
		TimeSpan:       [2]time.Time{startTime, endTime},
		TokenCount:     resp.Usage["completion_tokens"], // Use actual token count from response
	}

	// Add the summary to the context manager
	// Note: This doesn't trigger a recursive summarization since it's not a message
	m.contextManager.summaries = append(m.contextManager.summaries, *summary)

	return summary, nil
}

// SummarizeHistory summarizes the recent history (synchronously)
func (m *DualModelManager) SummarizeHistory(ctx context.Context, messageCount int) (*Summary, error) {
	// Get the most recent N messages
	allMessages := m.contextManager.GetFullHistory()

	if len(allMessages) == 0 {
		return nil, errors.New("no messages to summarize")
	}

	// Limit to requested count
	startIdx := 0
	if len(allMessages) > messageCount {
		startIdx = len(allMessages) - messageCount
	}

	messagesToSummarize := allMessages[startIdx:]

	// Get message IDs
	messageIDs := make([]string, len(messagesToSummarize))
	for i, msg := range messagesToSummarize {
		messageIDs[i] = msg.ID
	}

	// Create summary
	return m.createSummary(ctx, messageIDs)
}

// CheckSummarizationNeeded checks if summarization is needed based on the configuration
func (m *DualModelManager) CheckSummarizationNeeded() bool {
	// Get all messages
	messages := m.contextManager.GetFullHistory()

	// Count user and assistant messages (excluding system messages)
	var messageCount int
	for _, msg := range messages {
		if msg.Type == MessageTypeUser || msg.Type == MessageTypeAssistant {
			messageCount++
		}
	}

	// Check if we've reached the threshold
	return messageCount >= m.config.SummarizeAfterMessages
}

// AutoSummarize automatically summarizes history if needed
// Returns true if summarization was triggered, false otherwise
func (m *DualModelManager) AutoSummarize() bool {
	if !m.CheckSummarizationNeeded() {
		return false
	}

	// Get all messages
	allMessages := m.contextManager.GetFullHistory()
	if len(allMessages) == 0 {
		return false
	}

	// Determine which messages to summarize (the oldest half)
	count := len(allMessages) / 2
	if count < 5 {
		count = 5 // Ensure we have at least 5 messages to summarize
	}

	if count > len(allMessages) {
		count = len(allMessages)
	}

	messagesToSummarize := allMessages[:count]

	// Get message IDs
	messageIDs := make([]string, len(messagesToSummarize))
	for i, msg := range messagesToSummarize {
		messageIDs[i] = msg.ID
	}

	// Queue summarization with medium priority
	err := m.QueueSummarization(messageIDs, 5)
	return err == nil
}

// Shutdown stops the background worker
func (m *DualModelManager) Shutdown() {
	if m.workerRunning {
		close(m.shutdownCh)
		m.workerWaitGroup.Wait()
		m.workerRunning = false
	}

	// Close backends
	if m.primaryBackend != nil {
		m.primaryBackend.Close()
	}

	if m.summarizerBackend != nil {
		m.summarizerBackend.Close()
	}
}

// Helper functions

// formatConversationForSummary formats messages for the summarization prompt
func formatConversationForSummary(messages []EnhancedMessage) string {
	var result strings.Builder

	for i, msg := range messages {
		role := "User"
		if msg.Type == MessageTypeAssistant {
			role = "Assistant"
		} else if msg.Type == MessageTypeSystem {
			role = "System"
		}

		if i > 0 {
			result.WriteString("\n\n")
		}

		result.WriteString(fmt.Sprintf("[%s]: %s", role, msg.Content))
	}

	return result.String()
}

// extractTopics extracts potential topics from a summary
// This is a simple implementation that could be enhanced
func extractTopics(summary string) []string {
	// Split into sentences and take the first few words of each as potential topics
	sentences := strings.Split(summary, ".")

	topics := make(map[string]bool)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Split into words
		words := strings.Fields(sentence)

		// Take the first few significant words as a topic
		if len(words) > 2 {
			topic := strings.Join(words[:3], " ")
			topics[topic] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(topics))
	for topic := range topics {
		result = append(result, topic)
	}

	// Limit to 5 topics
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}
