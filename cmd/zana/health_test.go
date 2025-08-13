package zana

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/stretchr/testify/assert"
)

func TestHealthCommand(t *testing.T) {
	t.Run("health command structure", func(t *testing.T) {
		assert.Equal(t, "health", healthCmd.Use)
		assert.Equal(t, "Check system health and requirements", healthCmd.Short)
		assert.NotEmpty(t, healthCmd.Long)
		// Note: We can't easily test Args since it's a function type
	})

	t.Run("health command has no subcommands", func(t *testing.T) {
		assert.Empty(t, healthCmd.Commands())
	})
}

func TestDisplayRequirement(t *testing.T) {
	t.Run("prints expected output for met and missing", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Execute
		displayRequirement("NPM", true, "Node.js package manager for JavaScript packages")
		displayRequirement("Python", false, "Python interpreter for Python packages")

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		out := buf.String()
		assert.Contains(t, out, "✅ NPM: Available")
		assert.Contains(t, out, "Node.js package manager for JavaScript packages")
		assert.Contains(t, out, "❌ Python: Missing")
		assert.Contains(t, out, "Python interpreter for Python packages")
	})
}

func TestHealthCommandRun(t *testing.T) {
	t.Run("prints all requirement statuses and overall", func(t *testing.T) {
		// stub requirements
		prev := checkRequirementsFn
		checkRequirementsFn = func() (r providers.CheckRequirementsResult) {
			return providers.CheckRequirementsResult{HasNPM: true, HasPython: false, HasPythonDistutils: true, HasGo: true, HasCargo: false}
		}
		defer func() { checkRequirementsFn = prev }()

		// capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		healthCmd.Run(healthCmd, []string{})

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "NPM")
		assert.Contains(t, out, "Python")
		assert.Contains(t, out, "Python Distutils")
		assert.Contains(t, out, "Go")
		assert.Contains(t, out, "Cargo")
		assert.Contains(t, out, "Some requirements are not met")
	})

	t.Run("prints success message when all requirements met", func(t *testing.T) {
		// stub requirements - all met
		prev := checkRequirementsFn
		checkRequirementsFn = func() (r providers.CheckRequirementsResult) {
			return providers.CheckRequirementsResult{HasNPM: true, HasPython: true, HasPythonDistutils: true, HasGo: true, HasCargo: true}
		}
		defer func() { checkRequirementsFn = prev }()

		// capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		healthCmd.Run(healthCmd, []string{})

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "✅ All requirements are met! Your system is ready to use Zana.")
		assert.NotContains(t, out, "Some requirements are not met")
	})
}
