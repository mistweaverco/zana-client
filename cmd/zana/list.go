package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/spf13/cobra"
)

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

		if allFlag {
			listAllPackages()
		} else {
			listInstalledPackages()
		}
	},
}

func init() {
	listCmd.Flags().BoolP("all", "A", false, "List all available packages from the registry")
}

func listInstalledPackages() {
	fmt.Println("📦 Locally Installed Packages")
	fmt.Println()

	localPackages := local_packages_parser.GetData(true).Packages

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
	providers := []string{"npm", "golang", "pypi"}
	updateCount := 0
	totalCount := 0

	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("🔹 %s Packages:\n", strings.ToUpper(provider))
			for _, pkg := range packages {
				packageName := getPackageNameFromSourceID(pkg.SourceID)
				updateInfo := checkUpdateAvailability(pkg.SourceID, pkg.Version)
				fmt.Printf("   %s %s (v%s) %s\n", getProviderIcon(provider), packageName, pkg.Version, updateInfo)

				totalCount++
				if strings.Contains(updateInfo, "🔄 Update available") {
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

func listAllPackages() {
	fmt.Println("📚 All Available Packages")
	fmt.Println()

	registry := registry_parser.GetData(true)

	if len(registry) == 0 {
		fmt.Println("No packages found in the registry.")
		fmt.Println("🔄 Downloading registry...")

		// Try to download the registry
		if err := files.DownloadAndUnzipRegistry(); err != nil {
			fmt.Printf("❌ Failed to download registry: %v\n", err)
			fmt.Println("💡 Use 'zana' (without flags) to download the registry manually.")
			return
		}

		fmt.Println("✅ Registry downloaded successfully!")
		fmt.Println()

		// Try to get the registry data again
		registry = registry_parser.GetData(true)

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
	providers := []string{"npm", "golang", "pypi"}
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

func getProviderFromSourceID(sourceID string) string {
	if strings.HasPrefix(sourceID, "pkg:npm/") {
		return "npm"
	} else if strings.HasPrefix(sourceID, "pkg:golang/") {
		return "golang"
	} else if strings.HasPrefix(sourceID, "pkg:pypi/") {
		return "pypi"
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

func checkUpdateAvailability(sourceID, currentVersion string) string {
	// Get the latest version from the registry
	latestVersion := registry_parser.GetLatestVersion(sourceID)

	if latestVersion == "" {
		return "" // No registry info available
	}

	// Check if update is available
	if latestVersion != currentVersion {
		return fmt.Sprintf("🔄 Update available: v%s", latestVersion)
	}

	return "✅ Up to date"
}

func getProviderIcon(provider string) string {
	switch provider {
	case "npm":
		return "📦"
	case "golang":
		return "🐹"
	case "pypi":
		return "🐍"
	default:
		return "📋"
	}
}
