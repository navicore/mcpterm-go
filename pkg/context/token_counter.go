package context

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// SimpleTokenCounter is a basic implementation of TokenCounter that uses
// word count and multipliers to estimate tokens.
// This is a placeholder for a proper tokenizer-based implementation.
type SimpleTokenCounter struct {
	// A rough multiplier for estimating tokens based on word count
	// Different models have different tokenization strategies
	TokensPerWordMultiplier float64
}

// NewSimpleTokenCounter creates a new SimpleTokenCounter
func NewSimpleTokenCounter() *SimpleTokenCounter {
	return &SimpleTokenCounter{
		// Claude models tend to use about 1.3 tokens per word on average
		TokensPerWordMultiplier: 1.3,
	}
}

// CountTokens returns an estimate of the number of tokens in the text
func (c *SimpleTokenCounter) CountTokens(text string) (int, error) {
	// Split the text into words
	words := strings.Fields(text)
	wordCount := len(words)

	// Count characters for additional heuristics
	charCount := utf8.RuneCountInString(text)

	// Handle empty or very short text
	if charCount == 0 {
		return 0, nil
	}

	// Base token estimate using word count
	tokenEstimate := float64(wordCount) * c.TokensPerWordMultiplier

	// Adjust for special characters and punctuation (rough heuristic)
	specialChars := countSpecialChars(text)
	tokenEstimate += float64(specialChars) * 0.5

	// Adjust for code blocks (code often has more tokens per word)
	if containsCodeBlock(text) {
		tokenEstimate *= 1.2
	}

	// Convert to integer and ensure at least 1 token for non-empty text
	tokens := int(tokenEstimate)
	if tokens < 1 && charCount > 0 {
		tokens = 1
	}

	return tokens, nil
}

// CountMessageTokens returns an estimate of the number of tokens in the message
func (c *SimpleTokenCounter) CountMessageTokens(message EnhancedMessage) (int, error) {
	// If token count is already set, return it
	if message.TokenCount > 0 {
		return message.TokenCount, nil
	}

	// Count tokens in the content
	return c.CountTokens(message.Content)
}

// CountSummaryTokens returns an estimate of the number of tokens in the summary
func (c *SimpleTokenCounter) CountSummaryTokens(summary Summary) (int, error) {
	// If token count is already set, return it
	if summary.TokenCount > 0 {
		return summary.TokenCount, nil
	}

	// Count tokens in the content
	return c.CountTokens(summary.Content)
}

// EstimateContextTokens estimates the total tokens that would be used when sending
// these messages to the specified model
func (c *SimpleTokenCounter) EstimateContextTokens(messages []EnhancedMessage, model string) (int, error) {
	totalTokens := 0

	for _, msg := range messages {
		tokens := msg.TokenCount

		// If token count is not set, calculate it
		if tokens == 0 {
			var err error
			tokens, err = c.CountMessageTokens(msg)
			if err != nil {
				return 0, err
			}
		}

		// Add overhead for message formatting (role, etc.)
		// This varies by model but is roughly 4-5 tokens per message
		overhead := 5

		totalTokens += tokens + overhead
	}

	return totalTokens, nil
}

// Helper functions

// countSpecialChars counts the number of special characters in the text
func countSpecialChars(text string) int {
	specialCount := 0
	for _, r := range text {
		if !isAlphaNumeric(r) && !isWhitespace(r) {
			specialCount++
		}
	}
	return specialCount
}

// isAlphaNumeric checks if a rune is alphanumeric
func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// containsCodeBlock checks if the text contains code blocks
func containsCodeBlock(text string) bool {
	return strings.Contains(text, "```") ||
		strings.Contains(text, "    ") || // Four spaces often indicate code
		strings.Contains(text, "\t") // Tabs often indicate code
}

// ClaudeTokenCounter is a more accurate token counter specifically for Claude models
// This would use a proper tokenization library that matches Claude's tokenization
type ClaudeTokenCounter struct {
	// TODO: Implement a proper Claude tokenizer
	// For now, we'll just use the SimpleTokenCounter as a base
	simple *SimpleTokenCounter

	// Model-specific adjustments
	model string
}

// NewClaudeTokenCounter creates a new token counter for Claude models
func NewClaudeTokenCounter(model string) *ClaudeTokenCounter {
	return &ClaudeTokenCounter{
		simple: NewSimpleTokenCounter(),
		model:  model,
	}
}

// CountTokens returns the number of tokens in the text according to Claude's tokenization
func (c *ClaudeTokenCounter) CountTokens(text string) (int, error) {
	// In a real implementation, this would use a proper tokenizer for Claude
	// For now, we'll just use the simple counter with adjustments

	baseCount, err := c.simple.CountTokens(text)
	if err != nil {
		return 0, err
	}

	// Apply model-specific adjustments
	switch c.model {
	case "claude-3-opus":
		// Opus might be slightly more verbose in tokenization
		return int(float64(baseCount) * 1.05), nil
	case "claude-3-sonnet":
		// Use base count as is
		return baseCount, nil
	case "claude-3-haiku":
		// Haiku might be slightly more efficient
		return int(float64(baseCount) * 0.95), nil
	default:
		return baseCount, nil
	}
}

// CountMessageTokens returns the number of tokens in the message
func (c *ClaudeTokenCounter) CountMessageTokens(message EnhancedMessage) (int, error) {
	// If token count is already set, return it
	if message.TokenCount > 0 {
		return message.TokenCount, nil
	}

	// Count tokens in the content
	return c.CountTokens(message.Content)
}

// CountSummaryTokens returns the number of tokens in the summary
func (c *ClaudeTokenCounter) CountSummaryTokens(summary Summary) (int, error) {
	// If token count is already set, return it
	if summary.TokenCount > 0 {
		return summary.TokenCount, nil
	}

	// Count tokens in the content
	return c.CountTokens(summary.Content)
}

// EstimateContextTokens estimates the total tokens that would be used when sending
// these messages to the specified model
func (c *ClaudeTokenCounter) EstimateContextTokens(messages []EnhancedMessage, model string) (int, error) {
	// Set model if provided
	useModel := c.model
	if model != "" {
		useModel = model
	}

	// Create a counter for the specified model
	counter := NewClaudeTokenCounter(useModel)

	totalTokens := 0

	// Add message-level formatting overhead
	// Claude uses a few tokens for message formatting
	messageOverhead := 5

	for _, msg := range messages {
		tokens := msg.TokenCount

		// If token count is not set, calculate it
		if tokens == 0 {
			var err error
			tokens, err = counter.CountMessageTokens(msg)
			if err != nil {
				return 0, fmt.Errorf("failed to count message tokens: %w", err)
			}
		}

		totalTokens += tokens + messageOverhead
	}

	// Add conversation-level overhead
	// This accounts for the overall format of the conversation
	conversationOverhead := 10

	return totalTokens + conversationOverhead, nil
}
