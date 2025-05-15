package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// MkdirInput represents parameters for creating a directory
type MkdirInput struct {
	Path        string `json:"path"`              // Path to create
	MakeParents bool   `json:"parents,omitempty"` // Create parent directories if they don't exist (like mkdir -p)
}

// MkdirOutput represents the result of a directory creation
type MkdirOutput struct {
	Path    string `json:"path"`            // Path that was created
	Created bool   `json:"created"`         // Whether the directory was created
	Error   string `json:"error,omitempty"` // Error message, if any
}

// MkdirTool implements a tool for creating directories
type MkdirTool struct {
	core.BaseToolImpl
}

// NewMkdirTool creates a new mkdir tool
func NewMkdirTool() *MkdirTool {
	tool := &MkdirTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"mkdir",
		"Create new directory at the specified path with optional parent creation (-p)",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path of the directory to create",
				},
				"parents": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories as needed (like mkdir -p)",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *MkdirTool) Execute(input json.RawMessage) (interface{}, error) {
	var params MkdirInput
	if err := json.Unmarshal(input, &params); err != nil {
		return MkdirOutput{
			Created: false,
			Error:   fmt.Sprintf("Invalid input: %v", err),
		}, fmt.Errorf("invalid input for mkdir tool: %w", err)
	}

	// Validate path
	if params.Path == "" {
		return MkdirOutput{
			Created: false,
			Error:   "Path parameter is required",
		}, fmt.Errorf("path parameter is required")
	}

	// Clean the path to make it consistent
	cleanPath := filepath.Clean(params.Path)

	// Check if directory already exists
	if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
		return MkdirOutput{
			Path:    cleanPath,
			Created: false,
			Error:   fmt.Sprintf("Directory %s already exists", cleanPath),
		}, nil
	}

	var err error
	if params.MakeParents {
		// Create directory with parents (mkdir -p)
		err = os.MkdirAll(cleanPath, 0755)
	} else {
		// Create just one directory
		err = os.Mkdir(cleanPath, 0755)
	}

	if err != nil {
		return MkdirOutput{
			Path:    cleanPath,
			Created: false,
			Error:   err.Error(),
		}, fmt.Errorf("failed to create directory %s: %w", cleanPath, err)
	}

	return MkdirOutput{
		Path:    cleanPath,
		Created: true,
	}, nil
}
