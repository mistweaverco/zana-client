package providers

// MockProviderFactory is a mock implementation for testing
type MockProviderFactory struct {
	MockNPMProvider    PackageManager
	MockPyPIProvider   PackageManager
	MockGolangProvider PackageManager
	MockCargoProvider  PackageManager
}

func (f *MockProviderFactory) CreateNPMProvider() PackageManager {
	if f.MockNPMProvider != nil {
		return f.MockNPMProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreatePyPIProvider() PackageManager {
	if f.MockPyPIProvider != nil {
		return f.MockPyPIProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateGolangProvider() PackageManager {
	if f.MockGolangProvider != nil {
		return f.MockGolangProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateCargoProvider() PackageManager {
	if f.MockCargoProvider != nil {
		return f.MockCargoProvider
	}
	return &MockPackageManager{}
}
