package development

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// DiffInput represents parameters for comparing content
type DiffInput struct {
	Mode         string `json:"mode"`                    // "string", "file", or "mixed"
	Original     string `json:"original,omitempty"`      // Original content or file path (depending on mode)
	Modified     string `json:"modified,omitempty"`      // Modified content or file path (depending on mode)
	Context      int    `json:"context,omitempty"`       // Number of context lines (default: 3)
	OutputFormat string `json:"output_format,omitempty"` // "unified" (default) or "side-by-side"
}

// DiffOutput represents the result of a diff operation
type DiffOutput struct {
	DiffExists bool   `json:"diff_exists"` // Whether differences were found
	DiffOutput string `json:"diff_output"` // Output of the diff operation
}

// DiffTool compares two files or strings and shows their differences
type DiffTool struct {
	core.BaseToolImpl
}

// NewDiffTool creates a new diff tool
func NewDiffTool() *DiffTool {
	tool := &DiffTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"diff",
		"Compare two files or strings and show differences",
		"development",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Mode of comparison: 'string' for string-to-string, 'file' for file-to-file, 'mixed' for string-to-file",
					"enum":        []string{"string", "file", "mixed"},
					"default":     "string",
				},
				"original": map[string]interface{}{
					"type":        "string",
					"description": "Original content (for string mode) or file path (for file mode)",
				},
				"modified": map[string]interface{}{
					"type":        "string",
					"description": "Modified content (for string mode) or file path (for file mode)",
				},
				"context": map[string]interface{}{
					"type":        "integer",
					"description": "Number of context lines in the diff (default: 3)",
					"default":     3,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: 'unified' (default) or 'side-by-side'",
					"enum":        []string{"unified", "side-by-side"},
					"default":     "unified",
				},
			},
			"required": []string{"mode", "original", "modified"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *DiffTool) Execute(input json.RawMessage) (interface{}, error) {
	var params DiffInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for diff tool: %w", err)
	}

	// Set defaults
	if params.Context <= 0 {
		params.Context = 3 // Default context lines
	}

	if params.OutputFormat == "" {
		params.OutputFormat = "unified" // Default output format
	}

	// Validate common parameters
	if params.Original == "" || params.Modified == "" {
		return nil, fmt.Errorf("both original and modified content/paths are required")
	}

	// Choose operation based on mode
	switch params.Mode {
	case "string":
		return t.diffStrings(params)
	case "file":
		return t.diffFiles(params)
	case "mixed":
		return t.diffMixed(params)
	default:
		return nil, fmt.Errorf("invalid mode %q, must be 'string', 'file', or 'mixed'", params.Mode)
	}
}

// diffStrings compares two strings and returns their differences
func (t *DiffTool) diffStrings(params DiffInput) (interface{}, error) {
	// Create temporary files for diff
	origFile, err := os.CreateTemp("", "diff-orig-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(origFile.Name())
	defer origFile.Close()

	modFile, err := os.CreateTemp("", "diff-mod-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(modFile.Name())
	defer modFile.Close()

	// Write content to temp files
	if _, err := origFile.WriteString(params.Original); err != nil {
		return nil, fmt.Errorf("failed to write original content: %w", err)
	}

	if _, err := modFile.WriteString(params.Modified); err != nil {
		return nil, fmt.Errorf("failed to write modified content: %w", err)
	}

	// Close files to ensure content is flushed
	origFile.Close()
	modFile.Close()

	// Run diff command
	return t.runDiff(origFile.Name(), modFile.Name(), "string-a", "string-b", params)
}

// diffFiles compares two files and returns their differences
func (t *DiffTool) diffFiles(params DiffInput) (interface{}, error) {
	// Get absolute paths
	origPath, err := filepath.Abs(params.Original)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for original: %w", err)
	}

	modPath, err := filepath.Abs(params.Modified)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for modified: %w", err)
	}

	// Check if files exist
	if _, err := os.Stat(origPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("original file does not exist: %s", origPath)
	}
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("modified file does not exist: %s", modPath)
	}

	// Run diff command
	return t.runDiff(origPath, modPath, filepath.Base(origPath), filepath.Base(modPath), params)
}

// diffMixed compares a file with a string and returns their differences
func (t *DiffTool) diffMixed(params DiffInput) (interface{}, error) {
	// Get absolute path for file
	filePath, err := filepath.Abs(params.Original)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for file: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Create temporary file for string content
	stringFile, err := os.CreateTemp("", "diff-string-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(stringFile.Name())
	defer stringFile.Close()

	// Write string content to temp file
	if _, err := stringFile.WriteString(params.Modified); err != nil {
		return nil, fmt.Errorf("failed to write string content: %w", err)
	}
	stringFile.Close()

	// Run diff command
	return t.runDiff(filePath, stringFile.Name(), filepath.Base(filePath), "string", params)
}

// runDiff executes the diff command and returns its output
func (t *DiffTool) runDiff(file1, file2, label1, label2 string, params DiffInput) (interface{}, error) {
	// Build command args
	args := []string{}

	if params.OutputFormat == "unified" {
		args = append(args, "-u", fmt.Sprintf("-U%d", params.Context))
	} else if params.OutputFormat == "side-by-side" {
		args = append(args, "-y", "-W", "160")
		// Note: macOS diff doesn't support --left-column option
	}

	args = append(args, "--label", label1, "--label", label2, file1, file2)

	// Run diff command
	cmd := exec.Command("diff", args...)
	output, err := cmd.CombinedOutput()

	// diff returns exit code 1 when files differ, which is what we want
	diffExists := false
	if err != nil {
		if cmd.ProcessState.ExitCode() == 1 {
			// Files differ - this is normal
			diffExists = true
		} else {
			// Other error occurred
			return nil, fmt.Errorf("diff command failed: %w: %s", err, string(output))
		}
	}

	// Return diff output
	return DiffOutput{
		DiffExists: diffExists,
		DiffOutput: string(output),
	}, nil
}
