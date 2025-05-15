package context

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// SummarizerConfig contains configuration for the summarizer
type SummarizerConfig struct {
	// Base prompt template for summarization
	BasePrompt string

	// Maximum tokens for the generated summary
	MaxSummaryTokens int

	// Temperature for summary generation
	Temperature float64

	// Whether to preserve code blocks in the summary
	PreserveCodeBlocks bool

	// Maximum original message count to include in a single summarization
	MaxMessagesToSummarize int

	// Whether to include message metadata in the prompt
	IncludeMessageMetadata bool
}

// DefaultSummarizerConfig returns the default summarizer configuration
func DefaultSummarizerConfig() SummarizerConfig {
	return SummarizerConfig{
		BasePrompt:             defaultSummarizerPrompt,
		MaxSummaryTokens:       500,
		Temperature:            0.3,
		PreserveCodeBlocks:     true,
		MaxMessagesToSummarize: 30,
		IncludeMessageMetadata: true,
	}
}

// defaultSummarizerPrompt is the default prompt for the summarizer
const defaultSummarizerPrompt = `You are a context summarization specialist for an AI assistant. Your task is to create a concise yet comprehensive summary of the following conversation that will be used as context for future interactions.

Focus on:
1. Key user requirements and constraints
2. Important decisions made in the conversation
3. Technical details that would be valuable for future reference
4. Any explicit preferences stated by the user
5. Specific code snippets or technical concepts discussed

The summary should prioritize factual information over subjective content. Be specific and concrete rather than general.

{{CODE_PRESERVATION_INSTRUCTIONS}}

Here's the conversation to summarize:

{{CONVERSATION}}

Provide a summary that captures the essential context while being concise.`

// codePreservationInstructions are added to the prompt when code should be preserved
const codePreservationInstructions = `Important: Preserve all code blocks in their entirety in your summary. Code snippets are critical context for the AI assistant. Format them with proper markdown code blocks using triple backticks.`

// Summarizer handles creating summaries of conversation history
type Summarizer struct {
	config    SummarizerConfig
	backend   backend.Backend
	codeRegex *regexp.Regexp
}

// NewSummarizer creates a new summarizer
func NewSummarizer(config SummarizerConfig, backend backend.Backend) *Summarizer {
	return &Summarizer{
		config:    config,
		backend:   backend,
		codeRegex: regexp.MustCompile("```[a-zA-Z]*\\s*[\\s\\S]*?```"),
	}
}

// CreateSummary creates a summary of the given messages
func (s *Summarizer) CreateSummary(ctx context.Context, messages []EnhancedMessage) (*Summary, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to summarize")
	}

	// Limit the number of messages to summarize
	messagesToSummarize := messages
	if len(messagesToSummarize) > s.config.MaxMessagesToSummarize {
		messagesToSummarize = messages[len(messages)-s.config.MaxMessagesToSummarize:]
	}

	// Sort messages by creation time
	sort.Slice(messagesToSummarize, func(i, j int) bool {
		return messagesToSummarize[i].CreatedAt.Before(messagesToSummarize[j].CreatedAt)
	})

	// Format the conversation text
	conversation := s.formatConversation(messagesToSummarize)

	// Create the summary prompt
	prompt := s.createSummaryPrompt(conversation)

	// Create the request for the model
	req := backend.ChatRequest{
		Messages: []backend.Message{
			{
				Role:    "system",
				Content: "You are a context summarization specialist for an AI assistant.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   s.config.MaxSummaryTokens,
		Temperature: s.config.Temperature,
	}

	// Send the request to the model
	resp, err := s.backend.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("summarization error: %w", err)
	}

	// Extract topics from the summary
	topics := s.extractTopics(resp.Content)

	// Calculate time span
	var startTime, endTime time.Time
	if len(messagesToSummarize) > 0 {
		startTime = messagesToSummarize[0].CreatedAt
		endTime = messagesToSummarize[len(messagesToSummarize)-1].CreatedAt
	} else {
		startTime = time.Now()
		endTime = time.Now()
	}

	// Get message IDs
	messageIDs := make([]string, len(messagesToSummarize))
	for i, msg := range messagesToSummarize {
		messageIDs[i] = msg.ID
	}

	// Create the summary object
	summary := &Summary{
		ID:             fmt.Sprintf("summary_%d", time.Now().UnixNano()),
		Content:        resp.Content,
		CreatedAt:      time.Now(),
		Topics:         topics,
		SourceMessages: messageIDs,
		TimeSpan:       [2]time.Time{startTime, endTime},
		TokenCount:     resp.Usage["completion_tokens"], // Use actual token count if available
	}

	// If token count is not available, estimate it
	if summary.TokenCount == 0 {
		tokenCounter := NewSimpleTokenCounter()
		count, _ := tokenCounter.CountTokens(resp.Content)
		summary.TokenCount = count
	}

	return summary, nil
}

// createSummaryPrompt creates the prompt for summarization
func (s *Summarizer) createSummaryPrompt(conversation string) string {
	prompt := s.config.BasePrompt

	// Add code preservation instructions if needed
	codeInstructions := ""
	if s.config.PreserveCodeBlocks {
		codeInstructions = codePreservationInstructions
	}

	prompt = strings.Replace(prompt, "{{CODE_PRESERVATION_INSTRUCTIONS}}", codeInstructions, 1)
	prompt = strings.Replace(prompt, "{{CONVERSATION}}", conversation, 1)

	return prompt
}

// formatConversation formats the conversation for the summarization prompt
func (s *Summarizer) formatConversation(messages []EnhancedMessage) string {
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

		// Add message metadata if configured
		if s.config.IncludeMessageMetadata {
			result.WriteString(fmt.Sprintf("[%s - %s", role, msg.CreatedAt.Format(time.RFC3339)))

			// Add importance information
			switch msg.Importance {
			case ImportanceCritical:
				result.WriteString(" - CRITICAL")
			case ImportanceHigh:
				result.WriteString(" - HIGH")
			case ImportanceMedium:
				result.WriteString(" - MEDIUM")
			}

			// Add tags if available
			if len(msg.Tags) > 0 {
				result.WriteString(" - Tags: " + strings.Join(msg.Tags, ", "))
			}

			result.WriteString("]: ")
		} else {
			result.WriteString(fmt.Sprintf("[%s]: ", role))
		}

		result.WriteString(msg.Content)
	}

	return result.String()
}

// extractTopics extracts topics from the summary content
func (s *Summarizer) extractTopics(summary string) []string {
	// Split into sentences and analyze
	sentences := strings.Split(summary, ".")

	// Map to track topics
	topicMap := make(map[string]int)

	// Patterns for potentially important topics
	requirementPattern := regexp.MustCompile("(?i)required|must|should|needs to|has to")
	preferencePattern := regexp.MustCompile("(?i)prefer|want|like|desire|wish")
	technicalPattern := regexp.MustCompile("(?i)code|function|method|class|api|endpoint|database|authentication|implementation")

	// Process each sentence
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 10 {
			continue
		}

		words := strings.Fields(sentence)
		if len(words) < 3 {
			continue
		}

		// Create a potential topic from the first few words
		potentialTopic := strings.Join(words[:3], " ")

		// Score the topic based on patterns
		score := 1
		if requirementPattern.MatchString(sentence) {
			score += 3
		}
		if preferencePattern.MatchString(sentence) {
			score += 2
		}
		if technicalPattern.MatchString(sentence) {
			score += 2
		}

		// Store or update the topic score
		if currentScore, exists := topicMap[potentialTopic]; exists {
			topicMap[potentialTopic] = currentScore + score
		} else {
			topicMap[potentialTopic] = score
		}
	}

	// Convert map to slice of topics
	type topicScore struct {
		topic string
		score int
	}

	var topicScores []topicScore
	for topic, score := range topicMap {
		topicScores = append(topicScores, topicScore{topic, score})
	}

	// Sort by score (highest first)
	sort.Slice(topicScores, func(i, j int) bool {
		return topicScores[i].score > topicScores[j].score
	})

	// Extract top topics
	var topics []string
	for i, ts := range topicScores {
		if i >= 5 { // Limit to 5 topics
			break
		}
		topics = append(topics, ts.topic)
	}

	return topics
}

// ExtractCodeBlocks extracts code blocks from content
func (s *Summarizer) ExtractCodeBlocks(content string) []string {
	matches := s.codeRegex.FindAllString(content, -1)
	return matches
}
