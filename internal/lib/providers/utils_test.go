package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAllProvidersHealth(t *testing.T) {
	t.Run("returns health status for all providers", func(t *testing.T) {
		statuses := CheckAllProvidersHealth()

		// Should return all providers
		assert.Greater(t, len(statuses), 0)

		// Check that we have expected providers
		providerNames := make(map[string]bool)
		for _, status := range statuses {
			providerNames[status.Provider] = true
			// Each status should have a description
			assert.NotEmpty(t, status.Description)
			// If not available, should have RequiredTool
			if !status.Available {
				assert.NotEmpty(t, status.RequiredTool)
			}
		}

		// Verify key providers are included
		assert.True(t, providerNames["npm"])
		assert.True(t, providerNames["pypi"])
		assert.True(t, providerNames["generic"])
	})
}
