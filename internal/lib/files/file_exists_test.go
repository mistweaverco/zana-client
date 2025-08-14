package files

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileExists tests the FileExists function
func TestFileExists(t *testing.T) {
	t.Run("file exists with non-existing file", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Test with non-existing file
		result := FileExists("/non/existing/file")
		t.Logf("FileExists('/non/existing/file') = %v", result)
		assert.False(t, result)
	})

	t.Run("file exists with existing file", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Create a file in the in-memory filesystem
		file, err := mockFS.fs.Create("/test_file")
		require.NoError(t, err)
		file.Close()

		// Test with existing file
		result := FileExists("/test_file")
		t.Logf("FileExists('/test_file') = %v", result)
		assert.True(t, result)
	})

	t.Run("file exists with empty path", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Test with empty path
		result := FileExists("")
		t.Logf("FileExists('') = %v", result)
		assert.False(t, result)
	})

	t.Run("file exists with directory", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Create a directory in the in-memory filesystem
		err := mockFS.fs.MkdirAll("/test_dir", 0755)
		require.NoError(t, err)

		// Test with directory path
		result := FileExists("/test_dir")
		t.Logf("FileExists('/test_dir') = %v", result)
		assert.True(t, result)
	})

	t.Run("file exists with stat error", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, errors.New("stat error")
			},
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Test with stat error - should return false
		result := FileExists("/test_file")
		t.Logf("FileExists('/test_file') with stat error = %v", result)
		assert.False(t, result)
	})
}
