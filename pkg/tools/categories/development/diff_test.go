package development

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffTool(t *testing.T) {
	// Create a new diff tool
	diffTool := NewDiffTool()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "diff-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create test files
	file1Path := filepath.Join(tempDir, "file1.txt")
	file2Path := filepath.Join(tempDir, "file2.txt")
	content1 := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	content2 := "line 1\nmodified line 2\nline 3\nnew line\nline 5\n"

	if err := os.WriteFile(file1Path, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.WriteFile(file2Path, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test case: Diff strings
	t.Run("Diff strings", func(t *testing.T) {
		input := DiffInput{
			Mode:     "string",
			Original: content1,
			Modified: content2,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := diffTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to diff strings: %v", err)
		}

		output, ok := result.(DiffOutput)
		if !ok {
			t.Fatalf("Expected DiffOutput, got %T", result)
		}

		if !output.DiffExists {
			t.Errorf("Expected differences to exist")
		}

		// Verify the diff looks correct
		if !strings.Contains(output.DiffOutput, "-line 2") {
			t.Errorf("Expected diff to contain removed line '-line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+modified line 2") {
			t.Errorf("Expected diff to contain added line '+modified line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+new line") {
			t.Errorf("Expected diff to contain added line '+new line'")
		}
	})

	// Test case: Diff files
	t.Run("Diff files", func(t *testing.T) {
		input := DiffInput{
			Mode:     "file",
			Original: file1Path,
			Modified: file2Path,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := diffTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to diff files: %v", err)
		}

		output, ok := result.(DiffOutput)
		if !ok {
			t.Fatalf("Expected DiffOutput, got %T", result)
		}

		if !output.DiffExists {
			t.Errorf("Expected differences to exist")
		}

		// Verify the diff looks correct
		if !strings.Contains(output.DiffOutput, "-line 2") {
			t.Errorf("Expected diff to contain removed line '-line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+modified line 2") {
			t.Errorf("Expected diff to contain added line '+modified line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+new line") {
			t.Errorf("Expected diff to contain added line '+new line'")
		}
	})

	// Test case: Mixed mode (file + string)
	t.Run("Mixed mode diff", func(t *testing.T) {
		input := DiffInput{
			Mode:     "mixed",
			Original: file1Path,
			Modified: content2,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := diffTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to diff in mixed mode: %v", err)
		}

		output, ok := result.(DiffOutput)
		if !ok {
			t.Fatalf("Expected DiffOutput, got %T", result)
		}

		if !output.DiffExists {
			t.Errorf("Expected differences to exist")
		}

		// Verify the diff looks correct
		if !strings.Contains(output.DiffOutput, "-line 2") {
			t.Errorf("Expected diff to contain removed line '-line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+modified line 2") {
			t.Errorf("Expected diff to contain added line '+modified line 2'")
		}
		if !strings.Contains(output.DiffOutput, "+new line") {
			t.Errorf("Expected diff to contain added line '+new line'")
		}
	})

	// Test case: Side-by-side diff
	t.Run("Side-by-side diff", func(t *testing.T) {
		input := DiffInput{
			Mode:         "string",
			Original:     content1,
			Modified:     content2,
			OutputFormat: "side-by-side",
		}

		jsonInput, _ := json.Marshal(input)
		result, err := diffTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to create side-by-side diff: %v", err)
		}

		output, ok := result.(DiffOutput)
		if !ok {
			t.Fatalf("Expected DiffOutput, got %T", result)
		}

		if !output.DiffExists {
			t.Errorf("Expected differences to exist")
		}

		// In side-by-side, we can't easily check for exact patterns as the format is different
		// Just ensure we got some output
		if len(output.DiffOutput) == 0 {
			t.Errorf("Expected non-empty diff output for side-by-side format")
		}
	})

	// Test case: Identical content
	t.Run("Identical content", func(t *testing.T) {
		input := DiffInput{
			Mode:     "string",
			Original: content1,
			Modified: content1,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := diffTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to diff identical content: %v", err)
		}

		output, ok := result.(DiffOutput)
		if !ok {
			t.Fatalf("Expected DiffOutput, got %T", result)
		}

		if output.DiffExists {
			t.Errorf("Expected no differences to exist")
		}

		if len(output.DiffOutput) != 0 {
			t.Errorf("Expected empty diff output for identical content")
		}
	})

	// Test case: Non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		input := DiffInput{
			Mode:     "file",
			Original: filepath.Join(tempDir, "nonexistent.txt"),
			Modified: file2Path,
		}

		jsonInput, _ := json.Marshal(input)
		_, err := diffTool.Execute(jsonInput)

		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}
	})

	// Test case: Invalid mode
	t.Run("Invalid mode", func(t *testing.T) {
		input := DiffInput{
			Mode:     "invalid",
			Original: content1,
			Modified: content2,
		}

		jsonInput, _ := json.Marshal(input)
		_, err := diffTool.Execute(jsonInput)

		if err == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}
