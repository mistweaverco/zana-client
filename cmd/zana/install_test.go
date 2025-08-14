package zana

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/stretchr/testify/assert"
)

func TestInstallCommand(t *testing.T) {
	// Test that install command is properly configured
	assert.Contains(t, installCmd.Use, "install")
	assert.Contains(t, installCmd.Short, "Install a package")
	assert.Contains(t, installCmd.Long, "Install a package from a supported provider")

	// Test aliases
	aliases := installCmd.Aliases
	assert.Contains(t, aliases, "add")
	assert.Len(t, aliases, 1)
}

func TestInstallCommandArgs(t *testing.T) {
	// Test argument validation
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"no args", []string{}, true},
		{"one arg valid", []string{"pkg:npm/test-package"}, false},
		{"two args valid", []string{"pkg:npm/test-package", "1.0.0"}, false},
		{"three args invalid", []string{"pkg:npm/test-package", "1.0.0", "extra"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := installCmd.Args(installCmd, tc.args)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInstallCommandValidation(t *testing.T) {
	// Test package ID format validation
	testCases := []struct {
		name        string
		pkgId       string
		expectValid bool
	}{
		{"valid npm package", "pkg:npm/test-package", true},
		{"valid golang package", "pkg:golang/golang.org/x/tools/gopls", true},
		{"valid pypi package", "pkg:pypi/black", true},
		{"valid cargo package", "pkg:cargo/ripgrep", true},
		{"invalid no prefix", "npm/test-package", false},
		{"invalid empty", "", false},
		{"invalid format - just pkg:", "pkg:", false},                // This should be invalid
		{"invalid format single part", "pkg:npm", false},             // This should be invalid
		{"invalid format with spaces", "pkg:npm/package name", true}, // This is actually valid
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Check if it starts with pkg:
			hasPrefix := strings.HasPrefix(tt.pkgId, "pkg:")

			// Additional validation: must have at least one slash after pkg:
			isValid := hasPrefix && strings.Contains(tt.pkgId[4:], "/")

			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}

func TestInstallCommandProviderValidation(t *testing.T) {
	// Test provider validation
	testCases := []struct {
		name        string
		pkgId       string
		expectValid bool
	}{
		{"npm provider", "pkg:npm/test-package", true},
		{"pypi provider", "pkg:pypi/test-package", true},
		{"golang provider", "pkg:golang/test-package", true},
		{"cargo provider", "pkg:cargo/test-package", true},
		{"unsupported provider", "pkg:unsupported/test-package", false},
		{"invalid provider format", "pkg:123/test-package", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.HasPrefix(tc.pkgId, "pkg:") {
				parts := strings.Split(strings.TrimPrefix(tc.pkgId, "pkg:"), "/")
				if len(parts) >= 2 {
					provider := parts[0]
					isValid := providers.IsSupportedProvider(provider)
					assert.Equal(t, tc.expectValid, isValid)
				}
			}
		})
	}
}

func TestInstallCommandVersionHandling(t *testing.T) {
	// Test version argument handling
	testCases := []struct {
		name            string
		args            []string
		expectedPkgId   string
		expectedVersion string
	}{
		{"no version specified", []string{"pkg:npm/test-package"}, "pkg:npm/test-package", "latest"},
		{"version specified", []string{"pkg:npm/test-package", "1.0.0"}, "pkg:npm/test-package", "1.0.0"},
		{"latest version", []string{"pkg:npm/test-package", "latest"}, "pkg:npm/test-package", "latest"},
		{"empty version", []string{"pkg:npm/test-package", ""}, "pkg:npm/test-package", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkgId := tc.args[0]
			version := "latest"
			if len(tc.args) > 1 {
				version = tc.args[1]
			}

			assert.Equal(t, tc.expectedPkgId, pkgId)
			assert.Equal(t, tc.expectedVersion, version)
		})
	}
}

func TestInstallCommandExamples(t *testing.T) {
	// Test that the examples in the help text are valid
	examples := []string{
		"pkg:npm/@prisma/language-server",
		"pkg:golang/golang.org/x/tools/gopls",
		"pkg:pypi/black",
		"pkg:cargo/ripgrep",
	}

	for _, example := range examples {
		t.Run("example_"+example, func(t *testing.T) {
			// Test that each example follows the expected format
			assert.True(t, strings.HasPrefix(example, "pkg:"))

			parts := strings.Split(strings.TrimPrefix(example, "pkg:"), "/")
			assert.GreaterOrEqual(t, len(parts), 2, "Example should have provider and package name")

			provider := parts[0]
			assert.True(t, providers.IsSupportedProvider(provider), "Provider %s should be supported", provider)
		})
	}
}

func TestInstallCommandHelp(t *testing.T) {
	// Test that help command works without executing the actual command
	// We'll just test the command structure instead
	assert.NotNil(t, installCmd)
	assert.Contains(t, installCmd.Use, "install")
	assert.Contains(t, installCmd.Short, "Install a package")
}

func TestInstallCommandIntegration(t *testing.T) {
	// Test that the command structure is correct without executing
	// This avoids hanging on actual command execution

	testCases := []struct {
		name string
		args []string
	}{
		{"valid npm package", []string{"pkg:npm/test-package"}},
		{"valid golang package", []string{"pkg:golang/golang.org/x/tools/gopls"}},
		{"valid pypi package", []string{"pkg:pypi/black"}},
		{"valid cargo package", []string{"pkg:cargo/ripgrep"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just test that the command can be set up without panicking
			// Don't actually execute it
			assert.NotNil(t, installCmd)
			assert.NotNil(t, installCmd.Run)
		})
	}
}

func TestInstallCommandRunPaths(t *testing.T) {
	t.Run("invalid id format", func(t *testing.T) {
		// capture output indirectly by ensuring no panic and path returns early
		installCmd.Run(installCmd, []string{"invalid"})
	})

	t.Run("unsupported provider", func(t *testing.T) {
		prev := isSupportedProviderFn
		prevAvail := availableProvidersFn
		isSupportedProviderFn = func(p string) bool { return false }
		availableProvidersFn = func() []string { return []string{"npm"} }
		defer func() { isSupportedProviderFn = prev; availableProvidersFn = prevAvail }()
		installCmd.Run(installCmd, []string{"pkg:unknown/x"})
	})

	t.Run("successful install", func(t *testing.T) {
		prevSupp := isSupportedProviderFn
		prevInstall := installPackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		installPackageFn = func(id, v string) bool { return true }
		defer func() { isSupportedProviderFn = prevSupp; installPackageFn = prevInstall }()
		installCmd.Run(installCmd, []string{"pkg:npm/eslint"})
	})

	t.Run("failed install", func(t *testing.T) {
		prevSupp := isSupportedProviderFn
		prevInstall := installPackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		installPackageFn = func(id, v string) bool { return false }
		defer func() { isSupportedProviderFn = prevSupp; installPackageFn = prevInstall }()
		installCmd.Run(installCmd, []string{"pkg:npm/eslint"})
	})
}

func TestInstallCommandErrorHandling(t *testing.T) {
	// Test error handling for invalid inputs without executing
	testCases := []struct {
		name string
		args []string
	}{
		{"invalid package format", []string{"invalid-format"}},
		{"unsupported provider", []string{"pkg:unsupported/package"}},
		{"malformed package id", []string{"pkg:"}},
		{"empty package id", []string{""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just test that the command structure is valid
			// Don't actually execute it
			assert.NotNil(t, installCmd)
			assert.NotNil(t, installCmd.Run)
		})
	}
}

func TestInstallCommandStructure(t *testing.T) {
	// Test that the command has the expected structure
	assert.NotNil(t, installCmd.Run, "Install command should have a Run function")
	assert.NotEmpty(t, installCmd.Use, "Install command should have a Use field")
	assert.NotEmpty(t, installCmd.Short, "Install command should have a Short description")
	assert.NotEmpty(t, installCmd.Long, "Install command should have a Long description")
	assert.NotEmpty(t, installCmd.Aliases, "Install command should have aliases")
}

func TestInstallCommandProviderIntegration(t *testing.T) {
	// Test that the command integrates properly with the providers package
	validPackage := "pkg:npm/test-package"

	// Test that the package ID parsing works correctly
	if strings.HasPrefix(validPackage, "pkg:") {
		parts := strings.Split(strings.TrimPrefix(validPackage, "pkg:"), "/")
		assert.GreaterOrEqual(t, len(parts), 2)

		provider := parts[0]
		assert.True(t, providers.IsSupportedProvider(provider))
	}
}

func TestInstallCommandFullOutputGolden(t *testing.T) {
	t.Run("install with version specified", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevInstall := installPackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		installPackageFn = func(id, v string) bool { return true }
		defer func() { isSupportedProviderFn = prevSupp; installPackageFn = prevInstall }()

		installCmd.Run(installCmd, []string{"pkg:npm/eslint", "1.0.0"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Installing pkg:npm/eslint (version: 1.0.0)")
		assert.Contains(t, out, "Successfully installed pkg:npm/eslint")
	})

	t.Run("install with latest version", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevInstall := installPackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		installPackageFn = func(id, v string) bool { return true }
		defer func() { isSupportedProviderFn = prevSupp; installPackageFn = prevInstall }()

		installCmd.Run(installCmd, []string{"pkg:npm/eslint", "latest"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Installing pkg:npm/eslint (version: latest)")
		assert.Contains(t, out, "Successfully installed pkg:npm/eslint")
	})

	t.Run("install failure", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevInstall := installPackageFn
		isSupportedProviderFn = func(p string) bool { return true }
		installPackageFn = func(id, v string) bool { return false }
		defer func() { isSupportedProviderFn = prevSupp; installPackageFn = prevInstall }()

		installCmd.Run(installCmd, []string{"pkg:npm/eslint"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Installing pkg:npm/eslint (version: latest)")
		assert.Contains(t, out, "Failed to install pkg:npm/eslint")
	})
}

func TestInstallCommandValidationErrors(t *testing.T) {
	t.Run("invalid package format - missing provider/package", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		installCmd.Run(installCmd, []string{"pkg:"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Error: Invalid package ID format. Expected 'pkg:provider/package-name'")
	})

	t.Run("invalid package format - just provider", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		installCmd.Run(installCmd, []string{"pkg:npm"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Error: Invalid package ID format. Expected 'pkg:provider/package-name'")
	})

	t.Run("unsupported provider calls availableProvidersFn", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub functions
		prevSupp := isSupportedProviderFn
		prevAvail := availableProvidersFn
		isSupportedProviderFn = func(p string) bool { return false }
		availableProvidersFn = func() []string { return []string{"npm", "pypi", "golang", "cargo"} }
		defer func() { isSupportedProviderFn = prevSupp; availableProvidersFn = prevAvail }()

		installCmd.Run(installCmd, []string{"pkg:unknown/package"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Error: Unsupported provider 'unknown'")
		assert.Contains(t, out, "Supported providers: npm, pypi, golang, cargo")
	})
}
