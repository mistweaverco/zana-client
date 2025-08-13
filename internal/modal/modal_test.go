package modal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModal(t *testing.T) {
	t.Run("modal creation", func(t *testing.T) {
		modal := New("Test message", "info")

		assert.Equal(t, "Test message", modal.Message)
		assert.Equal(t, "info", modal.Type)
		assert.False(t, modal.quitting)
		assert.Equal(t, 0, modal.width)
		assert.Equal(t, 0, modal.height)
	})

	t.Run("modal types", func(t *testing.T) {
		errorModal := New("Error message", "error")
		successModal := New("Success message", "success")
		warningModal := New("Warning message", "warning")
		infoModal := New("Info message", "info")

		assert.Equal(t, "error", errorModal.Type)
		assert.Equal(t, "success", successModal.Type)
		assert.Equal(t, "warning", warningModal.Type)
		assert.Equal(t, "info", infoModal.Type)
	})

	t.Run("modal view method exists", func(t *testing.T) {
		modal := New("Test message", "info")
		result := modal.View()

		// The view method should return a string
		assert.IsType(t, "", result)
		assert.NotEmpty(t, result)

		// Should contain the message
		assert.Contains(t, result, "Test message")
	})

	t.Run("modal update method exists", func(t *testing.T) {
		modal := New("Test message", "info")

		// Test that Update method exists and can be called
		// We can't easily test the actual tea.Msg handling without complex mocking
		// But we can verify the method exists and returns the right types
		updatedModal, cmd := modal.Update(nil)

		assert.IsType(t, Modal{}, updatedModal)
		assert.Nil(t, cmd) // Should return nil command for nil message
	})
}

func TestKeyBindings(t *testing.T) {
	t.Run("key bindings exist", func(t *testing.T) {
		// Test that the key bindings are properly defined
		assert.NotNil(t, keys.Quit)
		assert.NotNil(t, keys.Close)
	})
}
