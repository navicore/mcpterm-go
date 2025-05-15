package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// FileRenameInput represents parameters for renaming a file or directory
type FileRenameInput struct {
	OldPath string `json:"old_path"` // Current path
	NewPath string `json:"new_path"` // New path
}

// FileRenameOutput represents the result of a file or directory rename operation
type FileRenameOutput struct {
	OldPath     string `json:"old_path"`        // Original path
	NewPath     string `json:"new_path"`        // New path
	Renamed     bool   `json:"renamed"`         // Whether the rename was successful
	IsDirectory bool   `json:"is_directory"`    // Whether this was a directory operation
	Error       string `json:"error,omitempty"` // Error message, if any
}

// FileRenameTool implements a tool for renaming files and directories
type FileRenameTool struct {
	core.BaseToolImpl
}

// NewFileRenameTool creates a new file rename tool
func NewFileRenameTool() *FileRenameTool {
	tool := &FileRenameTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"file_rename",
		"Rename a file or directory to a new name, or move it to a new location (works like mv)",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"old_path": map[string]interface{}{
					"type":        "string",
					"description": "Current path of the file or directory to rename or move",
				},
				"new_path": map[string]interface{}{
					"type":        "string",
					"description": "New path for the file or directory (can be in a different directory to move it)",
				},
			},
			"required": []string{"old_path", "new_path"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *FileRenameTool) Execute(input json.RawMessage) (interface{}, error) {
	var params FileRenameInput
	if err := json.Unmarshal(input, &params); err != nil {
		return FileRenameOutput{
			Renamed: false,
			Error:   fmt.Sprintf("Invalid input: %v", err),
		}, fmt.Errorf("invalid input for file_rename tool: %w", err)
	}

	// Validate paths
	if params.OldPath == "" {
		return FileRenameOutput{
			Renamed: false,
			Error:   "Old path parameter is required",
		}, fmt.Errorf("old path parameter is required")
	}

	if params.NewPath == "" {
		return FileRenameOutput{
			Renamed: false,
			Error:   "New path parameter is required",
		}, fmt.Errorf("new path parameter is required")
	}

	// Clean paths to make them consistent
	oldPath := filepath.Clean(params.OldPath)
	newPath := filepath.Clean(params.NewPath)

	// Check if old path exists
	fileInfo, err := os.Stat(oldPath)
	if os.IsNotExist(err) {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: false,
			Error:       fmt.Sprintf("Path %s does not exist", oldPath),
		}, fmt.Errorf("path %s does not exist", oldPath)
	} else if err != nil {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: false,
			Error:       fmt.Sprintf("Failed to access path: %v", err),
		}, fmt.Errorf("failed to access path: %w", err)
	}

	// Check if the target parent directory exists
	newParentDir := filepath.Dir(newPath)
	if _, err := os.Stat(newParentDir); os.IsNotExist(err) {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: fileInfo.IsDir(),
			Error:       fmt.Sprintf("Parent directory for new path does not exist: %s", newParentDir),
		}, fmt.Errorf("parent directory for new path does not exist: %s", newParentDir)
	} else if err != nil {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: fileInfo.IsDir(),
			Error:       fmt.Sprintf("Failed to access parent directory: %v", err),
		}, fmt.Errorf("failed to access parent directory: %w", err)
	}

	// Check if new path already exists
	if _, err := os.Stat(newPath); err == nil {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: fileInfo.IsDir(),
			Error:       fmt.Sprintf("Destination path already exists: %s", newPath),
		}, fmt.Errorf("destination path already exists: %s", newPath)
	}

	// Perform the rename operation
	if err := os.Rename(oldPath, newPath); err != nil {
		return FileRenameOutput{
			OldPath:     oldPath,
			NewPath:     newPath,
			Renamed:     false,
			IsDirectory: fileInfo.IsDir(),
			Error:       err.Error(),
		}, fmt.Errorf("failed to rename %s to %s: %w", oldPath, newPath, err)
	}

	return FileRenameOutput{
		OldPath:     oldPath,
		NewPath:     newPath,
		Renamed:     true,
		IsDirectory: fileInfo.IsDir(),
	}, nil
}
