package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupportedProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected bool
	}{
		{"npm provider", "npm", true},
		{"pypi provider", "pypi", true},
		{"golang provider", "golang", true},
		{"cargo provider", "cargo", true},
		{"unsupported provider", "unsupported", false},
		{"empty string", "", false},
		{"case sensitive npm", "NPM", false},
		{"case sensitive pypi", "PYPI", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSupportedProvider(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		expected Provider
	}{
		{"npm source", "pkg:npm/package-name", ProviderNPM},
		{"pypi source", "pkg:pypi/package-name", ProviderPyPi},
		{"golang source", "pkg:golang/package-name", ProviderGolang},
		{"cargo source", "pkg:cargo/package-name", ProviderCargo},
		{"unsupported source", "pkg:unsupported/package-name", ProviderUnsupported},
		{"empty source", "", ProviderUnsupported},
		{"no prefix", "npm/package-name", ProviderUnsupported},
		{"different prefix", "other:package-name", ProviderUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectProvider(tt.sourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckIfUpdateIsAvailable(t *testing.T) {
	tests := []struct {
		name          string
		localVersion  string
		remoteVersion string
		expected      bool
	}{
		{"update available", "1.2.3", "2.0.0", true},
		{"no update available", "2.0.0", "1.2.3", false},
		{"same version", "1.2.3", "1.2.3", false},
		{"local newer", "2.0.0", "1.2.3", false},
		// Empty versions should return false due to parsing errors
		{"empty local version", "", "1.2.3", false},
		{"empty remote version", "1.2.3", "", false},
		{"both empty", "", "", false},
		{"with v prefix", "v1.2.3", "v2.0.0", true},
		{"mixed prefixes", "v1.2.3", "2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := CheckIfUpdateIsAvailable(tt.localVersion, tt.remoteVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckIfUpdateIsAvailableReturnValue(t *testing.T) {
	// Test that the function returns the expected version string
	result, version := CheckIfUpdateIsAvailable("1.2.3", "2.0.0")
	assert.True(t, result)
	assert.Equal(t, "2.0.0", version)

	result, version = CheckIfUpdateIsAvailable("2.0.0", "1.2.3")
	assert.False(t, result)
	assert.Equal(t, "", version)
}

func TestAvailableProviders(t *testing.T) {
	// Test that all expected providers are available
	expectedProviders := []string{"npm", "pypi", "golang", "cargo", "github", "gitlab", "codeberg", "gem", "composer", "luarocks", "nuget", "opam", "openvsx", "generic"}

	assert.Len(t, AvailableProviders, len(expectedProviders))

	for _, expected := range expectedProviders {
		assert.Contains(t, AvailableProviders, expected)
	}
}

func TestProviderConstants(t *testing.T) {
	// Test that provider constants are properly defined
	assert.Equal(t, Provider(0), ProviderNPM)
	assert.Equal(t, Provider(1), ProviderPyPi)
	assert.Equal(t, Provider(2), ProviderGolang)
	assert.Equal(t, Provider(3), ProviderCargo)
	assert.Equal(t, Provider(4), ProviderGitHub)
	assert.Equal(t, Provider(5), ProviderGitLab)
	assert.Equal(t, Provider(6), ProviderCodeberg)
	assert.Equal(t, Provider(7), ProviderGem)
	assert.Equal(t, Provider(8), ProviderComposer)
	assert.Equal(t, Provider(9), ProviderLuaRocks)
	assert.Equal(t, Provider(10), ProviderNuGet)
	assert.Equal(t, Provider(11), ProviderOpam)
	assert.Equal(t, Provider(12), ProviderOpenVSX)
	assert.Equal(t, Provider(13), ProviderGeneric)
	assert.Equal(t, Provider(14), ProviderUnsupported)
}

func TestInstallWithMockFactory(t *testing.T) {
	// Create mock providers that return predictable results
	mockNPM := &MockPackageManager{
		InstallFunc: func(sourceID, version string) bool {
			return sourceID == "pkg:npm/test-package"
		},
	}

	mockPyPI := &MockPackageManager{
		InstallFunc: func(sourceID, version string) bool {
			return sourceID == "pkg:pypi/test-package"
		},
	}

	mockGolang := &MockPackageManager{
		InstallFunc: func(sourceID, version string) bool {
			return sourceID == "pkg:golang/test-package"
		},
	}

	mockCargo := &MockPackageManager{
		InstallFunc: func(sourceID, version string) bool {
			return sourceID == "pkg:cargo/test-package"
		},
	}

	// Create mock factory
	mockFactory := &MockProviderFactory{
		MockNPMProvider:    mockNPM,
		MockPyPIProvider:   mockPyPI,
		MockGolangProvider: mockGolang,
		MockCargoProvider:  mockCargo,
	}

	// Set the mock factory
	SetProviderFactory(mockFactory)
	defer ResetProviderFactory()

	tests := []struct {
		name     string
		sourceId string
		version  string
		expected bool
	}{
		{"npm package", "pkg:npm/test-package", "1.0.0", true},
		{"pypi package", "pkg:pypi/test-package", "1.0.0", true},
		{"golang package", "pkg:golang/test-package", "1.0.0", true},
		{"cargo package", "pkg:cargo/test-package", "1.0.0", true},
		{"unsupported package", "pkg:unsupported/test-package", "1.0.0", false},
		{"empty source", "", "1.0.0", false},
		{"empty version", "pkg:npm/test-package", "", true},
		{"wrong npm package", "pkg:npm/wrong-package", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Install(tt.sourceId, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveWithMockFactory(t *testing.T) {
	// Create mock providers that return predictable results
	mockNPM := &MockPackageManager{
		RemoveFunc: func(sourceID string) bool {
			return sourceID == "pkg:npm/test-package"
		},
	}

	mockPyPI := &MockPackageManager{
		RemoveFunc: func(sourceID string) bool {
			return sourceID == "pkg:pypi/test-package"
		},
	}

	mockGolang := &MockPackageManager{
		RemoveFunc: func(sourceID string) bool {
			return sourceID == "pkg:golang/test-package"
		},
	}

	mockCargo := &MockPackageManager{
		RemoveFunc: func(sourceID string) bool {
			return sourceID == "pkg:cargo/test-package"
		},
	}

	// Create mock factory
	mockFactory := &MockProviderFactory{
		MockNPMProvider:    mockNPM,
		MockPyPIProvider:   mockPyPI,
		MockGolangProvider: mockGolang,
		MockCargoProvider:  mockCargo,
	}

	// Set the mock factory
	SetProviderFactory(mockFactory)
	defer ResetProviderFactory()

	tests := []struct {
		name     string
		sourceId string
		expected bool
	}{
		{"npm package", "pkg:npm/test-package", true},
		{"pypi package", "pkg:pypi/test-package", true},
		{"golang package", "pkg:golang/test-package", true},
		{"cargo package", "pkg:cargo/test-package", true},
		{"unsupported package", "pkg:unsupported/test-package", false},
		{"empty source", "", false},
		{"wrong npm package", "pkg:npm/wrong-package", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Remove(tt.sourceId)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateWithMockFactory(t *testing.T) {
	// Create mock providers that return predictable results
	mockNPM := &MockPackageManager{
		UpdateFunc: func(sourceID string) bool {
			return sourceID == "pkg:npm/test-package"
		},
	}

	mockPyPI := &MockPackageManager{
		UpdateFunc: func(sourceID string) bool {
			return sourceID == "pkg:pypi/test-package"
		},
	}

	mockGolang := &MockPackageManager{
		UpdateFunc: func(sourceID string) bool {
			return sourceID == "pkg:golang/test-package"
		},
	}

	mockCargo := &MockPackageManager{
		UpdateFunc: func(sourceID string) bool {
			return sourceID == "pkg:cargo/test-package"
		},
	}

	// Create mock factory
	mockFactory := &MockProviderFactory{
		MockNPMProvider:    mockNPM,
		MockPyPIProvider:   mockPyPI,
		MockGolangProvider: mockGolang,
		MockCargoProvider:  mockCargo,
	}

	// Set the mock factory
	SetProviderFactory(mockFactory)
	defer ResetProviderFactory()

	tests := []struct {
		name     string
		sourceId string
		expected bool
	}{
		{"npm package", "pkg:npm/test-package", true},
		{"pypi package", "pkg:pypi/test-package", true},
		{"golang package", "pkg:golang/test-package", true},
		{"cargo package", "pkg:cargo/test-package", true},
		{"unsupported package", "pkg:unsupported/test-package", false},
		{"empty source", "", false},
		{"wrong npm package", "pkg:npm/wrong-package", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Update(tt.sourceId)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncAllWithMockFactory(t *testing.T) {
	// Create mock providers
	mockNPM := &MockPackageManager{}
	mockPyPI := &MockPackageManager{}
	mockGolang := &MockPackageManager{}
	mockCargo := &MockPackageManager{}

	// Create mock factory
	mockFactory := &MockProviderFactory{
		MockNPMProvider:    mockNPM,
		MockPyPIProvider:   mockPyPI,
		MockGolangProvider: mockGolang,
		MockCargoProvider:  mockCargo,
	}

	// Set the mock factory
	SetProviderFactory(mockFactory)
	defer ResetProviderFactory()

	// Test that SyncAll doesn't panic
	assert.NotPanics(t, func() {
		SyncAll()
	})
}

func TestProviderDetectionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		sourceId string
		expected Provider
	}{
		{"very long source", "pkg:npm/" + string(make([]byte, 1000)), ProviderNPM},
		{"special characters", "pkg:npm/package@#$%", ProviderNPM},
		{"numbers only", "pkg:123/456", ProviderUnsupported},
		{"unicode characters", "pkg:npm/package-测试", ProviderNPM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectProvider(tt.sourceId)
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestFactoriesAndGithubProvider(t *testing.T) {
	// Default factory creates providers
	f := &DefaultProviderFactory{}
	assert.NotNil(t, f.CreateNPMProvider())
	assert.NotNil(t, f.CreatePyPIProvider())
	assert.NotNil(t, f.CreateGolangProvider())
	assert.NotNil(t, f.CreateCargoProvider())

	// MockPackageManager getLatestVersion default path
	m := &MockPackageManager{}
	ver, err := m.getLatestVersion("anything")
	assert.NoError(t, err)
	assert.Equal(t, "", ver)

	// GitHubProvider Install
	g := &GitHubProvider{}
	g.Install("pkg:github/owner/repo", "latest")
}
