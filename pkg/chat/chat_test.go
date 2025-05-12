package chat

import (
	"strings"
	"testing"
)

func TestChatService(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedExists string
	}{
		{
			name:           "Hello command",
			input:          "Hello there",
			expectedExists: "Hello there!",
		},
		{
			name:           "Help command",
			input:          "help me please",
			expectedExists: "Help Menu",
		},
		{
			name:           "Feature command",
			input:          "what features do you have?",
			expectedExists: "Key Features",
		},
		{
			name:           "Vi command",
			input:          "How do I use VI mode?",
			expectedExists: "Vi Navigation",
		},
		{
			name:           "Markdown command",
			input:          "Show me markdown examples",
			expectedExists: "Markdown Examples",
		},
		{
			name:           "Default response",
			input:          "xyz123", // Something unlikely to match other patterns
			expectedExists: "I understand you said:",
		},
	}

	chatService := NewSimpleChatService()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Send message to chat service
			response, err := chatService.SendMessage(tc.input)
			if err != nil {
				t.Fatalf("Error sending message: %v", err)
			}

			// Uncomment for debugging
			// t.Logf("Response for '%s': %s", tc.input, response.Content)

			// Check if response contains expected text
			if !strings.Contains(response.Content, tc.expectedExists) {
				t.Errorf("Expected response to contain '%s', but got: %s",
					tc.expectedExists, response.Content)
			}

			// Verify the message was added to history
			history := chatService.GetHistory()
			if len(history) < 2 {
				t.Fatalf("Expected at least 2 messages in history (user + response), got %d", len(history))
			}

			// Verify last message is the response we just got
			lastMsg := history[len(history)-1]
			if lastMsg.Content != response.Content {
				t.Errorf("Last message in history doesn't match response")
			}
		})
	}
}
