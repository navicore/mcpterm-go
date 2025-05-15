package context

import (
	"regexp"
	"strings"
)

// PriorityRules defines rules for message prioritization
type PriorityRules struct {
	// Keywords that indicate higher message importance
	HighPriorityKeywords []string

	// Regular expressions for identifying code blocks
	CodePatterns []*regexp.Regexp

	// Regular expressions for identifying important messages
	ImportantPatterns []*regexp.Regexp

	// Regular expressions for identifying critical messages
	CriticalPatterns []*regexp.Regexp

	// Keywords that indicate user preferences
	PreferenceKeywords []string

	// Keywords that indicate requirements
	RequirementKeywords []string
}

// DefaultPriorityRules returns the default priority rules
func DefaultPriorityRules() PriorityRules {
	return PriorityRules{
		HighPriorityKeywords: []string{
			"important", "critical", "must", "need", "should",
			"requirement", "specification", "preference",
		},
		CodePatterns: []*regexp.Regexp{
			regexp.MustCompile("```[a-zA-Z]*\\s*[\\s\\S]*?```"),
			regexp.MustCompile("(?m)^( {4,}|\\t+)[^\\s].*$"),
		},
		ImportantPatterns: []*regexp.Regexp{
			regexp.MustCompile("(?i)important|must|should|requirement"),
			regexp.MustCompile("(?i)need to|needs to|please ensure|don't forget"),
		},
		CriticalPatterns: []*regexp.Regexp{
			regexp.MustCompile("(?i)critical|crucial|essential|priority"),
			regexp.MustCompile("(?i)urgent|immediately|asap|right away"),
		},
		PreferenceKeywords: []string{
			"prefer", "preference", "like", "wish", "want", "desire",
			"better if", "ideally", "should be", "would rather",
		},
		RequirementKeywords: []string{
			"must", "need", "require", "should", "has to", "have to",
			"necessary", "mandatory", "essential", "required", "needed",
		},
	}
}

// MessagePrioritizer handles message prioritization
type MessagePrioritizer struct {
	rules PriorityRules
}

// NewMessagePrioritizer creates a new message prioritizer
func NewMessagePrioritizer(rules PriorityRules) *MessagePrioritizer {
	return &MessagePrioritizer{
		rules: rules,
	}
}

// AssignImportanceLevel assigns an importance level to a message based on content
func (p *MessagePrioritizer) AssignImportanceLevel(message EnhancedMessage) ImportanceLevel {
	// System messages are always high priority
	if message.Type == MessageTypeSystem {
		return ImportanceHigh
	}

	content := message.Content

	// Check for critical patterns
	for _, pattern := range p.rules.CriticalPatterns {
		if pattern.MatchString(content) {
			return ImportanceCritical
		}
	}

	// Check for important patterns
	for _, pattern := range p.rules.ImportantPatterns {
		if pattern.MatchString(content) {
			return ImportanceHigh
		}
	}

	// Check for high priority keywords
	for _, keyword := range p.rules.HighPriorityKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
			return ImportanceHigh
		}
	}

	// Check for code blocks (medium importance)
	for _, pattern := range p.rules.CodePatterns {
		if pattern.MatchString(content) {
			return ImportanceMedium
		}
	}

	// Default to low importance
	return ImportanceLow
}

// ExtractTags extracts tags from a message
func (p *MessagePrioritizer) ExtractTags(message EnhancedMessage) []string {
	var tags []string
	content := message.Content

	// Check for code presence
	for _, pattern := range p.rules.CodePatterns {
		if pattern.MatchString(content) {
			tags = append(tags, "code")
			break
		}
	}

	// Check for preferences
	for _, keyword := range p.rules.PreferenceKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
			tags = append(tags, "preference")
			break
		}
	}

	// Check for requirements
	for _, keyword := range p.rules.RequirementKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
			tags = append(tags, "requirement")
			break
		}
	}

	// Check message type tags
	switch message.Type {
	case MessageTypeUser:
		tags = append(tags, "user_message")
	case MessageTypeAssistant:
		tags = append(tags, "assistant_message")
	case MessageTypeSystem:
		tags = append(tags, "system_message")
	case MessageTypeSummary:
		tags = append(tags, "summary")
	}

	return tags
}

// ExtractTopics attempts to extract topics from a message
// This is a basic implementation that could be enhanced with NLP
func (p *MessagePrioritizer) ExtractTopics(message EnhancedMessage) []string {
	content := message.Content

	// Split into sentences and take the first few words of each as potential topics
	sentences := strings.Split(content, ".")

	topics := make(map[string]bool)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" || len(sentence) < 10 {
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

	// Limit to 3 topics
	if len(result) > 3 {
		result = result[:3]
	}

	return result
}

// EnhanceMessage adds metadata to a message
func (p *MessagePrioritizer) EnhanceMessage(message *EnhancedMessage) {
	// Set importance if not already set
	if message.Importance == 0 {
		message.Importance = p.AssignImportanceLevel(*message)
	}

	// Extract tags if not already set
	if len(message.Tags) == 0 {
		message.Tags = p.ExtractTags(*message)
	}

	// Extract topics if not already set
	if len(message.Topics) == 0 {
		message.Topics = p.ExtractTopics(*message)
	}
}
