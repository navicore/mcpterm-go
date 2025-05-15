package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadTool(t *testing.T) {
	// Create a new file read tool
	fileReadTool := NewFileReadTool()

	// Create a temporary test file
	tempContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	tempFile, err := os.CreateTemp("", "fileread-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up

	_, err = tempFile.WriteString(tempContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Test cases
	testCases := []struct {
		name          string
		input         FileReadInput
		expectError   bool
		expectedLines int
		checkContent  string // Substring that should be in the result
	}{
		{
			name: "Read entire file",
			input: FileReadInput{
				Path: tempFile.Name(),
			},
			expectError:   false,
			expectedLines: 5,
			checkContent:  "line 1",
		},
		{
			name: "Read with offset",
			input: FileReadInput{
				Path:   tempFile.Name(),
				Offset: 2, // Start from line 3
			},
			expectError:   false,
			expectedLines: 3,
			checkContent:  "line 3",
		},
		{
			name: "Read with limit",
			input: FileReadInput{
				Path:  tempFile.Name(),
				Limit: 2,
			},
			expectError:   false,
			expectedLines: 2,
			checkContent:  "line 1",
		},
		{
			name: "Read with offset and limit",
			input: FileReadInput{
				Path:   tempFile.Name(),
				Offset: 1,
				Limit:  2,
			},
			expectError:   false,
			expectedLines: 2,
			checkContent:  "line 2",
		},
		{
			name: "Read nonexistent file",
			input: FileReadInput{
				Path: "/path/does/not/exist.txt",
			},
			expectError: true,
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

			// Execute tool
			result, err := fileReadTool.Execute(jsonInput)

			// Check error
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Skip further checks if we expected an error
			if tc.expectError {
				return
			}

			// Check result type
			content, ok := result.(string)
			if !ok {
				t.Errorf("Expected result of type string, got %T", result)
				return
			}

			// Count lines
			lines := strings.Split(content, "\n")
			// Adjust for trailing newline
			if lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}

			if len(lines) != tc.expectedLines {
				t.Errorf("Expected %d lines, got %d", tc.expectedLines, len(lines))
			}

			// Check content
			if !strings.Contains(content, tc.checkContent) {
				t.Errorf("Expected content to contain '%s', got: %s", tc.checkContent, content)
			}
		})
	}

	// Test reading a real Go file from this package
	t.Run("Read real Go file", func(t *testing.T) {
		// Get the project root directory
		// This approach works because we know our package path structure
		projectRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
		if err != nil {
			t.Fatalf("Failed to get project root: %v", err)
		}

		// Try to read the go.mod file from the project root
		goInput := FileReadInput{
			Path:  filepath.Join(projectRoot, "go.mod"),
			Limit: 5,
		}

		jsonInput, _ := json.Marshal(goInput)
		result, err := fileReadTool.Execute(jsonInput)

		if err != nil {
			t.Logf("Note: Could not read go.mod: %v", err)
			return
		}

		content, ok := result.(string)
		if !ok {
			t.Errorf("Expected result of type string, got %T", result)
			return
		}

		// Verify we got some content
		if len(content) == 0 {
			t.Errorf("Expected non-empty content from go.mod")
		}

		// Just log the first few lines
		lines := strings.Split(content, "\n")
		if len(lines) > 0 {
			t.Logf("go.mod first line: %s", lines[0])
		}
	})
}
