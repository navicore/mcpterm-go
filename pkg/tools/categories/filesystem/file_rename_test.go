package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRenameTool(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file-rename-tool-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewFileRenameTool()

	t.Run("RenameFile", func(t *testing.T) {
		// Create a test file
		oldPath := filepath.Join(tempDir, "test_file.txt")
		newPath := filepath.Join(tempDir, "renamed_file.txt")
		err := os.WriteFile(oldPath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(oldPath)
		require.NoError(t, err)

		// Run the rename operation
		input, err := json.Marshal(FileRenameInput{
			OldPath: oldPath,
			NewPath: newPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.Equal(t, oldPath, renameOut.OldPath)
		assert.Equal(t, newPath, renameOut.NewPath)
		assert.True(t, renameOut.Renamed)
		assert.False(t, renameOut.IsDirectory)
		assert.Empty(t, renameOut.Error)

		// Verify file doesn't exist at old path
		_, err = os.Stat(oldPath)
		assert.True(t, os.IsNotExist(err), "File should no longer exist at old path")

		// Verify file exists at new path
		_, err = os.Stat(newPath)
		assert.NoError(t, err, "File should exist at new path")
	})

	t.Run("RenameDirectory", func(t *testing.T) {
		// Create a test directory with a file in it
		oldDirPath := filepath.Join(tempDir, "test_dir")
		newDirPath := filepath.Join(tempDir, "renamed_dir")
		err := os.Mkdir(oldDirPath, 0755)
		require.NoError(t, err)

		// Create a file inside the directory
		testFilePath := filepath.Join(oldDirPath, "file_inside_dir.txt")
		err = os.WriteFile(testFilePath, []byte("test content in directory"), 0644)
		require.NoError(t, err)

		// Run the rename operation
		input, err := json.Marshal(FileRenameInput{
			OldPath: oldDirPath,
			NewPath: newDirPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.Equal(t, oldDirPath, renameOut.OldPath)
		assert.Equal(t, newDirPath, renameOut.NewPath)
		assert.True(t, renameOut.Renamed)
		assert.True(t, renameOut.IsDirectory)
		assert.Empty(t, renameOut.Error)

		// Verify directory doesn't exist at old path
		_, err = os.Stat(oldDirPath)
		assert.True(t, os.IsNotExist(err), "Directory should no longer exist at old path")

		// Verify directory exists at new path
		_, err = os.Stat(newDirPath)
		assert.NoError(t, err, "Directory should exist at new path")

		// Verify the file inside the directory was moved too
		movedFilePath := filepath.Join(newDirPath, "file_inside_dir.txt")
		_, err = os.Stat(movedFilePath)
		assert.NoError(t, err, "File inside the directory should have been moved too")
	})

	t.Run("MoveToNewLocation", func(t *testing.T) {
		// Create a test file
		oldPath := filepath.Join(tempDir, "source_file.txt")

		// Create a subdirectory to move the file to
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		newPath := filepath.Join(subDir, "moved_file.txt")

		err = os.WriteFile(oldPath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Run the rename/move operation
		input, err := json.Marshal(FileRenameInput{
			OldPath: oldPath,
			NewPath: newPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.True(t, renameOut.Renamed)

		// Verify file doesn't exist at old location
		_, err = os.Stat(oldPath)
		assert.True(t, os.IsNotExist(err))

		// Verify file exists at new location
		_, err = os.Stat(newPath)
		assert.NoError(t, err)
	})

	t.Run("NonExistentSource", func(t *testing.T) {
		// Attempt to rename a file that doesn't exist
		nonExistentPath := filepath.Join(tempDir, "does_not_exist.txt")
		newPath := filepath.Join(tempDir, "will_not_be_created.txt")

		input, err := json.Marshal(FileRenameInput{
			OldPath: nonExistentPath,
			NewPath: newPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return an error for non-existent source

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.Equal(t, nonExistentPath, renameOut.OldPath)
		assert.Equal(t, newPath, renameOut.NewPath)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "does not exist")
	})

	t.Run("NonExistentDestinationParent", func(t *testing.T) {
		// Create a test file
		oldPath := filepath.Join(tempDir, "test_file2.txt")
		err := os.WriteFile(oldPath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Try to move to a location with non-existent parent directory
		newPath := filepath.Join(tempDir, "non_existent_dir", "relocated_file.txt")

		input, err := json.Marshal(FileRenameInput{
			OldPath: oldPath,
			NewPath: newPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return an error for non-existent parent dir

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.Equal(t, oldPath, renameOut.OldPath)
		assert.Equal(t, newPath, renameOut.NewPath)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "does not exist")
	})

	t.Run("DestinationAlreadyExists", func(t *testing.T) {
		// Create source file
		oldPath := filepath.Join(tempDir, "source_file2.txt")
		err := os.WriteFile(oldPath, []byte("source content"), 0644)
		require.NoError(t, err)

		// Create destination file that already exists
		newPath := filepath.Join(tempDir, "existing_target.txt")
		err = os.WriteFile(newPath, []byte("existing content"), 0644)
		require.NoError(t, err)

		input, err := json.Marshal(FileRenameInput{
			OldPath: oldPath,
			NewPath: newPath,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return an error when destination exists

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "already exists")
	})

	t.Run("EmptyPaths", func(t *testing.T) {
		// Test when old path is empty
		input, err := json.Marshal(FileRenameInput{
			OldPath: "",
			NewPath: "some/path",
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err)
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "required")

		// Test when new path is empty
		input, err = json.Marshal(FileRenameInput{
			OldPath: "some/path",
			NewPath: "",
		})
		require.NoError(t, err)

		result, err = tool.Execute(input)
		assert.Error(t, err)
		renameOut, ok = result.(FileRenameOutput)
		require.True(t, ok)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "required")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test error on invalid JSON input
		result, err := tool.Execute([]byte(`{invalid json`))
		assert.Error(t, err)

		// Check the result
		renameOut, ok := result.(FileRenameOutput)
		require.True(t, ok)
		assert.False(t, renameOut.Renamed)
		assert.Contains(t, renameOut.Error, "Invalid input")
	})
}
