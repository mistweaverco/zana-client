package zana

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type MockLocalPackagesProvider struct {
	GetDataFunc func(force bool) local_packages_parser.LocalPackageRoot
}

func (m *MockLocalPackagesProvider) GetData(force bool) local_packages_parser.LocalPackageRoot {
	if m.GetDataFunc != nil {
		return m.GetDataFunc(force)
	}
	return local_packages_parser.LocalPackageRoot{Packages: []local_packages_parser.LocalPackageItem{}}
}

type MockRegistryProvider struct {
	GetDataFunc          func(force bool) []registry_parser.RegistryItem
	GetLatestVersionFunc func(sourceID string) string
}

func (m *MockRegistryProvider) GetData(force bool) []registry_parser.RegistryItem {
	if m.GetDataFunc != nil {
		return m.GetDataFunc(force)
	}
	return []registry_parser.RegistryItem{}
}

func (m *MockRegistryProvider) GetLatestVersion(sourceID string) string {
	if m.GetLatestVersionFunc != nil {
		return m.GetLatestVersionFunc(sourceID)
	}
	return ""
}

type MockUpdateChecker struct {
	CheckIfUpdateIsAvailableFunc func(currentVersion, latestVersion string) (bool, string)
}

func (m *MockUpdateChecker) CheckIfUpdateIsAvailable(currentVersion, latestVersion string) (bool, string) {
	if m.CheckIfUpdateIsAvailableFunc != nil {
		return m.CheckIfUpdateIsAvailableFunc(currentVersion, latestVersion)
	}
	return false, ""
}

type MockFileDownloader struct {
	DownloadAndUnzipRegistryFunc func() error
}

func (m *MockFileDownloader) DownloadAndUnzipRegistry() error {
	if m.DownloadAndUnzipRegistryFunc != nil {
		return m.DownloadAndUnzipRegistryFunc()
	}
	return nil
}

// Golden file testing utilities
func getGoldenFilePath(testName string) string {
	return filepath.Join("testdata", testName+".golden")
}

func readGoldenFile(t *testing.T, testName string) string {
	path := getGoldenFilePath(testName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v", path, err)
	}
	return string(content)
}

func writeGoldenFile(t *testing.T, testName string, content string) {
	path := getGoldenFilePath(testName)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write golden file %s: %v", path, err)
	}
}

func captureOutput(t *testing.T, fn func()) string {
	// Redirect stdout to capture output
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Run the function
	fn()

	// Restore stdout
	os.Stdout = old
	w.Close()

	// Read captured output
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)

	return buf.String()
}

func TestListService(t *testing.T) {
	t.Run("new list service creation", func(t *testing.T) {
		// Mock the factory to avoid real dependencies
		prevFactory := newListServiceFunc
		newListServiceFunc = func() *ListService {
			return NewListServiceWithDependencies(
				&MockLocalPackagesProvider{},
				&MockRegistryProvider{
					GetDataFunc: func(force bool) []registry_parser.RegistryItem {
						return []registry_parser.RegistryItem{}
					},
				},
				&MockUpdateChecker{},
				&MockFileDownloader{},
			)
		}
		defer func() { newListServiceFunc = prevFactory }()

		// Now call NewListService() which will use our mocked factory
		service := NewListService()
		assert.NotNil(t, service)
		assert.NotNil(t, service.localPackages)
		assert.NotNil(t, service.registry)
		assert.NotNil(t, service.updateChecker)
		assert.NotNil(t, service.fileDownloader)
	})

	t.Run("new list service with custom dependencies", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{}
		mockRegistry := &MockRegistryProvider{}
		mockUpdateChecker := &MockUpdateChecker{}
		mockFileDownloader := &MockFileDownloader{}

		service := NewListServiceWithDependencies(
			mockLocalPackages,
			mockRegistry,
			mockUpdateChecker,
			mockFileDownloader,
		)

		assert.Equal(t, mockLocalPackages, service.localPackages)
		assert.Equal(t, mockRegistry, service.registry)
		assert.Equal(t, mockUpdateChecker, service.updateChecker)
		assert.Equal(t, mockFileDownloader, service.fileDownloader)
	})

	t.Run("ListInstalledPackages refreshes registry via downloader", func(t *testing.T) {
		called := false

		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				// Return empty package list so the method exits quickly.
				return local_packages_parser.LocalPackageRoot{Packages: []local_packages_parser.LocalPackageItem{}}
			},
		}

		mockFileDownloader := &MockFileDownloader{
			DownloadAndUnzipRegistryFunc: func() error {
				called = true
				return nil
			},
		}

		service := NewListServiceWithDependencies(
			mockLocalPackages,
			&MockRegistryProvider{},
			&MockUpdateChecker{},
			mockFileDownloader,
		)

		// We don't care about the actual output here, just that the
		// downloader is invoked as part of the listing process.
		_ = captureOutput(t, func() {
			service.ListInstalledPackages(nil)
		})

		assert.True(t, called, "expected registry downloader to be called")
	})

	t.Run("ListAllPackages refreshes registry via downloader", func(t *testing.T) {
		called := false

		mockRegistry := &MockRegistryProvider{
			GetDataFunc: func(force bool) []registry_parser.RegistryItem {
				// Return an empty registry so the method can proceed
				// down the empty-path logic without extra noise.
				return []registry_parser.RegistryItem{}
			},
		}

		mockFileDownloader := &MockFileDownloader{
			DownloadAndUnzipRegistryFunc: func() error {
				called = true
				return nil
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			mockFileDownloader,
		)

		_ = captureOutput(t, func() {
			service.ListAllPackages(nil)
		})

		assert.True(t, called, "expected registry downloader to be called")
	})
}

func TestListInstalledPackagesGolden(t *testing.T) {
	t.Run("list installed packages with empty data", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{Packages: []local_packages_parser.LocalPackageItem{}}
			},
		}

		service := NewListServiceWithDependencies(
			mockLocalPackages,
			&MockRegistryProvider{},
			&MockUpdateChecker{},
			&MockFileDownloader{},
		)

		output := captureOutput(t, func() {
			service.ListInstalledPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_installed_empty")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			writeGoldenFile(t, "list_installed_empty", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			// Compare with existing golden file
			expected := readGoldenFile(t, "list_installed_empty")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})

	t.Run("list installed packages with data", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/test-package", Version: "1.0.0"},
						{SourceID: "pkg:pypi/black", Version: "2.0.0"},
					},
				}
			},
		}

		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				return "1.1.0" // Newer version available
			},
		}

		mockUpdateChecker := &MockUpdateChecker{
			CheckIfUpdateIsAvailableFunc: func(currentVersion, latestVersion string) (bool, string) {
				return true, "Update available" // Simulate update available
			},
		}

		service := NewListServiceWithDependencies(
			mockLocalPackages,
			mockRegistry,
			mockUpdateChecker,
			&MockFileDownloader{},
		)

		output := captureOutput(t, func() {
			service.ListInstalledPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_installed_with_data")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			writeGoldenFile(t, "list_installed_with_data", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			// Compare with existing golden file
			expected := readGoldenFile(t, "list_installed_with_data")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})
}

func TestListAllPackagesGolden(t *testing.T) {
	t.Run("list all packages with empty registry", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetDataFunc: func(force bool) []registry_parser.RegistryItem {
				return []registry_parser.RegistryItem{}
			},
		}

		mockFileDownloader := &MockFileDownloader{
			DownloadAndUnzipRegistryFunc: func() error {
				return nil // Success
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			mockFileDownloader,
		)

		output := captureOutput(t, func() {
			service.ListAllPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_all_empty_registry")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			writeGoldenFile(t, "list_all_empty_registry", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			// Compare with existing golden file
			expected := readGoldenFile(t, "list_all_empty_registry")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})

	t.Run("list all packages with download failure", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetDataFunc: func(force bool) []registry_parser.RegistryItem {
				return []registry_parser.RegistryItem{}
			},
		}

		mockFileDownloader := &MockFileDownloader{
			DownloadAndUnzipRegistryFunc: func() error {
				return assert.AnError // Simulate download failure
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			mockFileDownloader,
		)

		output := captureOutput(t, func() {
			service.ListAllPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_all_download_failure")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			writeGoldenFile(t, "list_all_download_failure", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			// Compare with existing golden file
			expected := readGoldenFile(t, "list_all_download_failure")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})

	t.Run("list all packages with existing data", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetDataFunc: func(force bool) []registry_parser.RegistryItem {
				return []registry_parser.RegistryItem{
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:npm/test-package"},
						Version:     "1.0.0",
						Description: "Test package",
					},
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:pypi/black"},
						Version:     "2.0.0",
						Description: "Python formatter",
					},
				}
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			&MockFileDownloader{},
		)

		output := captureOutput(t, func() {
			service.ListAllPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_all_with_data")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			writeGoldenFile(t, "list_all_with_data", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			// Compare with existing golden file
			expected := readGoldenFile(t, "list_all_with_data")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})
}

func TestDownloadAndUnzipRegistryWrapper(t *testing.T) {
	t.Run("defaultFileDownloader delegates to function var", func(t *testing.T) {
		// Arrange: swap out function
		called := false
		prev := downloadAndUnzipRegistryFn
		downloadAndUnzipRegistryFn = func() error { called = true; return nil }
		defer func() { downloadAndUnzipRegistryFn = prev }()

		d := &defaultFileDownloader{}
		err := d.DownloadAndUnzipRegistry()

		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestCheckUpdateAvailability(t *testing.T) {
	t.Run("check update availability with empty latest version", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				return ""
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			&MockFileDownloader{},
		)

		updateInfo, hasUpdate := service.checkUpdateAvailability("pkg:npm/test", "1.0.0")
		assert.Equal(t, "", updateInfo)
		assert.False(t, hasUpdate)
	})

	t.Run("check update availability with latest version", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				return "2.0.0"
			},
		}

		mockUpdateChecker := &MockUpdateChecker{
			CheckIfUpdateIsAvailableFunc: func(currentVersion, latestVersion string) (bool, string) {
				return true, "Update available"
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			mockUpdateChecker,
			&MockFileDownloader{},
		)

		updateInfo, hasUpdate := service.checkUpdateAvailability("pkg:npm/test", "1.0.0")
		assert.Contains(t, updateInfo, "🔄 Update available: v2.0.0")
		assert.True(t, hasUpdate)
	})

	t.Run("check update availability with empty current version", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				return "2.0.0"
			},
		}

		mockUpdateChecker := &MockUpdateChecker{
			CheckIfUpdateIsAvailableFunc: func(currentVersion, latestVersion string) (bool, string) {
				return true, "Update available"
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			mockUpdateChecker,
			&MockFileDownloader{},
		)

		updateInfo, hasUpdate := service.checkUpdateAvailability("pkg:npm/test", "")
		assert.Contains(t, updateInfo, "🔄 Update available: v2.0.0")
		assert.True(t, hasUpdate)
	})

	t.Run("check update availability with latest current version", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				return "2.0.0"
			},
		}

		mockUpdateChecker := &MockUpdateChecker{
			CheckIfUpdateIsAvailableFunc: func(currentVersion, latestVersion string) (bool, string) {
				return false, "Up to date"
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			mockUpdateChecker,
			&MockFileDownloader{},
		)

		updateInfo, hasUpdate := service.checkUpdateAvailability("pkg:npm/test", "2.0.0")
		assert.Equal(t, "✅ Up to date", updateInfo)
		assert.False(t, hasUpdate)
	})
}

func TestListCommand(t *testing.T) {
	t.Run("list command structure", func(t *testing.T) {
		assert.Equal(t, "list", listCmd.Use)
		assert.Equal(t, "List packages", listCmd.Short)
		assert.NotEmpty(t, listCmd.Long)
		assert.Contains(t, listCmd.Aliases, "ls")
	})

	t.Run("list command has all flag", func(t *testing.T) {
		allFlag := listCmd.Flags().Lookup("all")
		assert.NotNil(t, allFlag)
		assert.Equal(t, "all", allFlag.Name)
		assert.Equal(t, "A", allFlag.Shorthand)
		assert.Equal(t, "List all available packages from the registry", allFlag.Usage)
	})
}

func TestListCommandRunPaths(t *testing.T) {
	t.Run("runs installed packages by default", func(t *testing.T) {
		// Inject factory to capture path
		calledInstalled := false
		prevFactory := newListService
		newListService = func() *ListService {
			// service with local packages empty to finish quickly
			return NewListServiceWithDependencies(
				&MockLocalPackagesProvider{GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
					return local_packages_parser.LocalPackageRoot{Packages: nil}
				}},
				&MockRegistryProvider{},
				&MockUpdateChecker{},
				&MockFileDownloader{},
			)
		}
		defer func() { newListService = prevFactory }()

		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		listCmd.SetArgs([]string{})
		listCmd.Run(listCmd, []string{})

		// Restore
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		buf.ReadFrom(r)
		out := buf.String()
		assert.Contains(t, out, "Locally Installed Packages")
		_ = calledInstalled
	})

	t.Run("runs all when --all", func(t *testing.T) {
		prevFactory := newListService
		newListService = func() *ListService {
			return NewListServiceWithDependencies(
				&MockLocalPackagesProvider{},
				&MockRegistryProvider{GetDataFunc: func(force bool) []registry_parser.RegistryItem { return []registry_parser.RegistryItem{} }},
				&MockUpdateChecker{},
				&MockFileDownloader{},
			)
		}
		defer func() { newListService = prevFactory }()

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		listCmd.Flags().Set("all", "true")
		listCmd.Run(listCmd, []string{})
		listCmd.Flags().Set("all", "false")

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		buf.ReadFrom(r)
		out := buf.String()
		assert.Contains(t, out, "All Available Packages")
	})
}

func TestGetProviderFromSourceID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		expected string
	}{
		{"npm package", "pkg:npm/package-name", "npm"},
		{"golang package", "pkg:golang/package-name", "golang"},
		{"pypi package", "pkg:pypi/package-name", "pypi"},
		{"cargo package", "pkg:cargo/package-name", "cargo"},
		{"unknown package", "pkg:unknown/package-name", "unknown"},
		{"no prefix", "npm/package-name", "unknown"},
		{"empty string", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getProviderFromSourceID(tt.sourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPackageNameFromSourceID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		expected string
	}{
		{"npm package", "pkg:npm/package-name", "package-name"},
		{"golang package", "pkg:golang/package-name", "package-name"},
		{"pypi package", "pkg:pypi/package-name", "package-name"},
		{"cargo package", "pkg:cargo/package-name", "package-name"},
		{"complex package name", "pkg:npm/@scope/package-name", "@scope/package-name"},
		{"no prefix", "npm/package-name", "package-name"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPackageNameFromSourceID(tt.sourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetProviderIcon(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"npm provider", "npm", "📦"},
		{"golang provider", "golang", "🐹"},
		{"pypi provider", "pypi", "🐍"},
		{"cargo provider", "cargo", "🦀"},
		{"unknown provider", "unknown", "📋"},
		{"empty provider", "", "📋"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getProviderIcon(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLegacyFunctions(t *testing.T) {
	t.Run("legacy functions still work", func(t *testing.T) {
		// Mock the factory to avoid real dependencies
		prevFactory := newListServiceFunc
		newListServiceFunc = func() *ListService {
			return NewListServiceWithDependencies(
				&MockLocalPackagesProvider{},
				&MockRegistryProvider{
					GetDataFunc: func(force bool) []registry_parser.RegistryItem {
						return []registry_parser.RegistryItem{
							{
								Source:      registry_parser.RegistryItemSource{ID: "pkg:npm/test-package"},
								Version:     "1.0.0",
								Description: "Test package for testing",
							},
						}
					},
				},
				&MockUpdateChecker{},
				&MockFileDownloader{},
			)
		}
		defer func() { newListServiceFunc = prevFactory }()

		// Test that the legacy functions still work for backward compatibility
		assert.NotPanics(t, func() {
			listInstalledPackages()
			listAllPackages()
			checkUpdateAvailability("pkg:npm/test", "1.0.0")
		})
	})
}

func TestListCommandFullOutputGolden(t *testing.T) {
	t.Run("list installed with mixed update statuses", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/eslint", Version: "1.0.0"},
						{SourceID: "pkg:pypi/black", Version: "2.0.0"},
						{SourceID: "pkg:golang/gopls", Version: "0.1.0"},
						{SourceID: "pkg:cargo/ripgrep", Version: "latest"},
					},
				}
			},
		}

		mockRegistry := &MockRegistryProvider{
			GetLatestVersionFunc: func(sourceID string) string {
				switch sourceID {
				case "pkg:npm/eslint":
					return "1.1.0"
				case "pkg:pypi/black":
					return "2.0.0"
				case "pkg:golang/gopls":
					return "0.2.0"
				case "pkg:cargo/ripgrep":
					return "1.0.0"
				default:
					return ""
				}
			},
		}

		mockUpdateChecker := &MockUpdateChecker{
			CheckIfUpdateIsAvailableFunc: func(currentVersion, latestVersion string) (bool, string) {
				if currentVersion == "latest" {
					return true, "Update available"
				}
				if currentVersion == "1.0.0" && latestVersion == "1.1.0" {
					return true, "Update available"
				}
				if currentVersion == "0.1.0" && latestVersion == "0.2.0" {
					return true, "Update available"
				}
				return false, "Up to date"
			},
		}

		service := NewListServiceWithDependencies(
			mockLocalPackages,
			mockRegistry,
			mockUpdateChecker,
			&MockFileDownloader{},
		)

		output := captureOutput(t, func() {
			service.ListInstalledPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_installed_mixed_updates")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			writeGoldenFile(t, "list_installed_mixed_updates", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			expected := readGoldenFile(t, "list_installed_mixed_updates")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})

	t.Run("list all with descriptions and mixed providers", func(t *testing.T) {
		mockRegistry := &MockRegistryProvider{
			GetDataFunc: func(force bool) []registry_parser.RegistryItem {
				return []registry_parser.RegistryItem{
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:npm/eslint"},
						Version:     "1.0.0",
						Description: "JavaScript linter",
					},
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:pypi/black"},
						Version:     "2.0.0",
						Description: "Python formatter",
					},
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:golang/gopls"},
						Version:     "0.2.0",
						Description: "Go language server",
					},
					{
						Source:      registry_parser.RegistryItemSource{ID: "pkg:cargo/ripgrep"},
						Version:     "1.0.0",
						Description: "Rust grep tool",
					},
				}
			},
		}

		service := NewListServiceWithDependencies(
			&MockLocalPackagesProvider{},
			mockRegistry,
			&MockUpdateChecker{},
			&MockFileDownloader{},
		)

		output := captureOutput(t, func() {
			service.ListAllPackages(nil)
		})

		goldenPath := getGoldenFilePath("list_all_with_descriptions")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			writeGoldenFile(t, "list_all_with_descriptions", output)
			t.Logf("Created golden file: %s", goldenPath)
		} else {
			expected := readGoldenFile(t, "list_all_with_descriptions")
			assert.Equal(t, expected, output, "Output doesn't match golden file")
		}
	})
}
