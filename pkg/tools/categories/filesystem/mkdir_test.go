package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkdirTool(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "mkdir-tool-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewMkdirTool()

	t.Run("DirectoryCreation", func(t *testing.T) {
		// Test simple directory creation
		newDir := filepath.Join(tempDir, "test_dir")
		input, err := json.Marshal(MkdirInput{
			Path: newDir,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.Equal(t, newDir, mkdirOut.Path)
		assert.True(t, mkdirOut.Created)

		// Verify directory exists
		_, err = os.Stat(newDir)
		assert.NoError(t, err)
	})

	t.Run("DirectoryAlreadyExists", func(t *testing.T) {
		// Test creating a directory that already exists
		existingDir := filepath.Join(tempDir, "existing_dir")
		err := os.Mkdir(existingDir, 0755)
		require.NoError(t, err)

		input, err := json.Marshal(MkdirInput{
			Path: existingDir,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err) // Should not return an error, just indicate it wasn't created

		// Check the result
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.Equal(t, existingDir, mkdirOut.Path)
		assert.False(t, mkdirOut.Created)
		assert.Contains(t, mkdirOut.Error, "already exists")
	})

	t.Run("CreateParentDirectories", func(t *testing.T) {
		// Test creating a nested directory structure with parents flag
		nestedDir := filepath.Join(tempDir, "parent/child/grandchild")
		input, err := json.Marshal(MkdirInput{
			Path:        nestedDir,
			MakeParents: true,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		require.NoError(t, err)

		// Check the result
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.Equal(t, nestedDir, mkdirOut.Path)
		assert.True(t, mkdirOut.Created)

		// Verify all directories exist
		_, err = os.Stat(nestedDir)
		assert.NoError(t, err)
	})

	t.Run("NoParentsFlagFailure", func(t *testing.T) {
		// Test failure when parent directories don't exist and parents flag is false
		nestedDir := filepath.Join(tempDir, "missing_parent/child")
		input, err := json.Marshal(MkdirInput{
			Path:        nestedDir,
			MakeParents: false,
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err) // Should return error when parent doesn't exist

		// Check the result indicates failure
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.Equal(t, nestedDir, mkdirOut.Path)
		assert.False(t, mkdirOut.Created)
		assert.NotEmpty(t, mkdirOut.Error)
	})

	t.Run("EmptyPath", func(t *testing.T) {
		// Test error when path is empty
		input, err := json.Marshal(MkdirInput{
			Path: "",
		})
		require.NoError(t, err)

		result, err := tool.Execute(input)
		assert.Error(t, err)

		// Check the result
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.False(t, mkdirOut.Created)
		assert.Contains(t, mkdirOut.Error, "required")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test error on invalid JSON input
		result, err := tool.Execute([]byte(`{invalid json`))
		assert.Error(t, err)

		// Check the result
		mkdirOut, ok := result.(MkdirOutput)
		require.True(t, ok)
		assert.False(t, mkdirOut.Created)
		assert.Contains(t, mkdirOut.Error, "Invalid input")
	})
}
