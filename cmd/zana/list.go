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

// newListServiceFunc is a variable to allow test injection
var newListServiceFunc = func() *ListService {
	return &ListService{
		localPackages:  &defaultLocalPackagesProvider{},
		registry:       &defaultRegistryProvider{},
		updateChecker:  &defaultUpdateChecker{},
		fileDownloader: &defaultFileDownloader{},
	}
}

// NewListService creates a new ListService with default dependencies
func NewListService() *ListService {
	return newListServiceFunc()
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
	Use:     "list [filter...]",
	Aliases: []string{"ls"},
	Short:   "List packages",
	Long: `List packages based on the specified options.

By default, shows locally installed packages.
Use --all to show all available packages from the registry.
You can provide filter arguments to show only packages whose names start with the filter strings.`,
	Args: cobra.ArbitraryArgs,
	// Enable shell completion for package names
	ValidArgsFunction: packageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")
		service := newListService()

		if allFlag {
			service.ListAllPackages(args)
		} else {
			service.ListInstalledPackages(args)
		}
	},
}

func init() {
	listCmd.Flags().BoolP("all", "A", false, "List all available packages from the registry")
}

// newListService is a factory to allow test injection
var newListService = NewListService

// ListInstalledPackages lists locally installed packages
// If filters are provided, only shows packages whose names start with any of the filter strings
func (ls *ListService) ListInstalledPackages(filters []string) {
	// Ensure the registry is up to date so that update checks
	// for installed packages use the freshest available data.
	// Errors are ignored intentionally so that listing still works
	// even when the registry cannot be refreshed (e.g. offline).
	_ = ls.fileDownloader.DownloadAndUnzipRegistry()

	fmt.Printf("%s Locally Installed Packages\n", IconSummary())
	fmt.Println()

	localPackages := ls.localPackages.GetData(true).Packages

	if len(localPackages) == 0 {
		fmt.Println("No packages are currently installed.")
		fmt.Println("Use 'zana install <pkgId>' to install packages.")
		return
	}

	// Filter packages if filters are provided
	filteredPackages := localPackages
	if len(filters) > 0 {
		filteredPackages = []local_packages_parser.LocalPackageItem{}
		for _, pkg := range localPackages {
			packageName := getPackageNameFromSourceID(pkg.SourceID)
			packageNameLower := strings.ToLower(packageName)

			// Check if package name starts with any of the filter strings
			matches := false
			for _, filter := range filters {
				filterLower := strings.ToLower(filter)
				if strings.HasPrefix(packageNameLower, filterLower) {
					matches = true
					break
				}
			}

			if matches {
				filteredPackages = append(filteredPackages, pkg)
			}
		}
	}

	if len(filteredPackages) == 0 {
		if len(filters) > 0 {
			fmt.Printf("No installed packages found matching filters: %s\n", strings.Join(filters, ", "))
		} else {
			fmt.Println("No packages are currently installed.")
			fmt.Println("Use 'zana install <pkgId>' to install packages.")
		}
		return
	}

	fmt.Printf("Found %d installed packages", len(filteredPackages))
	if len(filters) > 0 {
		fmt.Printf(" matching filters: %s", strings.Join(filters, ", "))
	}
	fmt.Printf(":\n\n")

	// Group packages by provider
	packagesByProvider := make(map[string][]local_packages_parser.LocalPackageItem)
	for _, pkg := range filteredPackages {
		provider := getProviderFromSourceID(pkg.SourceID)
		packagesByProvider[provider] = append(packagesByProvider[provider], pkg)
	}

	// Display packages grouped by provider and count updates
	providers := []string{"npm", "golang", "pypi", "cargo", "github", "gitlab", "codeberg", "gem", "composer", "luarocks", "nuget", "opam", "openvsx", "generic"}
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
	fmt.Printf("%s Summary: %d of %d packages are up to date", IconSummary(), totalCount-updateCount, totalCount)
	if updateCount > 0 {
		fmt.Printf(", %d updates available", updateCount)
		fmt.Printf("\n%s Use 'zana update --all' to update all packages", IconLightbulb())
	}
	fmt.Println()
}

// ListAllPackages lists all available packages from the registry
// If filters are provided, only shows packages whose names start with any of the filter strings
func (ls *ListService) ListAllPackages(filters []string) {
	// Make sure we have an up-to-date registry before listing.
	// This mirrors the behavior of the TUI boot process which
	// refreshes the registry when the cache is too old.
	_ = ls.fileDownloader.DownloadAndUnzipRegistry()

	fmt.Println("📚 All Available Packages")
	fmt.Println()

	registry := ls.registry.GetData(true)

	if len(registry) == 0 {
		fmt.Println("No packages found in the registry.")
		fmt.Printf("%s Downloading registry...\n", IconRefresh())

		// Try to download the registry
		if err := ls.fileDownloader.DownloadAndUnzipRegistry(); err != nil {
			fmt.Printf("%s Failed to download registry: %v\n", IconCancel(), err)
			fmt.Printf("%s Use 'zana' (without flags) to download the registry manually.\n", IconLightbulb())
			return
		}

		fmt.Printf("%s Registry downloaded successfully!\n", IconCheckCircle())
		fmt.Println()

		// Try to get the registry data again
		registry = ls.registry.GetData(true)

		if len(registry) == 0 {
			fmt.Printf("%s Still no packages found after downloading registry.\n", IconCancel())
			return
		}
	}

	// Filter packages if filters are provided
	filteredRegistry := registry
	if len(filters) > 0 {
		filteredRegistry = []registry_parser.RegistryItem{}
		for _, pkg := range registry {
			packageName := getPackageNameFromSourceID(pkg.Source.ID)
			packageNameLower := strings.ToLower(packageName)

			// Check if package name starts with any of the filter strings
			matches := false
			for _, filter := range filters {
				filterLower := strings.ToLower(filter)
				if strings.HasPrefix(packageNameLower, filterLower) {
					matches = true
					break
				}
			}

			if matches {
				filteredRegistry = append(filteredRegistry, pkg)
			}
		}
	}

	if len(filteredRegistry) == 0 {
		if len(filters) > 0 {
			fmt.Printf("No packages found in the registry matching filters: %s\n", strings.Join(filters, ", "))
		} else {
			fmt.Println("No packages found in the registry.")
		}
		return
	}

	fmt.Printf("Found %d packages in the registry", len(filteredRegistry))
	if len(filters) > 0 {
		fmt.Printf(" matching filters: %s", strings.Join(filters, ", "))
	}
	fmt.Printf(":\n\n")

	// Get installed packages to check status
	installedPackages := ls.localPackages.GetData(false).Packages
	installedMap := make(map[string]string) // sourceID -> version
	for _, pkg := range installedPackages {
		installedMap[pkg.SourceID] = pkg.Version
	}

	// Group packages by provider
	packagesByProvider := make(map[string][]registry_parser.RegistryItem)
	for _, pkg := range filteredRegistry {
		provider := getProviderFromSourceID(pkg.Source.ID)
		packagesByProvider[provider] = append(packagesByProvider[provider], pkg)
	}

	// Display packages grouped by provider
	providers := []string{"npm", "golang", "pypi", "cargo", "github", "gitlab", "codeberg", "gem", "composer", "luarocks", "nuget", "opam", "openvsx", "generic"}
	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("🔹 %s Packages (%d):\n", strings.ToUpper(provider), len(packages))
			for _, pkg := range packages {
				packageName := getPackageNameFromSourceID(pkg.Source.ID)
				installedVersion, isInstalled := installedMap[pkg.Source.ID]

				// Build status indicators
				statusIndicators := []string{}
				if isInstalled {
					statusIndicators = append(statusIndicators, IconCheckCircle()+" Installed")
					// Check if update is available
					updateInfo, hasUpdate := ls.checkUpdateAvailability(pkg.Source.ID, installedVersion)
					if hasUpdate {
						statusIndicators = append(statusIndicators, updateInfo)
					} else {
						statusIndicators = append(statusIndicators, IconCheckCircle()+" Up to date")
					}
				}

				// Display package info
				fmt.Printf("   %s %s (v%s)", getProviderIcon(provider), packageName, pkg.Version)
				if len(statusIndicators) > 0 {
					fmt.Printf(" %s", strings.Join(statusIndicators, " | "))
				}
				fmt.Println()

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
		return fmt.Sprintf("%s Update available: v%s", IconRefresh(), latestVersion), true
	}
	updateAvailable, _ := ls.updateChecker.CheckIfUpdateIsAvailable(currentVersion, latestVersion)
	if updateAvailable {
		return fmt.Sprintf("%s Update available: v%s", IconRefresh(), latestVersion), true
	}
	return IconCheckCircle() + " Up to date", false
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
	parser := registry_parser.NewDefaultRegistryParser()
	return parser.GetData(force)
}

func (d *defaultRegistryProvider) GetLatestVersion(sourceID string) string {
	parser := registry_parser.NewDefaultRegistryParser()
	return parser.GetLatestVersion(sourceID)
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
	service.ListInstalledPackages(nil)
}

func listAllPackages() {
	service := NewListService()
	service.ListAllPackages(nil)
}

func checkUpdateAvailability(sourceID, currentVersion string) (string, bool) {
	service := NewListService()
	return service.checkUpdateAvailability(sourceID, currentVersion)
}

func getProviderFromSourceID(sourceID string) string {
	// Support both legacy (pkg:provider/pkg) and new (provider:pkg) formats
	if strings.HasPrefix(sourceID, "npm:") {
		return "npm"
	} else if strings.HasPrefix(sourceID, "golang:") {
		return "golang"
	} else if strings.HasPrefix(sourceID, "pypi:") {
		return "pypi"
	} else if strings.HasPrefix(sourceID, "cargo:") {
		return "cargo"
	} else if strings.HasPrefix(sourceID, "github:") {
		return "github"
	} else if strings.HasPrefix(sourceID, "gitlab:") {
		return "gitlab"
	} else if strings.HasPrefix(sourceID, "codeberg:") {
		return "codeberg"
	}
	// Legacy format support
	if strings.HasPrefix(sourceID, "pkg:npm/") {
		return "npm"
	} else if strings.HasPrefix(sourceID, "pkg:golang/") {
		return "golang"
	} else if strings.HasPrefix(sourceID, "pkg:pypi/") {
		return "pypi"
	} else if strings.HasPrefix(sourceID, "pkg:cargo/") {
		return "cargo"
	} else if strings.HasPrefix(sourceID, "pkg:github/") {
		return "github"
	} else if strings.HasPrefix(sourceID, "pkg:gitlab/") {
		return "gitlab"
	} else if strings.HasPrefix(sourceID, "pkg:codeberg/") {
		return "codeberg"
	}
	return "unknown"
}

func getPackageNameFromSourceID(sourceID string) string {
	// Support new format: provider:pkg
	if strings.Contains(sourceID, ":") && !strings.HasPrefix(sourceID, "pkg:") {
		parts := strings.SplitN(sourceID, ":", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	// Legacy format: pkg:provider/pkg
	withoutPrefix := strings.TrimPrefix(sourceID, "pkg:")
	parts := strings.SplitN(withoutPrefix, "/", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return sourceID
}

func getProviderIcon(provider string) string {
	switch provider {
	case "npm":
		return IconNPM()
	case "golang":
		return IconGolang()
	case "pypi":
		return IconPython()
	case "cargo":
		return IconCargo()
	case "github":
		return IconGitHub()
	case "gitlab":
		return IconGitLab()
	case "codeberg":
		return IconCodeberg()
	case "gem":
		return IconGem()
	case "composer":
		return IconComposer()
	case "luarocks":
		return IconLuaRocks()
	case "nuget":
		return IconNuGet()
	case "opam":
		return IconOpam()
	case "openvsx":
		return IconOpenVSX()
	case "generic":
		return IconGeneric()
	default:
		return IconGeneric()
	}
}
