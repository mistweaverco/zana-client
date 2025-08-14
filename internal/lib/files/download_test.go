package files

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestDownloadBasic tests basic download functionality
func TestDownloadBasic(t *testing.T) {
	t.Run("download function exists", func(t *testing.T) {
		// This test just verifies the function exists and can be called
		assert.NotNil(t, Download)
	})

	t.Run("download with mock HTTP client success", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock HTTP client that returns a successful response
		mockClient := &MockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return &http.Response{
					Body: &MockReadCloser{
						ReadFunc: func(p []byte) (n int, err error) {
							// Return EOF after reading some data to avoid infinite loops
							if len(p) > 0 {
								return 1, io.EOF
							}
							return 0, io.EOF
						},
						CloseFunc: func() error {
							return nil // Mock body close succeeds
						},
					},
				}, nil
			},
		}
		SetHTTPClient(mockClient)
		defer ResetDependencies()

		// Test download - should succeed
		err := Download("http://example.com/test", "/dest/test")
		assert.NoError(t, err)
	})

	t.Run("download with HTTP error", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock HTTP client that returns an error
		mockClient := &MockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return nil, errors.New("HTTP error")
			},
		}
		SetHTTPClient(mockClient)
		defer ResetDependencies()

		// Test download - should fail due to HTTP error
		err := Download("http://example.com/test", "/dest/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP error")
	})
}

// TestDownloadEdgeCases tests edge cases in the Download function
func TestDownloadEdgeCases(t *testing.T) {
	t.Run("download with empty response body", func(t *testing.T) {
		// Create an in-memory filesystem for testing
		mockFS := &MockFileSystem{
			fs: afero.NewMemMapFs(),
		}
		SetFileSystem(mockFS)
		defer ResetDependencies()

		// Mock HTTP client that returns response with empty body
		mockClient := &MockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return &http.Response{
					Body: &MockReadCloser{
						ReadFunc: func(p []byte) (n int, err error) {
							return 0, io.EOF // Empty body
						},
						CloseFunc: func() error {
							return nil
						},
					},
				}, nil
			},
		}
		SetHTTPClient(mockClient)
		defer ResetDependencies()

		// Test download with empty body - should succeed
		err := Download("http://example.com/test", "/dest/test")
		assert.NoError(t, err)
	})
}
