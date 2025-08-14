package providers

// PackageManager defines the interface for package management operations
type PackageManager interface {
	Install(sourceID, version string) bool
	Remove(sourceID string) bool
	Update(sourceID string) bool
	getLatestVersion(packageName string) (string, error)
}

// MockPackageManager is a mock implementation for testing
type MockPackageManager struct {
	InstallFunc          func(sourceID, version string) bool
	RemoveFunc           func(sourceID string) bool
	UpdateFunc           func(sourceID string) bool
	GetLatestVersionFunc func(packageName string) (string, error)
}

func (m *MockPackageManager) Install(sourceID, version string) bool {
	if m.InstallFunc != nil {
		return m.InstallFunc(sourceID, version)
	}
	return false
}

func (m *MockPackageManager) Remove(sourceID string) bool {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(sourceID)
	}
	return false
}

func (m *MockPackageManager) Update(sourceID string) bool {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(sourceID)
	}
	return false
}

func (m *MockPackageManager) getLatestVersion(packageName string) (string, error) {
	if m.GetLatestVersionFunc != nil {
		return m.GetLatestVersionFunc(packageName)
	}
	return "", nil
}

// ProviderFactory creates package managers
type ProviderFactory interface {
	CreateNPMProvider() PackageManager
	CreatePyPIProvider() PackageManager
	CreateGolangProvider() PackageManager
	CreateCargoProvider() PackageManager
}

// DefaultProviderFactory is the default implementation
type DefaultProviderFactory struct{}

func (f *DefaultProviderFactory) CreateNPMProvider() PackageManager {
	return NewProviderNPM()
}

func (f *DefaultProviderFactory) CreatePyPIProvider() PackageManager {
	return NewProviderPyPi()
}

func (f *DefaultProviderFactory) CreateGolangProvider() PackageManager {
	return NewProviderGolang()
}

func (f *DefaultProviderFactory) CreateCargoProvider() PackageManager {
	return NewProviderCargo()
}
