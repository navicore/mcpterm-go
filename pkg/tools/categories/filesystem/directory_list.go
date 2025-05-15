package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// DirectoryListInput represents parameters for listing a directory
type DirectoryListInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

// DirectoryListTool is a placeholder implementation of the directory listing tool
type DirectoryListTool struct {
	core.BaseToolImpl
}

// FileEntry represents a file in a directory
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// NewDirectoryListTool creates a new directory list tool
func NewDirectoryListTool() *DirectoryListTool {
	tool := &DirectoryListTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"directory_list",
		"List files and directories in a specified path",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the directory to list",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Pattern to filter files (e.g., '*.go')",
				},
			},
			"required": []string{"path"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *DirectoryListTool) Execute(input json.RawMessage) (interface{}, error) {
	var params DirectoryListInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for directory_list tool: %w", err)
	}

	// Validate path
	if params.Path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Check if directory exists
	info, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", params.Path)
	}

	// Read directory
	entries, err := os.ReadDir(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter by pattern if provided
	var filtered []os.DirEntry
	if params.Pattern != "" {
		for _, entry := range entries {
			matched, err := filepath.Match(params.Pattern, entry.Name())
			if err != nil {
				return nil, fmt.Errorf("invalid pattern: %w", err)
			}
			if matched {
				filtered = append(filtered, entry)
			}
		}
	} else {
		filtered = entries
	}

	// Convert to our output format
	result := make([]FileEntry, 0, len(filtered))
	for _, entry := range filtered {
		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't get info for
			continue
		}

		fileEntry := FileEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(params.Path, entry.Name()),
			IsDir: entry.IsDir(),
		}

		if !entry.IsDir() {
			fileEntry.Size = info.Size()
		}

		result = append(result, fileEntry)
	}

	return result, nil
}
