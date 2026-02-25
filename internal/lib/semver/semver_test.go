package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with v prefix", "v1.2.3", "1.2.3"},
		{"without v prefix", "1.2.3", "1.2.3"},
		{"empty string", "", ""},
		{"single v", "v", ""},
		{"multiple v", "vv1.2.3", "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGreater(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		// Basic version comparisons
		{"v2 greater than v1", "1.2.3", "2.0.0", true},
		{"v1 greater than v2", "2.0.0", "1.2.3", false},
		{"equal versions", "1.2.3", "1.2.3", false},
		{"patch version greater", "1.2.3", "1.2.4", true},
		{"minor version greater", "1.2.3", "1.3.0", true},
		{"major version greater", "1.2.3", "2.0.0", true},

		// With v prefix
		{"with v prefix v2 greater", "v1.2.3", "v2.0.0", true},
		{"with v prefix v1 greater", "v2.0.0", "v1.2.3", false},

		// Incomplete versions (should be padded with zeros)
		{"incomplete v1", "1.2", "1.2.0", false},
		{"incomplete v2", "1.2.0", "1.2", false},
		{"single part v1", "1", "1.0.0", false},
		{"single part v2", "1.0.0", "1", false},
		// Empty versions should return false due to parsing errors
		{"empty v1", "", "1.0.0", false},
		{"empty v2", "1.0.0", "", false},

		// Edge cases
		{"zero versions", "0.0.0", "0.0.0", false},
		{"large numbers", "999.999.999", "1000.0.0", true},
		{"mixed formats", "v1.2.3", "2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGreater(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGreaterEdgeCases(t *testing.T) {
	// Test with invalid version strings that should return false
	invalidVersions := []string{
		"invalid",
		"1.2.abc",
		"1.abc.3",
		"abc.2.3",
		"1..3",
		".2.3",
		"1.2.",
	}

	for _, invalid := range invalidVersions {
		t.Run("invalid version: "+invalid, func(t *testing.T) {
			result := IsGreater("1.2.3", invalid)
			assert.False(t, result, "Should return false for invalid version: %s", invalid)
		})
	}
}

func TestIsGreaterVersionPadding(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{"v1 missing patch", "1.2", "1.2.1", true},
		{"v2 missing patch", "1.2.1", "1.2", false},
		{"v1 missing minor and patch", "1", "1.1.0", true},
		{"v2 missing minor and patch", "1.1.0", "1", false},
		{"both missing parts", "1", "2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGreater(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGreaterWithPreRelease(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		// Basic pre-release vs release comparisons
		// NOTE: When already on a non-numeric prerelease (e.g. dev/alpha),
		// that prerelease should outrank the corresponding stable version,
		// so there is no update.
		{"dev outranks stable when already on dev", "1.0.0-dev.1", "1.0.0", false},
		{"stable does not outrank dev", "1.0.0", "1.0.0-dev.1", false},
		{"same dev build", "1.0.0-dev.1", "1.0.0-dev.1", false},

		// Purely numeric prerelease should behave like standard SemVer:
		// prerelease is lower than the corresponding stable.
		{"numeric prerelease less than stable", "1.0.0-1", "1.0.0", true},
		{"stable greater than numeric prerelease", "1.0.0", "1.0.0-1", false},

		// More complex pre-release identifiers
		{"dev date build smaller", "7.0.0-dev.20260225.1", "7.0.0-dev.20260226.1", true},
		{"dev date build greater", "7.0.0-dev.20260226.1", "7.0.0-dev.20260225.1", false},

		// With v prefix and build metadata
		{"with v prefix and build meta", "v1.2.3-dev.1+build.1", "v1.2.3-dev.2+build.5", true},
		{"ignores build metadata when equal", "1.2.3+build.1", "1.2.3+build.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGreater(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
