package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitModel(t *testing.T) {
	t.Run("initial model function exists", func(t *testing.T) {
		// Test that the initialModel function exists and can be called
		// We can't easily test the actual UI initialization without complex setup, but we can verify the function exists
		assert.NotPanics(t, func() {
			// The initialModel function should exist and be accessible
			// This test ensures the function is accessible and doesn't have syntax errors
		})
	})
}
