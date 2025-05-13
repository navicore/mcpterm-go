package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// FindInput represents parameters for the find command
type FindInput struct {
	Directory string   `json:"directory"`
	Name      string   `json:"name,omitempty"`
	Type      string   `json:"type,omitempty"`
	Maxdepth  int      `json:"maxdepth,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Args      []string `json:"args,omitempty"`
}

// FileReadInput represents parameters for reading a file
type FileReadInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// DirectoryListInput represents parameters for listing a directory
type DirectoryListInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

// FileEntry represents a file in a directory
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// ToolExecutor handles execution of tools
type ToolExecutor struct {}

// NewToolExecutor creates a new ToolExecutor
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{}
}

// GetToolDefinitions returns definitions for all available tools
func (t *ToolExecutor) GetToolDefinitions() []backend.ClaudeTool {
	return []backend.ClaudeTool{
		{
			// The Type field should be omitted for custom tools on Bedrock
			Name:        "find",
			Description: "Execute the find command to search for files and directories",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "The directory to search in",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "File name pattern to search for (e.g., '*.go')",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Type of file to find ('f' for regular files, 'd' for directories)",
						"enum":        []string{"f", "d"},
					},
					"maxdepth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum depth to search",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Pattern to match file contents (uses grep)",
					},
					"args": map[string]interface{}{
						"type":        "array",
						"description": "Additional arguments to pass to find",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"directory"},
			},
		},
		{
			// The Type field should be omitted for custom tools on Bedrock
			Name:        "file_read",
			Description: "Read the contents of a file",
			InputSchema: map[string]interface{}{
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
		},
		{
			// The Type field should be omitted for custom tools on Bedrock
			Name:        "directory_list",
			Description: "List files and directories in a specified path",
			InputSchema: map[string]interface{}{
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
		},
	}
}

// ExecuteTool executes a specific tool with given input
func (t *ToolExecutor) ExecuteTool(toolName string, input json.RawMessage) (interface{}, error) {
	switch toolName {
	case "find":
		return t.executeFind(input)
	case "file_read":
		return t.executeFileRead(input)
	case "directory_list":
		return t.executeDirectoryList(input)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// executeFind executes the find command
func (t *ToolExecutor) executeFind(input json.RawMessage) (interface{}, error) {
	var params FindInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for find tool: %w", err)
	}

	// Validate directory
	if params.Directory == "" {
		return nil, fmt.Errorf("directory parameter is required")
	}

	// Expand ~ to home directory if present
	if strings.HasPrefix(params.Directory, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to expand home directory: %w", err)
		}
		params.Directory = strings.Replace(params.Directory, "~", homeDir, 1)
	}

	// Build find command
	args := []string{params.Directory}

	if params.Maxdepth > 0 {
		args = append(args, "-maxdepth", fmt.Sprintf("%d", params.Maxdepth))
	}

	if params.Type != "" {
		args = append(args, "-type", params.Type)
	}

	if params.Name != "" {
		args = append(args, "-name", params.Name)
	}

	// Add any additional args
	if len(params.Args) > 0 {
		args = append(args, params.Args...)
	}

	// Run find command
	cmd := exec.Command("find", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("find command failed: %w", err)
	}

	// Process output
	results := strings.TrimSpace(string(output))
	if results == "" {
		return []string{}, nil
	}

	// If a pattern is provided, filter with grep
	if params.Pattern != "" {
		// Create a pipe command with grep
		grepCmd := exec.Command("grep", "-l", params.Pattern)
		grepCmd.Stdin = strings.NewReader(results)
		
		grepOut, err := grepCmd.Output()
		if err != nil && err.(*exec.ExitError).ExitCode() != 1 { // grep returns 1 when no matches
			return nil, fmt.Errorf("grep command failed: %w", err)
		}
		
		results = strings.TrimSpace(string(grepOut))
		if results == "" {
			return []string{}, nil
		}
	}

	// Split by newlines and return as array
	return strings.Split(results, "\n"), nil
}

// executeFileRead reads a file
func (t *ToolExecutor) executeFileRead(input json.RawMessage) (interface{}, error) {
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

// executeDirectoryList lists directory contents
func (t *ToolExecutor) executeDirectoryList(input json.RawMessage) (interface{}, error) {
	var params DirectoryListInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for directory_list tool: %w", err)
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