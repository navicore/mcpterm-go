package filesystem

import (
	"encoding/json"
	"testing"
)

func TestFindTool(t *testing.T) {
	// Create a new find tool
	findTool := NewFindTool()

	// Test cases
	testCases := []struct {
		name        string
		input       FindInput
		expectError bool
		minResults  int // Minimum number of results expected
	}{
		{
			name: "Basic find with type and name",
			input: FindInput{
				Directory: ".",
				Type:      "f",
				Name:      "*.go",
				Maxdepth:  2,
			},
			expectError: false,
			minResults:  1, // Should find at least this test file
		},
		{
			name: "Find with mtime",
			input: FindInput{
				Directory: ".",
				Type:      "f",
				Name:      "*.go",
				Mtime:     "-30", // Files modified in the last month
				Maxdepth:  2,
			},
			expectError: false,
			minResults:  1,
		},
		{
			name: "Find with size",
			input: FindInput{
				Directory: ".",
				Type:      "f",
				Name:      "*.go",
				Size:      "+1", // Files larger than 1 byte
				Maxdepth:  2,
			},
			expectError: false,
			minResults:  1,
		},
		{
			name: "Find with path exclusion",
			input: FindInput{
				Directory: ".",
				Type:      "f",
				Path:      "!*/\\.git/*",
				Maxdepth:  3,
			},
			expectError: false,
			minResults:  1,
		},
		{
			name: "Complex find with multiple options",
			input: FindInput{
				Directory: ".",
				Type:      "f",
				Name:      "*.go",
				Size:      "+1",
				Mtime:     "-30",
				Maxdepth:  3,
			},
			expectError: false,
			minResults:  1,
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
			result, err := findTool.Execute(jsonInput)

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
			files, ok := result.([]string)
			if !ok {
				t.Errorf("Expected result of type []string, got %T", result)
				return
			}

			// Check minimum number of results
			if len(files) < tc.minResults {
				t.Errorf("Expected at least %d results, got %d", tc.minResults, len(files))
			}

			// Output results count for information (not a test assertion)
			t.Logf("Found %d files", len(files))
		})
	}
}
