package development

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// ShellTool allows executing shell commands
type ShellTool struct {
	core.BaseToolImpl
}

// ShellInput represents parameters for running a shell command
type ShellInput struct {
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
	WorkingDir  string   `json:"working_dir,omitempty"`
	TimeoutSecs int      `json:"timeout_secs,omitempty"`
}

// ShellOutput represents the result of a shell command
type ShellOutput struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

// NewShellTool creates a new shell tool
func NewShellTool() *ShellTool {
	tool := &ShellTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"shell",
		"Execute a shell command on macOS",
		"development",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"args": map[string]interface{}{
					"type":        "array",
					"description": "Arguments to pass to the command",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command execution",
				},
				"timeout_secs": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum execution time in seconds (default 10)",
				},
			},
			"required": []string{"command"},
		},
	)
	return tool
}

// Execute runs a shell command based on the provided input
func (t *ShellTool) Execute(input json.RawMessage) (interface{}, error) {
	var params ShellInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for shell tool: %w", err)
	}

	// Validate command
	if params.Command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}

	// Set default timeout
	timeoutSecs := 10
	if params.TimeoutSecs > 0 {
		timeoutSecs = params.TimeoutSecs
	}
	if timeoutSecs > 60 {
		timeoutSecs = 60 // Maximum timeout of 60 seconds
	}

	// Create command
	cmd := exec.Command(params.Command, params.Args...)

	// Set working directory if specified
	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	// Set up buffers for stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create a channel to signal command completion
	done := make(chan error, 1)

	// Start the command
	if err := cmd.Start(); err != nil {
		return ShellOutput{
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to start command: %v", err),
		}, nil
	}

	// Run command with timeout
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for command to finish or timeout
	var result ShellOutput
	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ExitCode = -1
				result.Error = fmt.Sprintf("command failed: %v", err)
			}
		} else {
			result.ExitCode = 0
		}
	case <-time.After(time.Duration(timeoutSecs) * time.Second):
		// Kill the process on timeout
		if err := cmd.Process.Kill(); err != nil {
			result.Error = fmt.Sprintf("command timed out after %d seconds and failed to kill: %v", timeoutSecs, err)
		} else {
			result.Error = fmt.Sprintf("command timed out after %d seconds", timeoutSecs)
		}
		result.ExitCode = -1
	}

	// Get command output
	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())

	return result, nil
}
