package development

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestShellTool(t *testing.T) {
	// Create a new shell tool
	shellTool := NewShellTool()

	// Create a temporary directory for working directory tests
	tempDir, err := os.MkdirTemp("", "shell-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Test cases
	testCases := []struct {
		name               string
		input              ShellInput
		expectError        bool
		expectedExitCode   int
		checkStdoutContent string // Substring that should be in stdout
	}{
		{
			name: "Simple echo command",
			input: ShellInput{
				Command: "echo",
				Args:    []string{"Hello, Shell Tool!"},
			},
			expectError:        false,
			expectedExitCode:   0,
			checkStdoutContent: "Hello, Shell Tool!",
		},
		{
			name: "Command with working directory",
			input: ShellInput{
				Command:    "pwd",
				WorkingDir: tempDir,
			},
			expectError:        false,
			expectedExitCode:   0,
			checkStdoutContent: tempDir,
		},
		{
			name: "Command with timeout (should fail)",
			input: ShellInput{
				Command:     "sleep",
				Args:        []string{"3"},
				TimeoutSecs: 1,
			},
			expectError:      false, // The tool handles timeouts as part of its output
			expectedExitCode: -1,    // The exit code for timeouts
		},
		{
			name: "Nonexistent command",
			input: ShellInput{
				Command: "command_that_does_not_exist_12345",
			},
			expectError:      false, // The tool handles command errors in its output
			expectedExitCode: -1,
		},
		{
			name: "List files in working directory",
			input: ShellInput{
				Command:    "ls",
				Args:       []string{"-la"},
				WorkingDir: tempDir,
			},
			expectError:      false,
			expectedExitCode: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal input to JSON
			jsonInput, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			startTime := time.Now()

			// Execute tool
			result, err := shellTool.Execute(jsonInput)

			duration := time.Since(startTime)

			// Check error from Execute (this should be rare, as most errors are returned in the result)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Skip further checks if Execute returned an error
			if err != nil {
				return
			}

			// Check result type
			output, ok := result.(ShellOutput)
			if !ok {
				t.Errorf("Expected result of type ShellOutput, got %T", result)
				return
			}

			// Check exit code
			if output.ExitCode != tc.expectedExitCode {
				t.Errorf("Expected exit code %d, got %d", tc.expectedExitCode, output.ExitCode)
			}

			// Log all output for debugging
			t.Logf("Duration: %v", duration)
			t.Logf("Exit code: %d", output.ExitCode)
			t.Logf("Stdout: %s", output.Stdout)
			t.Logf("Stderr: %s", output.Stderr)
			if output.Error != "" {
				t.Logf("Error: %s", output.Error)
			}

			// Check stdout content if specified
			if tc.checkStdoutContent != "" && !strings.Contains(output.Stdout, tc.checkStdoutContent) {
				t.Errorf("Expected stdout to contain '%s', got: %s", tc.checkStdoutContent, output.Stdout)
			}

			// For timeout test, specifically check the error message
			if tc.name == "Command with timeout (should fail)" {
				if !strings.Contains(output.Error, "timed out") {
					t.Errorf("Expected timeout error message, got: %s", output.Error)
				}
			}
		})
	}
}
