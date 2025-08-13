package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

// UpdateService handles update operations with dependency injection
type UpdateService struct {
	localPackages  LocalPackagesProvider
	packageManager PackageManager
	output         OutputWriter
}

// PackageManager defines the interface for package management operations
type PackageManager interface {
	Update(sourceID string) bool
	IsSupportedProvider(provider string) bool
	AvailableProviders() []string
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
		localPackages:  &defaultLocalPackagesProvider{},
		packageManager: &defaultPackageManager{},
		output:         &DefaultOutputWriter{},
	}
}

// NewUpdateServiceWithDependencies creates a new UpdateService with custom dependencies
func NewUpdateServiceWithDependencies(
	localPackages LocalPackagesProvider,
	packageManager PackageManager,
	output OutputWriter,
) *UpdateService {
	return &UpdateService{
		localPackages:  localPackages,
		packageManager: packageManager,
		output:         output,
	}
}

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update packages to their latest versions",
	Long: `Update packages to their latest versions.

Examples:
  zana update pkg:npm/eslint
  zana update pkg:golang/golang.org/x/tools/gopls
  zana update pkg:pypi/black
  zana update pkg:cargo/ripgrep
  zana update --all (update all installed packages)`,
	Args: cobra.MaximumNArgs(1),
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

		// Check if package ID is provided
		if len(args) == 0 {
			service := newUpdateService()
			service.output.Println("Error: Please provide a package ID or use --all flag")
			cmd.Help()
			return
		}

		pkgId := args[0]

		// Validate package ID format
		if !strings.HasPrefix(pkgId, "pkg:") {
			service := newUpdateService()
			service.output.Printf("Error: Invalid package ID format. Must start with 'pkg:'\n")
			return
		}

		// Parse provider from package ID
		parts := strings.Split(strings.TrimPrefix(pkgId, "pkg:"), "/")
		if len(parts) < 2 {
			service := newUpdateService()
			service.output.Printf("Error: Invalid package ID format. Expected 'pkg:provider/package-name'\n")
			return
		}

		provider := parts[0]
		service := newUpdateService()
		if !service.packageManager.IsSupportedProvider(provider) {
			service.output.Printf("Error: Unsupported provider '%s'. Supported providers: %s\n", provider, strings.Join(service.packageManager.AvailableProviders(), ", "))
			return
		}

		service.output.Printf("Updating %s to latest version...\n", pkgId)

		// Update the package
		success := service.packageManager.Update(pkgId)
		if success {
			service.output.Printf("Successfully updated %s\n", pkgId)
		} else {
			service.output.Printf("Failed to update %s\n", pkgId)
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

		success := us.packageManager.Update(pkg.SourceID)
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

// Default implementations for backward compatibility
type defaultPackageManager struct{}

func (d *defaultPackageManager) Update(sourceID string) bool {
	return providers.Update(sourceID)
}

func (d *defaultPackageManager) IsSupportedProvider(provider string) bool {
	return providers.IsSupportedProvider(provider)
}

func (d *defaultPackageManager) AvailableProviders() []string {
	return providers.AvailableProviders
}

// Legacy function for backward compatibility
func updateAllPackages() bool {
	service := NewUpdateService()
	return service.UpdateAllPackages()
}
