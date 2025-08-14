package files

import (
	"archive/zip"
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnzipBasic tests basic unzip functionality
func TestUnzipBasic(t *testing.T) {
	t.Run("successful unzip with files and directories", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Create a real zip archive with files and directories
		filesToZip := map[string]string{
			"file1.txt":             "Hello, World!",
			"dir1/":                 "", // Directory entry
			"dir1/file2.json":       `{"key": "value"}`,
			"dir1/subdir/":          "", // Directory entry
			"dir1/subdir/file3.txt": "Nested file content.",
		}

		// Mock the zip opener to return our real zip archive
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return createRealZipArchive(filesToZip)
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		srcPath := "/path/to/mock_archive.zip"
		destPath := "/tmp/unzipped_files"

		// Call the function under test
		err := Unzip(srcPath, destPath)
		require.NoError(t, err)

		// Verify that all directories were created correctly
		dirs := []string{
			"/tmp/unzipped_files/dir1",
			"/tmp/unzipped_files/dir1/subdir",
		}
		for _, dir := range dirs {
			exists, _ := afero.IsDir(mockFS.fs, dir)
			assert.True(t, exists, "Expected directory %s to exist", dir)
		}
	})

	t.Run("unzip with zip open error", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock the zip opener to return an error
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return nil, errors.New("zip open error")
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		// This should fail due to zip open error
		err := Unzip("test.zip", "/dest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "zip open error")
	})

	t.Run("unzip with destination directory creation error", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return errors.New("mkdir error")
			},
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock the zip opener to return a successful zip archive
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return &MockZipArchive{
					Files: []*zip.File{},
				}, nil
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		// This should fail due to destination directory creation error
		err := Unzip("test.zip", "/dest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create destination directory")
	})
}

// TestUnzipEdgeCases tests edge cases in the Unzip function
func TestUnzipEdgeCases(t *testing.T) {
	t.Run("unzip with empty zip archive", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock the zip opener to return an empty zip archive
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return &MockZipArchive{
					Files: []*zip.File{},
				}, nil
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		// This should succeed with empty archive
		err := Unzip("test.zip", "/dest")
		assert.NoError(t, err)
	})

	t.Run("unzip with only directory entries", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Create a real zip archive with only directories
		filesToZip := map[string]string{
			"dir1/":        "", // Directory entry
			"dir1/subdir/": "", // Directory entry
		}

		// Mock the zip opener to return our real zip archive
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return createRealZipArchive(filesToZip)
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		// This should succeed with only directories
		err := Unzip("test.zip", "/dest")
		assert.NoError(t, err)

		// Verify directories were created
		dirs := []string{
			"/dest/dir1",
			"/dest/dir1/subdir",
		}
		for _, dir := range dirs {
			exists, _ := afero.IsDir(mockFS.fs, dir)
			assert.True(t, exists, "Expected directory %s to exist", dir)
		}
	})

	t.Run("unzip with zip slip protection", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Create a real zip archive with a malicious path
		filesToZip := map[string]string{
			"../../../etc/passwd": "malicious content",
		}

		// Mock the zip opener to return our real zip archive
		mockZipOpener := &MockZipFileOpener{
			OpenFunc: func(name string) (ZipArchive, error) {
				return createRealZipArchive(filesToZip)
			},
		}
		SetZipFileOpener(mockZipOpener)
		defer ResetDependencies()

		// This should fail due to zip slip protection
		err := Unzip("test.zip", "/dest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})
}
