package development

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// PatchInput represents parameters for creating or applying a patch
type PatchInput struct {
	Mode     string `json:"mode"`               // "create" or "apply"
	Path     string `json:"path"`               // Path to the file to patch
	Original string `json:"original,omitempty"` // Original content (for create mode)
	Modified string `json:"modified,omitempty"` // Modified content (for create mode)
	Patch    string `json:"patch,omitempty"`    // Patch content (for apply mode)
	DryRun   bool   `json:"dry_run,omitempty"`  // Whether to do a dry run (don't actually modify files)
	Context  int    `json:"context,omitempty"`  // Number of context lines (for create mode)
}

// PatchOutput represents the result of a patch operation
type PatchOutput struct {
	Success     bool   `json:"success"`      // Whether the operation succeeded
	Mode        string `json:"mode"`         // Mode that was used ("create" or "apply")
	Path        string `json:"path"`         // Path to the affected file
	PatchOutput string `json:"patch_output"` // Output of the patch operation (unified diff or apply results)
	DryRun      bool   `json:"dry_run"`      // Whether this was a dry run
}

// PatchTool allows creating and applying patches to files
type PatchTool struct {
	core.BaseToolImpl
}

// NewPatchTool creates a new patch tool
func NewPatchTool() *PatchTool {
	tool := &PatchTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"patch",
		"Create or apply patches to files on macOS",
		"development",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Mode of operation: 'create' to generate a patch, 'apply' to apply a patch",
					"enum":        []string{"create", "apply"},
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to patch",
				},
				"original": map[string]interface{}{
					"type":        "string",
					"description": "Original content (used in create mode)",
				},
				"modified": map[string]interface{}{
					"type":        "string",
					"description": "Modified content (used in create mode)",
				},
				"patch": map[string]interface{}{
					"type":        "string",
					"description": "Patch content in unified diff format (used in apply mode)",
				},
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to perform a dry run without modifying files (default: false)",
				},
				"context": map[string]interface{}{
					"type":        "integer",
					"description": "Number of context lines in the patch (default: 3)",
				},
			},
			"required": []string{"mode", "path"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *PatchTool) Execute(input json.RawMessage) (interface{}, error) {
	var params PatchInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for patch tool: %w", err)
	}

	// Set defaults
	if params.Context <= 0 {
		params.Context = 3 // Default context lines
	}

	// Validate common parameters
	if params.Path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Get absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Choose operation based on mode
	switch params.Mode {
	case "create":
		return t.createPatch(params, absPath)
	case "apply":
		return t.applyPatch(params, absPath)
	default:
		return nil, fmt.Errorf("invalid mode %q, must be 'create' or 'apply'", params.Mode)
	}
}

// createPatch generates a unified diff between original and modified content
func (t *PatchTool) createPatch(params PatchInput, absPath string) (interface{}, error) {
	// Validate create mode parameters
	if params.Modified == "" {
		return nil, fmt.Errorf("modified content is required in create mode")
	}

	// If original is not provided, try to read the file content as original
	if params.Original == "" {
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file for original content: %w", err)
		}
		params.Original = string(content)
	}

	// Create temporary files for diff
	origFile, err := os.CreateTemp("", "patch-orig-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(origFile.Name())
	defer origFile.Close()

	modFile, err := os.CreateTemp("", "patch-mod-*.txt")
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

	// Run diff command to generate unified diff
	cmd := exec.Command(
		"diff",
		"-u",
		fmt.Sprintf("-U%d", params.Context),
		"--label", filepath.Base(absPath),
		"--label", filepath.Base(absPath),
		origFile.Name(),
		modFile.Name(),
	)

	output, err := cmd.CombinedOutput()
	// diff returns exit code 1 when files differ, which is what we want
	if err != nil && cmd.ProcessState.ExitCode() > 1 {
		return nil, fmt.Errorf("diff command failed: %w", err)
	}

	// Return patch output
	return PatchOutput{
		Success:     true,
		Mode:        "create",
		Path:        absPath,
		PatchOutput: string(output),
		DryRun:      params.DryRun,
	}, nil
}

// applyPatch applies a patch to a file
func (t *PatchTool) applyPatch(params PatchInput, absPath string) (interface{}, error) {
	// Validate apply mode parameters
	if params.Patch == "" {
		return nil, fmt.Errorf("patch content is required in apply mode")
	}

	// Check if file exists
	_, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s does not exist", absPath)
		}
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	// Create temp directory for working files
	tempDir, err := os.MkdirTemp("", "patch-apply")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create patch file
	patchFile := filepath.Join(tempDir, "patch.diff")
	if err := os.WriteFile(patchFile, []byte(params.Patch), 0644); err != nil {
		return nil, fmt.Errorf("failed to write patch file: %w", err)
	}

	// Create a backup of the original file
	backupPath := ""
	if !params.DryRun {
		backupPath = absPath + ".bak." + time.Now().Format("20060102_150405")
		if err := copyFile(absPath, backupPath); err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Build patch command
	args := []string{"-p0", "-u"}
	if params.DryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, "-i", patchFile)

	// Run patch command
	cmd := exec.Command("patch", args...)
	cmd.Dir = filepath.Dir(absPath) // Run in the directory of the target file
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If patch failed and we made a backup, restore it
		if !params.DryRun && backupPath != "" {
			_ = copyFile(backupPath, absPath) // Try to restore, but don't throw error if this fails
		}
		return nil, fmt.Errorf("patch command failed: %s: %w", string(output), err)
	}

	// If dry run is successful, report it
	if params.DryRun {
		return PatchOutput{
			Success:     true,
			Mode:        "apply",
			Path:        absPath,
			PatchOutput: string(output),
			DryRun:      true,
		}, nil
	}

	// Clean up backup if everything was successful and return result
	if backupPath != "" {
		os.Remove(backupPath)
	}

	return PatchOutput{
		Success:     true,
		Mode:        "apply",
		Path:        absPath,
		PatchOutput: string(output),
		DryRun:      false,
	}, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
