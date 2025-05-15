package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// FileDeleteInput represents parameters for deleting a file or directory
type FileDeleteInput struct {
	Path string `json:"path"` // Path of the file or directory to delete
}

// FileDeleteOutput represents the result of a file deletion operation
type FileDeleteOutput struct {
	Path         string `json:"path"`            // Path that was processed
	Deleted      bool   `json:"deleted"`         // Whether the file/directory was deleted
	MovedToTrash bool   `json:"moved_to_trash"`  // Whether the file was moved to trash (macOS)
	Error        string `json:"error,omitempty"` // Error message, if any
}

// FileDeleteTool implements a tool for deleting files by moving them to trash on macOS
type FileDeleteTool struct {
	core.BaseToolImpl
}

// NewFileDeleteTool creates a new file delete tool
func NewFileDeleteTool() *FileDeleteTool {
	tool := &FileDeleteTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"file_delete",
		"Move a file or directory to the trash (on macOS) or delete it",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path of the file or directory to delete/move to trash",
				},
			},
			"required": []string{"path"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *FileDeleteTool) Execute(input json.RawMessage) (interface{}, error) {
	var params FileDeleteInput
	if err := json.Unmarshal(input, &params); err != nil {
		return FileDeleteOutput{
			Deleted:      false,
			MovedToTrash: false,
			Error:        fmt.Sprintf("Invalid input: %v", err),
		}, fmt.Errorf("invalid input for file_delete tool: %w", err)
	}

	// Validate path
	if params.Path == "" {
		return FileDeleteOutput{
			Deleted:      false,
			MovedToTrash: false,
			Error:        "Path parameter is required",
		}, fmt.Errorf("path parameter is required")
	}

	// Clean the path to make it consistent
	cleanPath := filepath.Clean(params.Path)

	// Check if file/directory exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return FileDeleteOutput{
			Path:         cleanPath,
			Deleted:      false,
			MovedToTrash: false,
			Error:        fmt.Sprintf("Path %s does not exist", cleanPath),
		}, nil
	}

	// Try to use platform-specific trash functionality
	if runtime.GOOS == "darwin" {
		// On macOS, we can use the built-in "move to trash" AppleScript functionality
		return moveToMacOSTrash(cleanPath)
	}

	// On other platforms, just delete the file
	if err := os.RemoveAll(cleanPath); err != nil {
		return FileDeleteOutput{
			Path:         cleanPath,
			Deleted:      false,
			MovedToTrash: false,
			Error:        err.Error(),
		}, fmt.Errorf("failed to delete %s: %w", cleanPath, err)
	}

	return FileDeleteOutput{
		Path:         cleanPath,
		Deleted:      true,
		MovedToTrash: false,
	}, nil
}

// moveToMacOSTrash moves a file/directory to the macOS Trash using osascript
func moveToMacOSTrash(path string) (interface{}, error) {
	// Check if path exists first
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return FileDeleteOutput{
			Path:         path,
			Deleted:      false,
			MovedToTrash: false,
			Error:        fmt.Sprintf("Path %s does not exist", path),
		}, nil
	}

	// Get absolute path for AppleScript
	absPath, err := filepath.Abs(path)
	if err != nil {
		return FileDeleteOutput{
			Path:         path,
			Deleted:      false,
			MovedToTrash: false,
			Error:        fmt.Sprintf("Failed to get absolute path: %v", err),
		}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// AppleScript to move file to trash
	script := fmt.Sprintf(`
	tell application "Finder"
		set itemToDelete to POSIX file "%s" as alias
		move itemToDelete to trash
	end tell
	`, absPath)

	// Execute the AppleScript
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		// If AppleScript fails, fall back to regular delete
		if err := os.RemoveAll(path); err != nil {
			return FileDeleteOutput{
				Path:         path,
				Deleted:      false,
				MovedToTrash: false,
				Error:        fmt.Sprintf("Failed to delete or move to trash: %v", err),
			}, fmt.Errorf("failed to delete or move to trash: %w", err)
		}

		return FileDeleteOutput{
			Path:         path,
			Deleted:      true,
			MovedToTrash: false,
			Error:        "AppleScript failed, file was deleted permanently",
		}, nil
	}

	return FileDeleteOutput{
		Path:         path,
		Deleted:      true,
		MovedToTrash: true,
	}, nil
}
