package providers

// MockProviderFactory is a mock implementation for testing
type MockProviderFactory struct {
	MockNPMProvider      PackageManager
	MockPyPIProvider     PackageManager
	MockGolangProvider   PackageManager
	MockCargoProvider    PackageManager
	MockGitHubProvider   PackageManager
	MockGitLabProvider   PackageManager
	MockCodebergProvider PackageManager
	MockGemProvider      PackageManager
	MockComposerProvider PackageManager
	MockLuaRocksProvider PackageManager
	MockNuGetProvider    PackageManager
	MockOpamProvider     PackageManager
	MockOpenVSXProvider  PackageManager
	MockGenericProvider  PackageManager
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

func (f *MockProviderFactory) CreateGitHubProvider() PackageManager {
	if f.MockGitHubProvider != nil {
		return f.MockGitHubProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateGitLabProvider() PackageManager {
	if f.MockGitLabProvider != nil {
		return f.MockGitLabProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateCodebergProvider() PackageManager {
	if f.MockCodebergProvider != nil {
		return f.MockCodebergProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateGemProvider() PackageManager {
	if f.MockGemProvider != nil {
		return f.MockGemProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateComposerProvider() PackageManager {
	if f.MockComposerProvider != nil {
		return f.MockComposerProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateLuaRocksProvider() PackageManager {
	if f.MockLuaRocksProvider != nil {
		return f.MockLuaRocksProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateNuGetProvider() PackageManager {
	if f.MockNuGetProvider != nil {
		return f.MockNuGetProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateOpamProvider() PackageManager {
	if f.MockOpamProvider != nil {
		return f.MockOpamProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateOpenVSXProvider() PackageManager {
	if f.MockOpenVSXProvider != nil {
		return f.MockOpenVSXProvider
	}
	return &MockPackageManager{}
}

func (f *MockProviderFactory) CreateGenericProvider() PackageManager {
	if f.MockGenericProvider != nil {
		return f.MockGenericProvider
	}
	return &MockPackageManager{}
}
