package development

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriteTool(t *testing.T) {
	// Create a new file write tool
	fileWriteTool := NewFileWriteTool()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "filewrite-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Test cases
	testCases := []struct {
		name           string
		input          FileWriteInput
		expectError    bool
		verifyContent  string // Expected content to verify after writing
		verifyExists   bool   // Whether the file should exist after operation
		verifyAppended bool   // Whether content should be appended (check for duplicated content)
	}{
		{
			name: "Write new file",
			input: FileWriteInput{
				Path:    filepath.Join(tempDir, "test1.txt"),
				Content: "Hello, World!",
			},
			expectError:   false,
			verifyContent: "Hello, World!",
			verifyExists:  true,
		},
		{
			name: "Overwrite existing file",
			input: FileWriteInput{
				Path:    filepath.Join(tempDir, "test2.txt"),
				Content: "Original content",
			},
			expectError:   false,
			verifyContent: "New content", // Will be overwritten in the second write
			verifyExists:  true,
		},
		{
			name: "Append to file",
			input: FileWriteInput{
				Path:    filepath.Join(tempDir, "test3.txt"),
				Content: "First line\n",
				Append:  false,
			},
			expectError:    false,
			verifyContent:  "First line\nSecond line\n",
			verifyExists:   true,
			verifyAppended: true,
		},
		{
			name: "Create directories if needed",
			input: FileWriteInput{
				Path:    filepath.Join(tempDir, "subdir", "nested", "test4.txt"),
				Content: "Content in nested directory",
				MkDirs:  true,
			},
			expectError:   false,
			verifyContent: "Content in nested directory",
			verifyExists:  true,
		},
		{
			name: "Fail when parent directory doesn't exist",
			input: FileWriteInput{
				Path:    filepath.Join(tempDir, "nonexistent", "test5.txt"),
				Content: "This should fail",
				MkDirs:  false,
			},
			expectError: true,
		},
	}

	// First pass: Create and write files
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal input to JSON
			jsonInput, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// Execute tool
			result, err := fileWriteTool.Execute(jsonInput)

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
			output, ok := result.(FileWriteOutput)
			if !ok {
				t.Errorf("Expected result of type FileWriteOutput, got %T", result)
				return
			}

			// Verify the file exists
			if tc.verifyExists {
				_, err := os.Stat(tc.input.Path)
				if os.IsNotExist(err) {
					t.Errorf("Expected file %s to exist, but it doesn't", tc.input.Path)
				}

				// Read the content to verify
				content, err := os.ReadFile(tc.input.Path)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
				} else {
					// Only verify content here if not testing append
					if !tc.verifyAppended {
						if string(content) != tc.input.Content {
							t.Errorf("Expected content '%s', got '%s'", tc.input.Content, string(content))
						}
					}
				}
			}

			t.Logf("File operation: Path=%s, BytesWritten=%d, Created=%v",
				output.Path, output.BytesWritten, output.Created)
		})
	}

	// Second pass for specific tests that need further operations

	// Test overwriting
	overwritePath := filepath.Join(tempDir, "test2.txt")
	jsonInput, _ := json.Marshal(FileWriteInput{
		Path:    overwritePath,
		Content: "New content",
	})
	fileWriteTool.Execute(jsonInput)

	// Verify overwritten content
	content, _ := os.ReadFile(overwritePath)
	if string(content) != "New content" {
		t.Errorf("Overwrite failed: expected 'New content', got '%s'", string(content))
	}

	// Test appending
	appendPath := filepath.Join(tempDir, "test3.txt")
	jsonInput, _ = json.Marshal(FileWriteInput{
		Path:    appendPath,
		Content: "Second line\n",
		Append:  true,
	})
	fileWriteTool.Execute(jsonInput)

	// Verify appended content
	content, _ = os.ReadFile(appendPath)
	expectedContent := "First line\nSecond line\n"
	if string(content) != expectedContent {
		t.Errorf("Append failed: expected '%s', got '%s'", expectedContent, string(content))
	}
}
