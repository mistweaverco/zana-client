package zana

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing
type MockPackageManager struct {
	UpdateFunc              func(sourceID string) bool
	IsSupportedProviderFunc func(provider string) bool
	AvailableProvidersFunc  func() []string
}

func (m *MockPackageManager) Update(sourceID string) bool {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(sourceID)
	}
	return false
}

func (m *MockPackageManager) IsSupportedProvider(provider string) bool {
	if m.IsSupportedProviderFunc != nil {
		return m.IsSupportedProviderFunc(provider)
	}
	return false
}

func (m *MockPackageManager) AvailableProviders() []string {
	if m.AvailableProvidersFunc != nil {
		return m.AvailableProvidersFunc()
	}
	return []string{}
}

func TestUpdateService(t *testing.T) {
	t.Run("new update service creation", func(t *testing.T) {
		service := NewUpdateService()
		assert.NotNil(t, service)
		assert.NotNil(t, service.localPackages)
		assert.NotNil(t, service.packageManager)
		assert.NotNil(t, service.output)
	})

	t.Run("new update service with custom dependencies", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{}
		mockPackageManager := &MockPackageManager{}
		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		assert.Equal(t, mockLocalPackages, service.localPackages)
		assert.Equal(t, mockPackageManager, service.packageManager)
		assert.Equal(t, mockOutput, service.output)
	})
}

func TestUpdateAllPackagesGolden(t *testing.T) {
	t.Run("update all packages with empty data", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{Packages: []local_packages_parser.LocalPackageItem{}}
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			&MockPackageManager{},
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.True(t, success)
		assert.Contains(t, mockOutput.Output, "No packages are currently installed")
	})

	t.Run("update all packages with all successful updates", func(t *testing.T) {
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

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				return true // All updates succeed
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.True(t, success)
		// Join all output and check for content
		allOutput := strings.Join(mockOutput.Output, "")
		assert.Contains(t, allOutput, "Found 2 installed packages")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:npm/test-package")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:pypi/black")
		assert.Contains(t, allOutput, "Successfully updated: 2")
		assert.Contains(t, allOutput, "Failed to update: 0")
	})

	t.Run("update all packages with mixed success", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/success-package", Version: "1.0.0"},
						{SourceID: "pkg:pypi/failed-package", Version: "2.0.0"},
					},
				}
			},
		}

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				// First package succeeds, second fails
				if sourceID == "pkg:npm/success-package" {
					return true
				}
				return false
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.False(t, success)
		// Join all output and check for content
		allOutput := strings.Join(mockOutput.Output, "")
		assert.Contains(t, allOutput, "Found 2 installed packages")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:npm/success-package")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:pypi/failed-package")
		assert.Contains(t, allOutput, "Successfully updated: 1")
		assert.Contains(t, allOutput, "Failed to update: 1")
	})

	t.Run("update all packages with all failed updates", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/failed-package1", Version: "1.0.0"},
						{SourceID: "pkg:pypi/failed-package2", Version: "2.0.0"},
					},
				}
			},
		}

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				return false // All updates fail
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.False(t, success)
		// Join all output and check for content
		allOutput := strings.Join(mockOutput.Output, "")
		assert.Contains(t, allOutput, "Found 2 installed packages")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:npm/failed-package1")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:pypi/failed-package2")
		assert.Contains(t, allOutput, "Successfully updated: 0")
		assert.Contains(t, allOutput, "Failed to update: 2")
	})
}

func TestUpdateCommand(t *testing.T) {
	t.Run("update command structure", func(t *testing.T) {
		assert.Equal(t, "update", updateCmd.Use)
		assert.Equal(t, "Update packages to their latest versions", updateCmd.Short)
		assert.NotEmpty(t, updateCmd.Long)
		assert.Contains(t, updateCmd.Aliases, "up")
	})

	t.Run("update command has all flag", func(t *testing.T) {
		allFlag := updateCmd.Flags().Lookup("all")
		assert.NotNil(t, allFlag)
		assert.Equal(t, "all", allFlag.Name)
		assert.Equal(t, "A", allFlag.Shorthand)
		assert.Equal(t, "Update all installed packages to their latest versions", allFlag.Usage)
	})
}

func TestUpdateCommandRunPaths(t *testing.T) {
	t.Run("prints error when no args and no --all", func(t *testing.T) {
		captured := &MockOutputWriter{}
		prevFactory := newUpdateService
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(&MockLocalPackagesProvider{}, &MockPackageManager{}, captured)
		}
		defer func() { newUpdateService = prevFactory }()

		updateCmd.SetArgs([]string{})
		updateCmd.Flags().Set("all", "false")
		updateCmd.Run(updateCmd, []string{})

		all := strings.Join(captured.Output, "\n")
		assert.Contains(t, all, "Please provide a package ID or use --all flag")
	})

	t.Run("validates pkg id and provider", func(t *testing.T) {
		// Invalid prefix
		out1 := &MockOutputWriter{}
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(&MockLocalPackagesProvider{}, &MockPackageManager{}, out1)
		}
		updateCmd.Run(updateCmd, []string{"invalid:id"})
		assert.Contains(t, strings.Join(out1.Output, "\n"), "Invalid package ID format")

		// Missing provider/package
		out2 := &MockOutputWriter{}
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(&MockLocalPackagesProvider{}, &MockPackageManager{}, out2)
		}
		updateCmd.Run(updateCmd, []string{"pkg:only"})
		assert.Contains(t, strings.Join(out2.Output, "\n"), "Expected 'pkg:provider/package-name'")

		// Unsupported provider
		out3 := &MockOutputWriter{}
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(
				&MockLocalPackagesProvider{},
				&MockPackageManager{IsSupportedProviderFunc: func(provider string) bool { return false }, AvailableProvidersFunc: func() []string { return []string{"npm"} }},
				out3,
			)
		}
		updateCmd.Run(updateCmd, []string{"pkg:unknown/pkg"})
		assert.Contains(t, strings.Join(out3.Output, "\n"), "Unsupported provider")
	})

	t.Run("updates when valid id", func(t *testing.T) {
		out := &MockOutputWriter{}
		mgrCalled := false
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(
				&MockLocalPackagesProvider{},
				&MockPackageManager{IsSupportedProviderFunc: func(provider string) bool { return true }, UpdateFunc: func(sourceID string) bool { mgrCalled = true; return true }},
				out,
			)
		}
		updateCmd.Run(updateCmd, []string{"pkg:npm/eslint"})
		assert.True(t, mgrCalled)
		assert.Contains(t, strings.Join(out.Output, "\n"), "Successfully updated pkg:npm/eslint")
	})

	t.Run("--all path calls UpdateAllPackages", func(t *testing.T) {
		out := &MockOutputWriter{}
		newUpdateService = func() *UpdateService {
			return NewUpdateServiceWithDependencies(
				&MockLocalPackagesProvider{GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
					return local_packages_parser.LocalPackageRoot{Packages: []local_packages_parser.LocalPackageItem{}}
				}},
				&MockPackageManager{},
				out,
			)
		}
		updateCmd.Flags().Set("all", "true")
		updateCmd.Run(updateCmd, []string{})
		updateCmd.Flags().Set("all", "false")
		assert.Contains(t, strings.Join(out.Output, "\n"), "Updating all installed packages to latest versions...")
	})
}

func TestMockPackageManager(t *testing.T) {
	t.Run("mock package manager default behavior", func(t *testing.T) {
		mock := &MockPackageManager{}

		// Test default behavior
		assert.False(t, mock.Update("test"))
		assert.False(t, mock.IsSupportedProvider("test"))
		assert.Empty(t, mock.AvailableProviders())
	})

	t.Run("mock package manager custom behavior", func(t *testing.T) {
		mock := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				return sourceID == "pkg:npm/success"
			},
			IsSupportedProviderFunc: func(provider string) bool {
				return provider == "npm"
			},
			AvailableProvidersFunc: func() []string {
				return []string{"npm", "pypi"}
			},
		}

		// Test custom behavior
		assert.True(t, mock.Update("pkg:npm/success"))
		assert.False(t, mock.Update("pkg:pypi/fail"))
		assert.True(t, mock.IsSupportedProvider("npm"))
		assert.False(t, mock.IsSupportedProvider("pypi"))
		assert.Equal(t, []string{"npm", "pypi"}, mock.AvailableProviders())
	})
}

func TestMockOutputWriter(t *testing.T) {
	t.Run("mock output writer default behavior", func(t *testing.T) {
		mock := &MockOutputWriter{}

		mock.Println("test")
		mock.Printf("format %s", "test")

		assert.Len(t, mock.Output, 2)
		assert.Contains(t, mock.Output, "test")
		assert.Contains(t, mock.Output, "format test")
	})

	t.Run("mock output writer custom behavior", func(t *testing.T) {
		captured := []string{}

		mock := &MockOutputWriter{
			PrintlnFunc: func(args ...interface{}) {
				captured = append(captured, "custom println")
			},
			PrintfFunc: func(format string, args ...interface{}) {
				captured = append(captured, "custom printf")
			},
		}

		mock.Println("test")
		mock.Printf("format %s", "test")

		assert.Len(t, captured, 2)
		assert.Contains(t, captured, "custom println")
		assert.Contains(t, captured, "custom printf")
	})
}

func TestUpdateLegacyFunctions(t *testing.T) {
	t.Run("legacy functions still work", func(t *testing.T) {
		// Test that the legacy functions still work for backward compatibility
		assert.NotPanics(t, func() {
			updateAllPackages()
		})
	})
}

func TestDefaultImplementations(t *testing.T) {
	t.Run("default package manager", func(t *testing.T) {
		manager := &defaultPackageManager{}

		// Test Update method (this will call the real providers.Update)
		// We can't easily test the actual result without real package managers,
		// but we can verify the method exists and doesn't panic
		assert.NotPanics(t, func() {
			manager.Update("pkg:npm/test")
		})

		// Test IsSupportedProvider
		assert.NotPanics(t, func() {
			result := manager.IsSupportedProvider("npm")
			assert.IsType(t, true, result)
		})

		// Test AvailableProviders
		assert.NotPanics(t, func() {
			result := manager.AvailableProviders()
			assert.IsType(t, []string{}, result)
		})
	})

	t.Run("default output writer", func(t *testing.T) {
		writer := &DefaultOutputWriter{}

		// Test that the methods exist and don't panic
		assert.NotPanics(t, func() {
			writer.Println("test")
		})

		assert.NotPanics(t, func() {
			writer.Printf("test %s", "format")
		})
	})
}

func TestUpdateCommandFullOutputGolden(t *testing.T) {
	t.Run("update all with mixed success/failure and full summary", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/eslint", Version: "1.0.0"},
						{SourceID: "pkg:pypi/black", Version: "2.0.0"},
						{SourceID: "pkg:golang/gopls", Version: "0.1.0"},
					},
				}
			},
		}

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				switch sourceID {
				case "pkg:npm/eslint":
					return true
				case "pkg:pypi/black":
					return false
				case "pkg:golang/gopls":
					return true
				default:
					return false
				}
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.False(t, success) // Mixed results should return false
		allOutput := strings.Join(mockOutput.Output, "\n")
		assert.Contains(t, allOutput, "Found 3 installed packages")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:npm/eslint")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:pypi/black")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:golang/gopls")
		assert.Contains(t, allOutput, "Successfully updated: 2")
		assert.Contains(t, allOutput, "Failed to update: 1")
	})

	t.Run("update all with all failures", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/eslint", Version: "1.0.0"},
						{SourceID: "pkg:pypi/black", Version: "2.0.0"},
					},
				}
			},
		}

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				return false // All updates fail
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.False(t, success)
		allOutput := strings.Join(mockOutput.Output, "\n")
		assert.Contains(t, allOutput, "Found 2 installed packages")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:npm/eslint")
		assert.Contains(t, allOutput, "✗ Failed to update pkg:pypi/black")
		assert.Contains(t, allOutput, "Successfully updated: 0")
		assert.Contains(t, allOutput, "Failed to update: 2")
	})

	t.Run("update all with all successes", func(t *testing.T) {
		mockLocalPackages := &MockLocalPackagesProvider{
			GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
				return local_packages_parser.LocalPackageRoot{
					Packages: []local_packages_parser.LocalPackageItem{
						{SourceID: "pkg:npm/eslint", Version: "1.0.0"},
					},
				}
			},
		}

		mockPackageManager := &MockPackageManager{
			UpdateFunc: func(sourceID string) bool {
				return true // All updates succeed
			},
		}

		mockOutput := &MockOutputWriter{}

		service := NewUpdateServiceWithDependencies(
			mockLocalPackages,
			mockPackageManager,
			mockOutput,
		)

		success := service.UpdateAllPackages()

		assert.True(t, success)
		allOutput := strings.Join(mockOutput.Output, "\n")
		assert.Contains(t, allOutput, "Found 1 installed packages")
		assert.Contains(t, allOutput, "✓ Successfully updated pkg:npm/eslint")
		assert.Contains(t, allOutput, "Successfully updated: 1")
		assert.Contains(t, allOutput, "Failed to update: 0")
	})

	t.Run("update single package failure", func(t *testing.T) {
		// Capture stdout to test the "Failed to update" message
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub the package manager to return false (failure)
		prev := newUpdateService
		newUpdateService = func() *UpdateService {
			return &UpdateService{
				packageManager: &MockPackageManager{
					UpdateFunc: func(sourceID string) bool {
						return false // Update fails
					},
					IsSupportedProviderFunc: func(provider string) bool {
						return true
					},
					AvailableProvidersFunc: func() []string {
						return []string{"npm"}
					},
				},
				output: &MockOutputWriter{
					PrintlnFunc: func(a ...interface{}) {
						fmt.Println(a...)
					},
					PrintfFunc: func(format string, a ...interface{}) {
						fmt.Printf(format, a...)
					},
				},
			}
		}
		defer func() { newUpdateService = prev }()

		updateCmd.Run(updateCmd, []string{"pkg:npm/eslint"})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Failed to update pkg:npm/eslint")
	})

	t.Run("update all with mixed results shows failure summary", func(t *testing.T) {
		// Capture stdout to test the "Failed to update some packages" message
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Stub the service to return false (mixed results)
		prev := newUpdateService
		newUpdateService = func() *UpdateService {
			return &UpdateService{
				localPackages: &MockLocalPackagesProvider{
					GetDataFunc: func(force bool) local_packages_parser.LocalPackageRoot {
						return local_packages_parser.LocalPackageRoot{
							Packages: []local_packages_parser.LocalPackageItem{
								{SourceID: "pkg:npm/eslint", Version: "1.0.0"},
								{SourceID: "pkg:pypi/black", Version: "2.0.0"},
							},
						}
					},
				},
				packageManager: &MockPackageManager{
					UpdateFunc: func(sourceID string) bool {
						return false // All updates fail
					},
				},
				output: &MockOutputWriter{
					PrintlnFunc: func(a ...interface{}) {
						fmt.Println(a...)
					},
					PrintfFunc: func(format string, a ...interface{}) {
						fmt.Printf(format, a...)
					},
				},
			}
		}
		defer func() { newUpdateService = prev }()

		// Set the --all flag and run the command
		updateCmd.Flags().Set("all", "true")
		updateCmd.Run(updateCmd, []string{})

		// Restore stdout and read
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		out := buf.String()

		assert.Contains(t, out, "Failed to update some packages")
	})
}
