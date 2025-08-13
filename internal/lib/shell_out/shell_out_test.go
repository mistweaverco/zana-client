package shell_out

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellOut(t *testing.T) {
	t.Run("shell out with echo command", func(t *testing.T) {
		// Test with a simple echo command that should work on most systems
		exitCode, err := ShellOut("echo", []string{"hello"}, "", nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("shell out with command that exits non-zero", func(t *testing.T) {
		exitCode, err := ShellOut("false", []string{}, "", nil)
		assert.Error(t, err)
		assert.Equal(t, 1, exitCode)
	})

	t.Run("shell out with invalid command", func(t *testing.T) {
		// Test with a command that doesn't exist
		exitCode, err := ShellOut("nonexistentcommand12345", []string{}, "", nil)
		assert.Error(t, err)
		assert.Equal(t, -1, exitCode)
	})

	t.Run("shell out with custom directory", func(t *testing.T) {
		// Test with current directory
		currentDir, _ := os.Getwd()
		exitCode, shellErr := ShellOut("pwd", []string{}, currentDir, nil)
		assert.NoError(t, shellErr)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("shell out with custom environment", func(t *testing.T) {
		// Test with custom environment variable
		customEnv := []string{"CUSTOM_VAR=test_value"}
		exitCode, _ := ShellOut("echo", []string{"$CUSTOM_VAR"}, "", customEnv)
		// Note: This might not work as expected on all systems due to shell interpretation
		// But we can at least test that it doesn't panic
		assert.NotEqual(t, -1, exitCode) // Should not be the error exit code
	})
}

func TestHasCommand(t *testing.T) {
	t.Run("has command with echo", func(t *testing.T) {
		// echo should exist on most systems
		exists := HasCommand("echo", []string{}, nil)
		assert.True(t, exists)
	})

	t.Run("has command returns false on exit error", func(t *testing.T) {
		exists := HasCommand("false", []string{}, nil)
		assert.False(t, exists)
	})

	t.Run("has command with custom env and args", func(t *testing.T) {
		exists := HasCommand("sh", []string{"-c", "exit 0"}, []string{"CUSTOM_VAR=test"})
		assert.True(t, exists)
	})

	t.Run("has command with nonexistent command", func(t *testing.T) {
		// This command should not exist
		exists := HasCommand("nonexistentcommand12345", []string{}, nil)
		assert.False(t, exists)
	})
}

func TestShellOutCapture(t *testing.T) {
	t.Run("capture echo output", func(t *testing.T) {
		// Test capturing output from echo
		exitCode, output, err := ShellOutCapture("echo", []string{"hello world"}, "", nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, output, "hello world")
	})

	t.Run("capture command with error", func(t *testing.T) {
		// Test capturing output from a command that fails
		exitCode, output, err := ShellOutCapture("nonexistentcommand12345", []string{}, "", nil)
		assert.Error(t, err)
		assert.Equal(t, -1, exitCode)
		assert.Empty(t, output)
	})

	t.Run("capture exit error with output and code", func(t *testing.T) {
		exitCode, output, err := ShellOutCapture("sh", []string{"-c", "echo oops; exit 2"}, "", nil)
		assert.Error(t, err)
		assert.Equal(t, 2, exitCode)
		assert.Contains(t, output, "oops")
	})

	// Cover env merging path
	t.Run("capture with custom env merged", func(t *testing.T) {
		exitCode, output, err := ShellOutCapture("sh", []string{"-c", "echo $MY_VAR"}, "", []string{"MY_VAR=xyz"})
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
		assert.Contains(t, output, "xyz")
	})
}
