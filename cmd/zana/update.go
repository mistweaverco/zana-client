package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

// UpdateService handles update operations with dependency injection
type UpdateService struct {
	localPackages LocalPackagesProvider
	output        OutputWriter
}

// OutputWriter defines the interface for writing output (for testing)
type OutputWriter interface {
	Println(args ...interface{})
	Printf(format string, args ...interface{})
}

// DefaultOutputWriter implements OutputWriter using fmt
type DefaultOutputWriter struct{}

func (d *DefaultOutputWriter) Println(args ...interface{}) {
	fmt.Println(args...)
}

func (d *DefaultOutputWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

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

// MockOutputWriter is a mock implementation for testing
type MockOutputWriter struct {
	PrintlnFunc func(args ...interface{})
	PrintfFunc  func(format string, args ...interface{})
	Output      []string
}

func (m *MockOutputWriter) Println(args ...interface{}) {
	if m.PrintlnFunc != nil {
		m.PrintlnFunc(args...)
	}
	// Capture output for testing
	if len(args) > 0 {
		m.Output = append(m.Output, fmt.Sprint(args...))
	}
}

func (m *MockOutputWriter) Printf(format string, args ...interface{}) {
	if m.PrintfFunc != nil {
		m.PrintfFunc(format, args...)
	}
	// Capture output for testing
	m.Output = append(m.Output, fmt.Sprintf(format, args...))
}

// NewUpdateService creates a new UpdateService with default dependencies
func NewUpdateService() *UpdateService {
	return &UpdateService{
		localPackages: &defaultLocalPackagesProvider{},
		output:        &DefaultOutputWriter{},
	}
}

// NewUpdateServiceWithDependencies creates a new UpdateService with custom dependencies
func NewUpdateServiceWithDependencies(
	localPackages LocalPackagesProvider,
	output OutputWriter,
) *UpdateService {
	return &UpdateService{
		localPackages: localPackages,
		output:        output,
	}
}

// updatePackage updates a single package using the provider factory system
func (us *UpdateService) updatePackage(sourceID string) bool {
	// Use the provider factory system which can be mocked in tests
	return providers.Update(sourceID)
}

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update packages to their latest versions",
	Long: `Update packages to their latest versions.

Examples:
  zana update pkg:npm/eslint
  zana update pkg:golang/golang.org/x/tools/gopls pkg:npm/prettier
  zana update pkg:pypi/black pkg:cargo/ripgrep
  zana update --all (update all installed packages)`,
	Args: cobra.MinimumNArgs(0), // Allow no args if --all is used
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")

		if allFlag {
			// Update all installed packages
			service := newUpdateService()
			service.output.Println("Updating all installed packages to latest versions...")

			success := service.UpdateAllPackages()

			if success {
				service.output.Println("Successfully updated all packages")
			} else {
				service.output.Println("Failed to update some packages")
			}
			return
		}

		// Check if package IDs are provided
		if len(args) == 0 {
			service := newUpdateService()
			service.output.Println("Error: Please provide package IDs or use --all flag")
			return
		}

		// Validate all package IDs first
		packages := args
		for _, pkgId := range packages {
			if !strings.HasPrefix(pkgId, "pkg:") {
				service := newUpdateService()
				service.output.Printf("Error: Invalid package ID format '%s'. Must start with 'pkg:'\n", pkgId)
				return
			}

			// Parse provider from package ID
			parts := strings.Split(strings.TrimPrefix(pkgId, "pkg:"), "/")
			if len(parts) < 2 {
				service := newUpdateService()
				service.output.Printf("Error: Invalid package ID format '%s'. Expected 'pkg:provider/package-name'\n", pkgId)
				return
			}

			provider := parts[0]
			if !providers.IsSupportedProvider(provider) {
				service := newUpdateService()
				service.output.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, pkgId, strings.Join(providers.AvailableProviders, ", "))
				return
			}
		}

		// Update individual packages
		service := newUpdateService()
		service.output.Printf("Updating %d package(s) to latest versions...\n", len(packages))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for _, pkgId := range packages {
			service.output.Printf("Updating %s...\n", pkgId)

			// Update the package using the service method (which can be mocked in tests)
			success := service.updatePackage(pkgId)
			if success {
				service.output.Printf("✓ Successfully updated %s\n", pkgId)
				successCount++
			} else {
				service.output.Printf("✗ Failed to update %s\n", pkgId)
				failedCount++
				allSuccess = false
			}
		}

		// Print summary
		service.output.Printf("\nUpdate Summary:\n")
		service.output.Printf("  Successfully updated: %d\n", successCount)
		service.output.Printf("  Failed to update: %d\n", failedCount)

		if allSuccess {
			service.output.Printf("All packages updated successfully!\n")
		} else {
			service.output.Printf("Some packages failed to update.\n")
		}
	},
}

func init() {
	updateCmd.Flags().BoolP("all", "A", false, "Update all installed packages to their latest versions")
}

// newUpdateService is a factory to allow test injection
var newUpdateService = NewUpdateService

// UpdateAllPackages updates all installed packages to their latest versions
func (us *UpdateService) UpdateAllPackages() bool {
	// Get all installed packages
	localPackages := us.localPackages.GetData(true).Packages

	if len(localPackages) == 0 {
		us.output.Println("No packages are currently installed")
		return true
	}

	us.output.Printf("Found %d installed packages\n", len(localPackages))

	allSuccess := true
	successCount := 0
	failedCount := 0

	for _, pkg := range localPackages {
		us.output.Printf("Updating %s...\n", pkg.SourceID)

		// Use the service method (which can be mocked in tests)
		success := us.updatePackage(pkg.SourceID)
		if success {
			successCount++
			us.output.Printf("✓ Successfully updated %s\n", pkg.SourceID)
		} else {
			failedCount++
			us.output.Printf("✗ Failed to update %s\n", pkg.SourceID)
			allSuccess = false
		}
	}

	us.output.Printf("\nUpdate Summary:\n")
	us.output.Printf("  Successfully updated: %d\n", successCount)
	us.output.Printf("  Failed to update: %d\n", failedCount)

	return allSuccess
}
