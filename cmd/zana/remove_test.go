package zana

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveCommand(t *testing.T) {
	t.Run("remove command structure", func(t *testing.T) {
		assert.Equal(t, "remove <pkgId>", removeCmd.Use)
		assert.Equal(t, "Remove a package", removeCmd.Short)
		assert.NotEmpty(t, removeCmd.Long)
		// Note: We can't easily test Args since it's a function type
		assert.Contains(t, removeCmd.Aliases, "rm")
		assert.Contains(t, removeCmd.Aliases, "delete")
	})

	t.Run("remove command has no subcommands", func(t *testing.T) {
		assert.Empty(t, removeCmd.Commands())
	})
}

func TestRemoveCommandRunPaths(t *testing.T) {
	t.Run("invalid id format", func(t *testing.T) {
		removeCmd.Run(removeCmd, []string{"invalid"})
	})

	t.Run("unsupported provider", func(t *testing.T) {
		prevSupp := isSupportedProviderFn
		prevAvail := availableProvidersFn
		isSupportedProviderFn = func(p string) bool { return false }
		availableProvidersFn = func() []string { return []string{"npm"} }
		defer func() { isSupportedProviderFn = prevSupp; availableProvidersFn = prevAvail }()
		removeCmd.Run(removeCmd, []string{"pkg:unknown/x"})
	})

	t.Run("successful remove", func(t *testing.T) {
		prevSupp := isSupportedProviderFn
		prevRemove := removePackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		removePackageFn = func(id string) bool { return true }
		defer func() { isSupportedProviderFn = prevSupp; removePackageFn = prevRemove }()
		removeCmd.Run(removeCmd, []string{"pkg:npm/eslint"})
	})

	t.Run("failed remove", func(t *testing.T) {
		prevSupp := isSupportedProviderFn
		prevRemove := removePackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		removePackageFn = func(id string) bool { return false }
		defer func() { isSupportedProviderFn = prevSupp; removePackageFn = prevRemove }()
		removeCmd.Run(removeCmd, []string{"pkg:npm/eslint"})
	})
}

func TestRemoveCommandFullOutputGolden(t *testing.T) {
	t.Run("remove success", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevRemove := removePackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		removePackageFn = func(id string) bool { return true }
		defer func() { isSupportedProviderFn = prevSupp; removePackageFn = prevRemove }()

		removeCmd.Run(removeCmd, []string{"pkg:npm/eslint"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Removing pkg:npm/eslint")
		assert.Contains(t, out, "Successfully removed pkg:npm/eslint")
	})

	t.Run("remove failure", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevRemove := removePackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		removePackageFn = func(id string) bool { return false }
		defer func() { isSupportedProviderFn = prevSupp; removePackageFn = prevRemove }()

		removeCmd.Run(removeCmd, []string{"pkg:npm/eslint"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Removing pkg:npm/eslint")
		assert.Contains(t, out, "Failed to remove pkg:npm/eslint")
	})
}
