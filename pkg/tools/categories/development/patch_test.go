package development

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchTool(t *testing.T) {
	// Create a new patch tool
	patchTool := NewPatchTool()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "patch-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test.txt")
	originalContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	modifiedContent := "line 1\nmodified line 2\nline 3\nnew line\nline 4\nline 5\n"

	if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test case: Create patch
	t.Run("Create patch", func(t *testing.T) {
		input := PatchInput{
			Mode:     "create",
			Path:     testFilePath,
			Original: originalContent,
			Modified: modifiedContent,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := patchTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to create patch: %v", err)
		}

		output, ok := result.(PatchOutput)
		if !ok {
			t.Fatalf("Expected PatchOutput, got %T", result)
		}

		t.Logf("Create patch output:\n%s", output.PatchOutput)

		// Verify the patch looks correct
		if !strings.Contains(output.PatchOutput, "-line 2") {
			t.Errorf("Expected patch to contain removed line '-line 2'")
		}
		if !strings.Contains(output.PatchOutput, "+modified line 2") {
			t.Errorf("Expected patch to contain added line '+modified line 2'")
		}
		if !strings.Contains(output.PatchOutput, "+new line") {
			t.Errorf("Expected patch to contain added line '+new line'")
		}

		// Store the patch for next test
		createdPatch := output.PatchOutput

		// Test case: Apply patch (dry run)
		t.Run("Apply patch (dry run)", func(t *testing.T) {
			input := PatchInput{
				Mode:   "apply",
				Path:   testFilePath,
				Patch:  createdPatch,
				DryRun: true,
			}

			jsonInput, _ := json.Marshal(input)
			result, err := patchTool.Execute(jsonInput)

			if err != nil {
				t.Fatalf("Failed to apply patch (dry run): %v", err)
			}

			output, ok := result.(PatchOutput)
			if !ok {
				t.Fatalf("Expected PatchOutput, got %T", result)
			}

			t.Logf("Apply patch (dry run) output:\n%s", output.PatchOutput)

			// Verify file wasn't modified
			content, _ := os.ReadFile(testFilePath)
			if string(content) != originalContent {
				t.Errorf("File was modified despite dry run")
			}
		})

		// Test case: Apply patch
		t.Run("Apply patch", func(t *testing.T) {
			input := PatchInput{
				Mode:  "apply",
				Path:  testFilePath,
				Patch: createdPatch,
			}

			jsonInput, _ := json.Marshal(input)
			result, err := patchTool.Execute(jsonInput)

			if err != nil {
				t.Fatalf("Failed to apply patch: %v", err)
			}

			output, ok := result.(PatchOutput)
			if !ok {
				t.Fatalf("Expected PatchOutput, got %T", result)
			}

			t.Logf("Apply patch output:\n%s", output.PatchOutput)

			// Verify file was modified correctly
			content, _ := os.ReadFile(testFilePath)
			if string(content) != modifiedContent {
				t.Errorf("File content after patch doesn't match expected content")
				t.Logf("Expected:\n%s", modifiedContent)
				t.Logf("Got:\n%s", string(content))
			}
		})
	})

	// Test case: Create patch with file content (no original provided)
	t.Run("Create patch with file content", func(t *testing.T) {
		// Reset test file
		if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to reset test file: %v", err)
		}

		input := PatchInput{
			Mode:     "create",
			Path:     testFilePath,
			Modified: modifiedContent,
		}

		jsonInput, _ := json.Marshal(input)
		result, err := patchTool.Execute(jsonInput)

		if err != nil {
			t.Fatalf("Failed to create patch from file: %v", err)
		}

		output, ok := result.(PatchOutput)
		if !ok {
			t.Fatalf("Expected PatchOutput, got %T", result)
		}

		t.Logf("Create patch from file output:\n%s", output.PatchOutput)

		// Verify the patch looks correct
		if !strings.Contains(output.PatchOutput, "-line 2") {
			t.Errorf("Expected patch to contain removed line '-line 2'")
		}
		if !strings.Contains(output.PatchOutput, "+modified line 2") {
			t.Errorf("Expected patch to contain added line '+modified line 2'")
		}
		if !strings.Contains(output.PatchOutput, "+new line") {
			t.Errorf("Expected patch to contain added line '+new line'")
		}
	})

	// Test case: Apply non-existent patch
	t.Run("Apply patch to non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")

		input := PatchInput{
			Mode:  "apply",
			Path:  nonExistentPath,
			Patch: "--- nonexistent.txt\n+++ nonexistent.txt\n@@ -1 +1 @@\n-old\n+new\n",
		}

		jsonInput, _ := json.Marshal(input)
		_, err := patchTool.Execute(jsonInput)

		if err == nil {
			t.Errorf("Expected error when applying patch to non-existent file")
		}
	})

	// Test case: Invalid mode
	t.Run("Invalid mode", func(t *testing.T) {
		input := PatchInput{
			Mode: "invalid",
			Path: testFilePath,
		}

		jsonInput, _ := json.Marshal(input)
		_, err := patchTool.Execute(jsonInput)

		if err == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}
