package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryListTool(t *testing.T) {
	// Create a new directory list tool
	dirListTool := NewDirectoryListTool()

	// Create a temporary directory with some test files
	tempDir, err := os.MkdirTemp("", "dirlist-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create some test files
	files := []struct {
		name    string
		content string
	}{
		{"test1.txt", "test content 1"},
		{"test2.txt", "test content 2"},
		{"test.go", "package main\n\nfunc main() {}"},
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(path, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Test cases
	testCases := []struct {
		name         string
		input        DirectoryListInput
		expectError  bool
		minEntries   int
		checkEntries map[string]bool // Map of entries that should be present
	}{
		{
			name: "List temp directory",
			input: DirectoryListInput{
				Path: tempDir,
			},
			expectError: false,
			minEntries:  4, // 3 files + 1 subdirectory
			checkEntries: map[string]bool{
				"test1.txt": false, // Not a directory
				"subdir":    true,  // Is a directory
			},
		},
		{
			name: "List with pattern - only .txt files",
			input: DirectoryListInput{
				Path:    tempDir,
				Pattern: "*.txt",
			},
			expectError: false,
			minEntries:  2,
			checkEntries: map[string]bool{
				"test1.txt": false,
				"test2.txt": false,
			},
		},
		{
			name: "List with pattern - only .go files",
			input: DirectoryListInput{
				Path:    tempDir,
				Pattern: "*.go",
			},
			expectError: false,
			minEntries:  1,
			checkEntries: map[string]bool{
				"test.go": false,
			},
		},
		{
			name: "List nonexistent directory",
			input: DirectoryListInput{
				Path: "/path/does/not/exist",
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
			result, err := dirListTool.Execute(jsonInput)

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
			entries, ok := result.([]FileEntry)
			if !ok {
				t.Errorf("Expected result of type []FileEntry, got %T", result)
				return
			}

			// Check minimum entries
			if len(entries) < tc.minEntries {
				t.Errorf("Expected at least %d entries, got %d", tc.minEntries, len(entries))
			}

			// Check specific entries
			entryMap := make(map[string]FileEntry)
			for _, entry := range entries {
				entryMap[entry.Name] = entry
			}

			for name, isDir := range tc.checkEntries {
				entry, exists := entryMap[name]
				if !exists {
					t.Errorf("Expected entry %s not found", name)
					continue
				}

				if entry.IsDir != isDir {
					t.Errorf("Entry %s: expected IsDir=%v, got %v", name, isDir, entry.IsDir)
				}
			}

			// Output entries count for information (not a test assertion)
			t.Logf("Found %d entries", len(entries))
		})
	}
}
