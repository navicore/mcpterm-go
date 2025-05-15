package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrepTool(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grep-tool-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files with known content
	createTestFiles(t, tempDir)

	tool := NewGrepTool()

	t.Run("SimpleSearch", func(t *testing.T) {
		// Test a simple search pattern
		input, err := json.Marshal(GrepInput{
			Pattern: "TestFunction",
			Path:    tempDir,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.Equal(t, "TestFunction", grepResult.Pattern)
		assert.GreaterOrEqual(t, grepResult.FilesMatched, 2) // Should match in at least test1.go and test2.go
		assert.GreaterOrEqual(t, grepResult.TotalMatches, 2)
		assert.Empty(t, grepResult.Error)
	})

	t.Run("RegexPattern", func(t *testing.T) {
		// Test with a regex pattern
		input, err := json.Marshal(GrepInput{
			Pattern: "func [A-Z][a-z]+",
			Path:    tempDir,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.Equal(t, "func [A-Z][a-z]+", grepResult.Pattern)
		assert.GreaterOrEqual(t, grepResult.FilesMatched, 1)
		assert.GreaterOrEqual(t, grepResult.TotalMatches, 1)
		assert.Empty(t, grepResult.Error)
	})

	t.Run("FileFilter", func(t *testing.T) {
		// Test with file pattern filter
		input, err := json.Marshal(GrepInput{
			Pattern: "import",
			Path:    tempDir,
			Include: "*.go",
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.Equal(t, "import", grepResult.Pattern)
		assert.GreaterOrEqual(t, grepResult.TotalMatches, 2) // Both .go files have imports
		assert.Empty(t, grepResult.Error)

		// Make sure .txt file wasn't matched
		for _, fileResult := range grepResult.Results {
			assert.NotEqual(t, filepath.Join(tempDir, "test.txt"), fileResult.FilePath)
		}
	})

	t.Run("ExcludeFilter", func(t *testing.T) {
		// Test with exclude filter
		input, err := json.Marshal(GrepInput{
			Pattern: "test",
			Path:    tempDir,
			Exclude: "*.txt",
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)

		// Make sure .txt file wasn't matched
		for _, fileResult := range grepResult.Results {
			assert.NotEqual(t, filepath.Join(tempDir, "test.txt"), fileResult.FilePath)
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		// Test case insensitive search
		input, err := json.Marshal(GrepInput{
			Pattern:    "testfunction", // lowercase
			Path:       tempDir,
			IgnoreCase: true,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.GreaterOrEqual(t, grepResult.TotalMatches, 2) // Should find TestFunction despite case difference
		assert.Empty(t, grepResult.Error)
	})

	t.Run("MaxResults", func(t *testing.T) {
		// Test with max results
		input, err := json.Marshal(GrepInput{
			Pattern:    "test",
			Path:       tempDir,
			MaxResults: 2,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.LessOrEqual(t, grepResult.TotalMatches, 2) // Should be limited to 2 matches
		assert.True(t, grepResult.Truncated)              // Should be marked as truncated
	})

	t.Run("InvalidPattern", func(t *testing.T) {
		// Test with an invalid regex pattern
		input, err := json.Marshal(GrepInput{
			Pattern: "[unclosed",
			Path:    tempDir,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return error for invalid pattern

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.Equal(t, "[unclosed", grepResult.Pattern)
		assert.Contains(t, grepResult.Error, "Invalid pattern")
	})

	t.Run("NonexistentPath", func(t *testing.T) {
		// Test with a path that doesn't exist
		input, err := json.Marshal(GrepInput{
			Pattern: "test",
			Path:    filepath.Join(tempDir, "nonexistent"),
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return error for invalid path

		// Check the result
		grepResult, ok := result.(GrepResult)
		require.True(t, ok)
		assert.Equal(t, "test", grepResult.Pattern)
		assert.Contains(t, grepResult.Error, "Search failed")
	})
}

// createTestFiles creates sample files for testing
func createTestFiles(t *testing.T, dir string) {
	// Create a Go file with typical content
	test1Content := `package example

import (
    "fmt"
)

// TestFunction is a function for testing
func TestFunction() {
    fmt.Println("Hello, world!")
}
`
	err := os.WriteFile(filepath.Join(dir, "test1.go"), []byte(test1Content), 0644)
	require.NoError(t, err)

	// Create another Go file with similar content
	test2Content := `package example

import (
    "strings"
    "testing"
)

func TestFunctionWithArgs(t *testing.T) {
    s := "test string"
    if !strings.Contains(s, "test") {
        t.Fail()
    }
}
`
	err = os.WriteFile(filepath.Join(dir, "test2.go"), []byte(test2Content), 0644)
	require.NoError(t, err)

	// Create a non-Go file
	textContent := `This is a simple text file.
It contains some test content but isn't a Go file.
No TestFunction here, but the word test appears multiple times.
TEST in uppercase should only be found with case-insensitive search.
`
	err = os.WriteFile(filepath.Join(dir, "test.txt"), []byte(textContent), 0644)
	require.NoError(t, err)
}
