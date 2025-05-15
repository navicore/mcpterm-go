package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileDeleteTool(t *testing.T) {
	// Skip trash-related assertions on non-macOS platforms
	isMacOS := runtime.GOOS == "darwin"

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file-delete-tool-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewFileDeleteTool()

	t.Run("DeleteFile", func(t *testing.T) {
		// Create a test file
		testFilePath := filepath.Join(tempDir, "test_file.txt")
		err := os.WriteFile(testFilePath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(testFilePath)
		require.NoError(t, err)

		// Run the delete operation
		input, err := json.Marshal(FileDeleteInput{
			Path: testFilePath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		deleteOut, ok := result.(FileDeleteOutput)
		require.True(t, ok)
		assert.Equal(t, testFilePath, deleteOut.Path)
		assert.True(t, deleteOut.Deleted)

		// We can't verify the file is in trash because that would make the test unstable,
		// but we can check that the file is no longer in its original location
		_, err = os.Stat(testFilePath)
		assert.True(t, os.IsNotExist(err), "File should no longer exist at original path")

		// Only on macOS can we test trash functionality, and even then it's not directly verifiable
		if isMacOS {
			t.Log("On macOS, the file should have been moved to trash")
		}
	})

	t.Run("DeleteNonExistentFile", func(t *testing.T) {
		// Attempt to delete a file that doesn't exist
		nonExistentPath := filepath.Join(tempDir, "does_not_exist.txt")

		input, err := json.Marshal(FileDeleteInput{
			Path: nonExistentPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err) // Should not return an error, just indicate it wasn't deleted

		// Check the result
		deleteOut, ok := result.(FileDeleteOutput)
		require.True(t, ok)
		assert.Equal(t, nonExistentPath, deleteOut.Path)
		assert.False(t, deleteOut.Deleted)
		assert.Contains(t, deleteOut.Error, "does not exist")
	})

	t.Run("DeleteDirectory", func(t *testing.T) {
		// Create a test directory with some files
		testDirPath := filepath.Join(tempDir, "test_dir")
		err := os.Mkdir(testDirPath, 0755)
		require.NoError(t, err)

		// Create a file in the directory
		testFilePath := filepath.Join(testDirPath, "test_file.txt")
		err = os.WriteFile(testFilePath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Run the delete operation
		input, err := json.Marshal(FileDeleteInput{
			Path: testDirPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		deleteOut, ok := result.(FileDeleteOutput)
		require.True(t, ok)
		assert.Equal(t, testDirPath, deleteOut.Path)
		assert.True(t, deleteOut.Deleted)

		// Check that directory no longer exists
		_, err = os.Stat(testDirPath)
		assert.True(t, os.IsNotExist(err), "Directory should no longer exist")
	})

	t.Run("EmptyPath", func(t *testing.T) {
		// Test error when path is empty
		input, err := json.Marshal(FileDeleteInput{
			Path: "",
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err)

		// Check the result
		deleteOut, ok := result.(FileDeleteOutput)
		require.True(t, ok)
		assert.False(t, deleteOut.Deleted)
		assert.False(t, deleteOut.MovedToTrash)
		assert.Contains(t, deleteOut.Error, "required")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test error on invalid JSON input
		result, err := tool.Execute([]byte(`{invalid json`))
		assert.Error(t, err)

		// Check the result
		deleteOut, ok := result.(FileDeleteOutput)
		require.True(t, ok)
		assert.False(t, deleteOut.Deleted)
		assert.False(t, deleteOut.MovedToTrash)
		assert.Contains(t, deleteOut.Error, "Invalid input")
	})
}
