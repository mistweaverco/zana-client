package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/spf13/cobra"
)

// ListService handles listing operations with dependency injection
type ListService struct {
	localPackages  LocalPackagesProvider
	registry       RegistryProvider
	updateChecker  UpdateChecker
	fileDownloader FileDownloader
}

// NewListService creates a new ListService with default dependencies
func NewListService() *ListService {
	return &ListService{
		localPackages:  &defaultLocalPackagesProvider{},
		registry:       &defaultRegistryProvider{},
		updateChecker:  &defaultUpdateChecker{},
		fileDownloader: &defaultFileDownloader{},
	}
}

// NewListServiceWithDependencies creates a new ListService with custom dependencies
func NewListServiceWithDependencies(
	localPackages LocalPackagesProvider,
	registry RegistryProvider,
	updateChecker UpdateChecker,
	fileDownloader FileDownloader,
) *ListService {
	return &ListService{
		localPackages:  localPackages,
		registry:       registry,
		updateChecker:  updateChecker,
		fileDownloader: fileDownloader,
	}
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List packages",
	Long: `List packages based on the specified options.

By default, shows locally installed packages.
Use --all to show all available packages from the registry.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")
		service := newListService()

		if allFlag {
			service.ListAllPackages()
		} else {
			service.ListInstalledPackages()
		}
	},
}

func init() {
	listCmd.Flags().BoolP("all", "A", false, "List all available packages from the registry")
}

// newListService is a factory to allow test injection
var newListService = NewListService

// ListInstalledPackages lists locally installed packages
func (ls *ListService) ListInstalledPackages() {
	fmt.Println("📦 Locally Installed Packages")
	fmt.Println()

	localPackages := ls.localPackages.GetData(true).Packages

	if len(localPackages) == 0 {
		fmt.Println("No packages are currently installed.")
		fmt.Println("Use 'zana install <pkgId>' to install packages.")
		return
	}

	fmt.Printf("Found %d installed packages:\n\n", len(localPackages))

	// Group packages by provider
	packagesByProvider := make(map[string][]local_packages_parser.LocalPackageItem)
	for _, pkg := range localPackages {
		provider := getProviderFromSourceID(pkg.SourceID)
		packagesByProvider[provider] = append(packagesByProvider[provider], pkg)
	}

	// Display packages grouped by provider and count updates
	providers := []string{"npm", "golang", "pypi", "cargo"}
	updateCount := 0
	totalCount := 0

	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("🔹 %s Packages:\n", strings.ToUpper(provider))
			for _, pkg := range packages {
				packageName := getPackageNameFromSourceID(pkg.SourceID)
				updateInfo, hasUpdate := ls.checkUpdateAvailability(pkg.SourceID, pkg.Version)
				fmt.Printf("   %s %s (v%s) %s\n", getProviderIcon(provider), packageName, pkg.Version, updateInfo)

				totalCount++
				if hasUpdate {
					updateCount++
				}
			}
			fmt.Println()
		}
	}

	// Show summary
	fmt.Printf("📊 Summary: %d of %d packages are up to date", totalCount-updateCount, totalCount)
	if updateCount > 0 {
		fmt.Printf(", %d updates available", updateCount)
		fmt.Printf("\n💡 Use 'zana update --all' to update all packages")
	}
	fmt.Println()
}

// ListAllPackages lists all available packages from the registry
func (ls *ListService) ListAllPackages() {
	fmt.Println("📚 All Available Packages")
	fmt.Println()

	registry := ls.registry.GetData(true)

	if len(registry) == 0 {
		fmt.Println("No packages found in the registry.")
		fmt.Println("🔄 Downloading registry...")

		// Try to download the registry
		if err := ls.fileDownloader.DownloadAndUnzipRegistry(); err != nil {
			fmt.Printf("❌ Failed to download registry: %v\n", err)
			fmt.Println("💡 Use 'zana' (without flags) to download the registry manually.")
			return
		}

		fmt.Println("✅ Registry downloaded successfully!")
		fmt.Println()

		// Try to get the registry data again
		registry = ls.registry.GetData(true)

		if len(registry) == 0 {
			fmt.Println("❌ Still no packages found after downloading registry.")
			return
		}
	}

	fmt.Printf("Found %d packages in the registry:\n\n", len(registry))

	// Group packages by provider
	packagesByProvider := make(map[string][]registry_parser.RegistryItem)
	for _, pkg := range registry {
		provider := getProviderFromSourceID(pkg.Source.ID)
		packagesByProvider[provider] = append(packagesByProvider[provider], pkg)
	}

	// Display packages grouped by provider
	providers := []string{"npm", "golang", "pypi", "cargo"}
	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("🔹 %s Packages (%d):\n", strings.ToUpper(provider), len(packages))
			for _, pkg := range packages {
				packageName := getPackageNameFromSourceID(pkg.Source.ID)
				fmt.Printf("   %s %s (v%s)\n", getProviderIcon(provider), packageName, pkg.Version)
				if pkg.Description != "" {
					fmt.Printf("      %s\n", pkg.Description)
				}
			}
			fmt.Println()
		}
	}
}

// checkUpdateAvailability checks if an update is available for a package
func (ls *ListService) checkUpdateAvailability(sourceID, currentVersion string) (string, bool) {
	latestVersion := ls.registry.GetLatestVersion(sourceID)
	if latestVersion == "" {
		return "", false // No registry info available
	}
	// If local version is unknown or set to "latest", always show update to the concrete remote version
	if currentVersion == "" || currentVersion == "latest" {
		return fmt.Sprintf("🔄 Update available: v%s", latestVersion), true
	}
	updateAvailable, _ := ls.updateChecker.CheckIfUpdateIsAvailable(currentVersion, latestVersion)
	if updateAvailable {
		return fmt.Sprintf("🔄 Update available: v%s", latestVersion), true
	}
	return "✅ Up to date", false
}

// Default implementations for backward compatibility
type defaultLocalPackagesProvider struct{}
type defaultRegistryProvider struct{}
type defaultUpdateChecker struct{}
type defaultFileDownloader struct{}

func (d *defaultLocalPackagesProvider) GetData(force bool) local_packages_parser.LocalPackageRoot {
	return local_packages_parser.GetData(force)
}

func (d *defaultRegistryProvider) GetData(force bool) []registry_parser.RegistryItem {
	return registry_parser.GetData(force)
}

func (d *defaultRegistryProvider) GetLatestVersion(sourceID string) string {
	return registry_parser.GetLatestVersion(sourceID)
}

func (d *defaultUpdateChecker) CheckIfUpdateIsAvailable(currentVersion, latestVersion string) (bool, string) {
	return providers.CheckIfUpdateIsAvailable(currentVersion, latestVersion)
}

// indirection for testability
var downloadAndUnzipRegistryFn = files.DownloadAndUnzipRegistry

func (d *defaultFileDownloader) DownloadAndUnzipRegistry() error {
	return downloadAndUnzipRegistryFn()
}

// Legacy functions for backward compatibility
func listInstalledPackages() {
	service := NewListService()
	service.ListInstalledPackages()
}

func listAllPackages() {
	service := NewListService()
	service.ListAllPackages()
}

func checkUpdateAvailability(sourceID, currentVersion string) (string, bool) {
	service := NewListService()
	return service.checkUpdateAvailability(sourceID, currentVersion)
}

func getProviderFromSourceID(sourceID string) string {
	if strings.HasPrefix(sourceID, "pkg:npm/") {
		return "npm"
	} else if strings.HasPrefix(sourceID, "pkg:golang/") {
		return "golang"
	} else if strings.HasPrefix(sourceID, "pkg:pypi/") {
		return "pypi"
	} else if strings.HasPrefix(sourceID, "pkg:cargo/") {
		return "cargo"
	}
	return "unknown"
}

func getPackageNameFromSourceID(sourceID string) string {
	// Remove the "pkg:" prefix
	withoutPrefix := strings.TrimPrefix(sourceID, "pkg:")

	// Split by "/" to separate provider from package name
	parts := strings.SplitN(withoutPrefix, "/", 2)
	if len(parts) >= 2 {
		return parts[1] // Return everything after the first "/"
	}
	return sourceID
}

func getProviderIcon(provider string) string {
	switch provider {
	case "npm":
		return "📦"
	case "golang":
		return "🐹"
	case "pypi":
		return "🐍"
	case "cargo":
		return "🦀"
	default:
		return "📋"
	}
}
