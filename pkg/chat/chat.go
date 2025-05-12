package chat

import (
	"strings"
)

// Message represents a chat message
type Message struct {
	Sender  string
	Content string
	IsUser  bool
}

// ChatService defines the interface for chat functionality
type ChatService interface {
	SendMessage(content string) (Message, error)
	GetHistory() []Message
}

// SimpleChatService is a basic implementation of ChatService
type SimpleChatService struct {
	history []Message
}

// NewSimpleChatService creates a new chat service instance
func NewSimpleChatService() *SimpleChatService {
	return &SimpleChatService{
		history: []Message{},
	}
}

// SendMessage sends a message and returns a response
func (s *SimpleChatService) SendMessage(content string) (Message, error) {
	// Add user message to history
	userMsg := Message{
		Sender:  "You",
		Content: content,
		IsUser:  true,
	}
	s.history = append(s.history, userMsg)

	// In a real application, this would make an API call to a chat service
	// For this demo, we'll return different responses based on the input
	var response string

	// Simple logic to simulate a chat
	lowerContent := strings.ToLower(content)

	switch {
	case strings.Contains(lowerContent, "hello") || strings.Contains(lowerContent, "hi"):
		response = "## Hello there!\n\nHow can I help you today?"

	case strings.Contains(lowerContent, "how are you"):
		response = "I'm just a program, but I'm working well. Thanks for asking! How are you?"

	case strings.Contains(lowerContent, "help"):
		response = "# Help Menu\n\n" +
			"I can help with various things:\n\n" +
			"- Type `features` to see available features\n" +
			"- Type `vi` or `vim` for navigation help\n" +
			"- Type `markdown` for formatting examples\n" +
			"- Type `bye` to end the conversation"

	case strings.Contains(lowerContent, "bye") || strings.Contains(lowerContent, "goodbye"):
		response = "👋 **Goodbye!** Have a great day!"

	case strings.Contains(lowerContent, "feature") || strings.Contains(lowerContent, "function"):
		response = "## Key Features\n\n" +
			"- **Vi-like Motion**: Navigate with vi keybindings\n" +
			"- **Markdown Support**: Format messages with markdown\n" +
			"- **TUI Interface**: Clean terminal user interface\n" +
			"- **Command Mode**: Toggle between insert and normal modes"

	case strings.Contains(lowerContent, "vi") || strings.Contains(lowerContent, "vim"):
		response = "## Vi Navigation\n\n" +
			"Press `Esc` to toggle vi mode. In vi-mode:\n\n" +
			"| Key | Action |\n" +
			"|-----|--------|\n" +
			"| `j` | Move down one line |\n" +
			"| `k` | Move up one line |\n" +
			"| `g` | Go to top |\n" +
			"| `G` | Go to bottom |\n" +
			"| `d` | Scroll half-page down |\n" +
			"| `u` | Scroll half-page up |\n" +
			"| `i` | Enter insert mode |"

	case strings.Contains(lowerContent, "markdown"):
		response = "# Markdown Examples\n\n" +
			"You can use markdown in your messages:\n\n" +
			"## Headers\n" +
			"# Header 1\n" +
			"## Header 2\n\n" +
			"## Text Formatting\n" +
			"*italic text*\n" +
			"**bold text**\n" +
			"`code text`\n\n" +
			"## Lists\n" +
			"- Item 1\n" +
			"- Item 2\n" +
			"  - Nested item\n\n" +
			"1. Numbered item 1\n" +
			"2. Numbered item 2\n\n" +
			"## Blockquotes\n" +
			"> This is a blockquote\n\n" +
			"## Code Blocks\n" +
			"```go\n" +
			"func main() {\n" +
			"    fmt.Println(\"Hello World\")\n" +
			"}\n" +
			"```"

	default:
		response = "I understand you said: \"" + content + "\".\n\n" +
			"How can I help you with that? Type `help` for a list of commands."
	}

	botMsg := Message{
		Sender:  "Assistant",
		Content: response,
		IsUser:  false,
	}
	s.history = append(s.history, botMsg)

	return botMsg, nil
}

// GetHistory returns the chat history
func (s *SimpleChatService) GetHistory() []Message {
	return s.history
}
