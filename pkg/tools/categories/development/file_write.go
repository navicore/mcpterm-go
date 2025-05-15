package development

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// FileWriteInput represents parameters for writing to a file
type FileWriteInput struct {
	Path    string `json:"path"`             // Path to the file to write
	Content string `json:"content"`          // Content to write to the file
	Append  bool   `json:"append,omitempty"` // Whether to append to the file instead of overwriting
	Mode    string `json:"mode,omitempty"`   // File permissions (e.g., "0644")
	MkDirs  bool   `json:"mkdirs,omitempty"` // Whether to create parent directories if they don't exist
}

// FileWriteOutput represents the result of a file write operation
type FileWriteOutput struct {
	Path         string `json:"path"`          // Path to the file that was written
	BytesWritten int    `json:"bytes_written"` // Number of bytes written
	Created      bool   `json:"created"`       // Whether the file was created (didn't exist before)
}

// FileWriteTool allows writing content to files
type FileWriteTool struct {
	core.BaseToolImpl
}

// NewFileWriteTool creates a new file write tool
func NewFileWriteTool() *FileWriteTool {
	tool := &FileWriteTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"file_write",
		"Write content to a file on macOS",
		"development",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to the file",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to append to the file instead of overwriting (default: false)",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "File permissions (e.g., '0644', default: OS default)",
				},
				"mkdirs": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to create parent directories if they don't exist (default: false)",
				},
			},
			"required": []string{"path", "content"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *FileWriteTool) Execute(input json.RawMessage) (interface{}, error) {
	var params FileWriteInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for file_write tool: %w", err)
	}

	// Validate parameters
	if params.Path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	if params.Content == "" {
		// Allow empty content but log a warning
		fmt.Println("Warning: Writing empty content to file:", params.Path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	fileExists := true
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		fileExists = false
	} else if err != nil {
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	// Create parent directories if needed
	if params.MkDirs && !fileExists {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Determine file mode
	var fileMode os.FileMode = 0644 // Default mode
	if params.Mode != "" {
		var mode uint64
		_, err := fmt.Sscanf(params.Mode, "%o", &mode)
		if err != nil {
			return nil, fmt.Errorf("invalid file mode '%s': %w", params.Mode, err)
		}
		fileMode = os.FileMode(mode)
	}

	// Open file with appropriate flags
	flags := os.O_CREATE | os.O_WRONLY
	if params.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(absPath, flags, fileMode)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer file.Close()

	// Write content to file
	bytesWritten, err := file.WriteString(params.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to write to file: %w", err)
	}

	// Return result
	return FileWriteOutput{
		Path:         absPath,
		BytesWritten: bytesWritten,
		Created:      !fileExists,
	}, nil
}
