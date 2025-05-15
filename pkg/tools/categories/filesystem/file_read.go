package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// FileReadInput represents parameters for reading a file
type FileReadInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// FileReadTool allows reading file contents
type FileReadTool struct {
	core.BaseToolImpl
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool() *FileReadTool {
	tool := &FileReadTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"file_read",
		"Read the contents of a file",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (0-based)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read",
				},
			},
			"required": []string{"path"},
		},
	)
	return tool
}

// Execute reads a file based on the provided input
func (t *FileReadTool) Execute(input json.RawMessage) (interface{}, error) {
	var params FileReadInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for file_read tool: %w", err)
	}

	// Validate path
	if params.Path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Expand ~ to home directory if present
	if strings.HasPrefix(params.Path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to expand home directory: %w", err)
		}
		params.Path = strings.Replace(params.Path, "~", homeDir, 1)
	}

	// Read file
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string and split into lines
	lines := strings.Split(string(content), "\n")

	// Apply offset and limit if provided
	if params.Offset > 0 || params.Limit > 0 {
		if params.Offset >= len(lines) {
			return "", nil
		}

		end := len(lines)
		if params.Limit > 0 && params.Offset+params.Limit < end {
			end = params.Offset + params.Limit
		}

		lines = lines[params.Offset:end]
	}

	// Join lines back together
	return strings.Join(lines, "\n"), nil
}
