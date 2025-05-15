package chat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultSystemPrompt(t *testing.T) {
	// Get the default system prompt
	prompt := GetDefaultSystemPrompt()

	// Verify it's not empty and matches our constant
	assert.NotEmpty(t, prompt)
	assert.Equal(t, DefaultSystemPrompt, prompt)

	// Verify it contains expected keywords
	assert.True(t, strings.Contains(prompt, "helpful AI assistant"), "Prompt should describe the assistant role")
	assert.True(t, strings.Contains(prompt, "terminal"), "Prompt should mention the terminal environment")
	assert.True(t, strings.Contains(prompt, "Markdown"), "Prompt should mention Markdown formatting")

	// Verify it includes capabilities
	assert.True(t, strings.Contains(prompt, "Reading files"), "Prompt should mention file reading capability")
	assert.True(t, strings.Contains(prompt, "shell commands"), "Prompt should mention shell command capability")
}
