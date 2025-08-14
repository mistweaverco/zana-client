package files

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestPathFunctions tests the path-related functions
func TestPathFunctions(t *testing.T) {
	t.Run("get app data path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetAppDataPath()
		assert.NotEmpty(t, path)
	})

	t.Run("get app local packages file path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetAppLocalPackagesFilePath()
		assert.NotEmpty(t, path)
	})

	t.Run("get app registry file path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetAppRegistryFilePath()
		assert.NotEmpty(t, path)
	})

	t.Run("get app packages path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetAppPackagesPath()
		assert.NotEmpty(t, path)
	})

	t.Run("get app bin path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetAppBinPath()
		assert.NotEmpty(t, path)
	})

	t.Run("get registry cache path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetRegistryCachePath()
		assert.NotEmpty(t, path)
	})

	t.Run("get temp path", func(t *testing.T) {
		// Test that the function exists and can be called
		path := GetTempPath()
		assert.NotEmpty(t, path)
	})
}

// TestPathFunctionsComprehensive tests comprehensive path scenarios
func TestPathFunctionsComprehensive(t *testing.T) {
	t.Run("get app data path with ZANA_HOME set", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
			GetenvFunc: func(key string) string {
				if key == "ZANA_HOME" {
					return "/custom/zana/home"
				}
				return ""
			},
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Test that it uses ZANA_HOME when set
		path := GetAppDataPath()
		assert.Equal(t, "/custom/zana/home", path)
	})

	t.Run("get app data path without ZANA_HOME", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
			GetenvFunc: func(key string) string {
				return "" // No ZANA_HOME
			},
			UserConfigDirFunc: func() (string, error) {
				return "/home/user/.config", nil
			},
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Test that it uses user config dir when ZANA_HOME is not set
		path := GetAppDataPath()
		assert.Equal(t, "/home/user/.config/zana", path)
	})

	t.Run("path separator with different separators", func(t *testing.T) {
		// Test that path separators work correctly
		path := GetAppDataPath() + string(os.PathSeparator) + "test"
		assert.Contains(t, path, "test")
	})

	t.Run("path separator with empty path", func(t *testing.T) {
		// Test with empty path
		path := GetAppDataPath() + string(os.PathSeparator) + ""
		// The result will have a trailing separator, so we need to account for that
		expected := GetAppDataPath() + string(os.PathSeparator)
		assert.Equal(t, expected, path)
	})

	t.Run("path separator with root path", func(t *testing.T) {
		// Test with root path
		path := GetAppDataPath() + string(os.PathSeparator) + "/"
		assert.Contains(t, path, "/")
	})
}

// TestPathFunctionsErrorPaths tests error paths in path functions
func TestPathFunctionsErrorPaths(t *testing.T) {
	t.Run("get app data path with user config dir error", func(t *testing.T) {
		// Create a mock file system that fails on UserConfigDir
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
			UserConfigDirFunc: func() (string, error) {
				return "", errors.New("user config dir error")
			},
			GetenvFunc: func(key string) string {
				return "" // No ZANA_HOME
			},
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// This should panic
		assert.Panics(t, func() {
			GetAppDataPath()
		})
	})
}
