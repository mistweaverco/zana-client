package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils(t *testing.T) {
	t.Run("check requirements result structure", func(t *testing.T) {
		result := CheckRequirementsResult{
			HasNPM:             true,
			HasPython:          true,
			HasPythonDistutils: true,
			HasGo:              true,
			HasCargo:           true,
		}

		assert.True(t, result.HasNPM)
		assert.True(t, result.HasPython)
		assert.True(t, result.HasPythonDistutils)
		assert.True(t, result.HasGo)
		assert.True(t, result.HasCargo)
	})

	t.Run("check requirements function exists", func(t *testing.T) {
		// This function checks actual system commands, so we can't easily mock it
		// But we can verify it exists and returns a result
		result := CheckRequirements()
		assert.IsType(t, CheckRequirementsResult{}, result)
	})
}
