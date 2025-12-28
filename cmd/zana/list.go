package zana

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/term"
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

	localPackages := ls.localPackages.GetData(true).Packages

	// Filter packages if filters are provided
	filteredPackages := localPackages
	if len(filters) > 0 {
		filteredPackages = []local_packages_parser.LocalPackageItem{}
		parser := newRegistryParser()
		for _, pkg := range localPackages {
			packageName := getPackageNameFromSourceID(pkg.SourceID)
			packageNameLower := strings.ToLower(packageName)
			sourceIDLower := strings.ToLower(pkg.SourceID)

			// Check if package name, full sourceID, or aliases start with any of the filter strings
			matches := false
			for _, filter := range filters {
				filterLower := strings.ToLower(filter)
				// Match against full sourceID (provider:package-id) or just package name
				if strings.HasPrefix(sourceIDLower, filterLower) || strings.HasPrefix(packageNameLower, filterLower) {
					matches = true
					break
				}

				// Also check aliases from registry
				registryItem := parser.GetBySourceId(pkg.SourceID)
				if registryItem.Source.ID != "" {
					for _, alias := range registryItem.Aliases {
						aliasLower := strings.ToLower(alias)
						if strings.HasPrefix(aliasLower, filterLower) {
							matches = true
							break
						}
					}
					if matches {
						break
					}
				}
			}

			if matches {
				filteredPackages = append(filteredPackages, pkg)
			}
		}
	}

	// Output based on mode
	if ShouldUseJSONOutput() {
		ls.listInstalledPackagesJSON(filteredPackages, filters)
	} else if ShouldUsePlainOutput() {
		ls.listInstalledPackagesPlain(filteredPackages, filters)
	} else {
		ls.listInstalledPackagesRich(filteredPackages, filters)
	}
}

// listInstalledPackagesRich lists installed packages with rich formatting using markdown tables
func (ls *ListService) listInstalledPackagesRich(filteredPackages []local_packages_parser.LocalPackageItem, filters []string) {
	var markdown strings.Builder

	markdown.WriteString(fmt.Sprintf("# %s Locally Installed Packages\n\n", IconSummaryPlain()))

	if len(filteredPackages) == 0 {
		if len(filters) > 0 {
			markdown.WriteString(fmt.Sprintf("No installed packages found matching filters: %s\n", strings.Join(filters, ", ")))
		} else {
			markdown.WriteString("No packages are currently installed.\n\n")
			markdown.WriteString("Use `zana install <pkgId>` to install packages.\n")
		}
		ls.renderMarkdown(markdown.String())
		return
	}

	markdown.WriteString(fmt.Sprintf("Found **%d** installed packages", len(filteredPackages)))
	if len(filters) > 0 {
		markdown.WriteString(fmt.Sprintf(" matching filters: %s", strings.Join(filters, ", ")))
	}
	markdown.WriteString("\n\n")

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
			markdown.WriteString(fmt.Sprintf("## %s Packages\n\n", strings.ToUpper(provider)))
			markdown.WriteString("| Package ID | Version | Status |\n")
			markdown.WriteString("|------------|---------|--------|\n")

			for _, pkg := range packages {
				updateInfo, hasUpdate := ls.checkUpdateAvailability(pkg.SourceID, pkg.Version)
				// Clean up update info for table display (remove icons, keep text)
				statusText := strings.ReplaceAll(updateInfo, IconRefresh(), "")
				statusText = strings.ReplaceAll(statusText, IconCheckCircle(), "")
				statusText = strings.TrimSpace(statusText)
				if statusText == "" {
					if hasUpdate {
						// Use ANSI yellow color for update status
						statusText = "\033[33mUpdate available\033[0m"
					} else {
						statusText = "Up to date"
					}
				} else if hasUpdate {
					// Wrap existing text in yellow ANSI codes
					statusText = fmt.Sprintf("\033[33m%s\033[0m", statusText)
				}

				markdown.WriteString(fmt.Sprintf("| %s | %s | %s |\n", pkg.SourceID, pkg.Version, statusText))

				totalCount++
				if hasUpdate {
					updateCount++
				}
			}
			markdown.WriteString("\n")
		}
	}

	// Show summary
	markdown.WriteString("### Summary\n\n")
	markdown.WriteString(fmt.Sprintf("- **%d** of **%d** packages are up to date", totalCount-updateCount, totalCount))
	if updateCount > 0 {
		markdown.WriteString(fmt.Sprintf("\n- **%d** updates available", updateCount))
		markdown.WriteString(fmt.Sprintf("\n- %s Use `zana update --all` to update all packages", IconLightbulbPlain()))
	}
	markdown.WriteString("\n")

	ls.renderMarkdown(markdown.String())
}

// listInstalledPackagesPlain lists installed packages in plain text format
func (ls *ListService) listInstalledPackagesPlain(filteredPackages []local_packages_parser.LocalPackageItem, filters []string) {
	fmt.Printf("%s Locally Installed Packages\n\n", IconSummary())

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

	providers := []string{"npm", "golang", "pypi", "cargo", "github", "gitlab", "codeberg", "gem", "composer", "luarocks", "nuget", "opam", "openvsx", "generic"}
	updateCount := 0
	totalCount := 0

	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("%s %s Packages:\n", IconDiamond(), strings.ToUpper(provider))
			for _, pkg := range packages {
				updateInfo, hasUpdate := ls.checkUpdateAvailability(pkg.SourceID, pkg.Version)
				fmt.Printf("   %s %s (v%s) %s\n", getProviderIcon(provider), pkg.SourceID, pkg.Version, updateInfo)
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

// listInstalledPackagesJSON lists installed packages in JSON format
func (ls *ListService) listInstalledPackagesJSON(filteredPackages []local_packages_parser.LocalPackageItem, filters []string) {
	result := make(map[string]any)
	result["type"] = "installed"
	if len(filters) > 0 {
		result["filters"] = filters
	}

	if len(filteredPackages) == 0 {
		result["count"] = 0
		result["packages"] = []any{}
		PrintJSON(result)
		return
	}

	packagesData := make([]map[string]any, 0, len(filteredPackages))
	updateCount := 0

	for _, pkg := range filteredPackages {
		packageName := getPackageNameFromSourceID(pkg.SourceID)
		provider := getProviderFromSourceID(pkg.SourceID)
		_, hasUpdate := ls.checkUpdateAvailability(pkg.SourceID, pkg.Version)

		pkgData := map[string]any{
			"source_id":  pkg.SourceID,
			"name":       packageName,
			"provider":   provider,
			"version":    pkg.Version,
			"has_update": hasUpdate,
		}
		packagesData = append(packagesData, pkgData)

		if hasUpdate {
			updateCount++
		}
	}

	result["count"] = len(filteredPackages)
	result["packages"] = packagesData
	result["updates_available"] = updateCount
	PrintJSON(result)
}

// ListAllPackages lists all available packages from the registry
// If filters are provided, only shows packages whose names start with any of the filter strings
func (ls *ListService) ListAllPackages(filters []string) {
	// Make sure we have an up-to-date registry before listing.
	// This mirrors the behavior of the TUI boot process which
	// refreshes the registry when the cache is too old.
	_ = ls.fileDownloader.DownloadAndUnzipRegistry()

	registry := ls.registry.GetData(true)

	if len(registry) == 0 {
		if !ShouldUseJSONOutput() {
			if ShouldUsePlainOutput() {
				fmt.Println("No packages found in the registry.")
				fmt.Println("[~] Downloading registry...")
			} else {
				fmt.Println("No packages found in the registry.")
				fmt.Printf("%s Downloading registry...\n", IconRefresh())
			}
		}

		// Try to download the registry
		if err := ls.fileDownloader.DownloadAndUnzipRegistry(); err != nil {
			if ShouldUseJSONOutput() {
				result := map[string]any{
					"type":    "all",
					"error":   "failed to download registry",
					"details": err.Error(),
				}
				PrintJSON(result)
			} else if ShouldUsePlainOutput() {
				fmt.Printf("[✗] Failed to download registry: %v\n", err)
				fmt.Println("[*] Use 'zana' (without flags) to download the registry manually.")
			} else {
				fmt.Printf("%s Failed to download registry: %v\n", IconCancel(), err)
				fmt.Printf("%s Use 'zana' (without flags) to download the registry manually.\n", IconLightbulb())
			}
			return
		}

		if !ShouldUseJSONOutput() {
			if ShouldUsePlainOutput() {
				fmt.Println("[✓] Registry downloaded successfully!")
				fmt.Println()
			} else {
				fmt.Printf("%s Registry downloaded successfully!\n", IconCheckCircle())
				fmt.Println()
			}
		}

		// Try to get the registry data again
		registry = ls.registry.GetData(true)

		if len(registry) == 0 {
			if ShouldUseJSONOutput() {
				result := map[string]any{
					"type":  "all",
					"error": "still no packages found after downloading registry",
				}
				PrintJSON(result)
			} else if ShouldUsePlainOutput() {
				fmt.Println("[✗] Still no packages found after downloading registry.")
			} else {
				fmt.Printf("%s Still no packages found after downloading registry.\n", IconCancel())
			}
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
			sourceIDLower := strings.ToLower(pkg.Source.ID)

			// Check if package name, full sourceID, or aliases start with any of the filter strings
			matches := false
			for _, filter := range filters {
				filterLower := strings.ToLower(filter)
				// Match against full sourceID (provider:package-id) or just package name
				if strings.HasPrefix(sourceIDLower, filterLower) || strings.HasPrefix(packageNameLower, filterLower) {
					matches = true
					break
				}

				// Also check aliases
				for _, alias := range pkg.Aliases {
					aliasLower := strings.ToLower(alias)
					if strings.HasPrefix(aliasLower, filterLower) {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}

			if matches {
				filteredRegistry = append(filteredRegistry, pkg)
			}
		}
	}

	// Output based on mode
	if ShouldUseJSONOutput() {
		ls.listAllPackagesJSON(filteredRegistry, filters)
	} else if ShouldUsePlainOutput() {
		ls.listAllPackagesPlain(filteredRegistry, filters)
	} else {
		ls.listAllPackagesRich(filteredRegistry, filters)
	}
}

// listAllPackagesRich lists all packages with rich formatting using markdown tables
func (ls *ListService) listAllPackagesRich(filteredRegistry []registry_parser.RegistryItem, filters []string) {
	var markdown strings.Builder

	markdown.WriteString(fmt.Sprintf("## %s All Available Packages\n\n", IconBookPlain()))

	if len(filteredRegistry) == 0 {
		if len(filters) > 0 {
			markdown.WriteString(fmt.Sprintf("No packages found in the registry matching filters: %s\n", strings.Join(filters, ", ")))
		} else {
			markdown.WriteString("No packages found in the registry.\n")
		}
		ls.renderMarkdown(markdown.String())
		return
	}

	markdown.WriteString(fmt.Sprintf("Found **%d** packages in the registry", len(filteredRegistry)))
	if len(filters) > 0 {
		markdown.WriteString(fmt.Sprintf(" matching filters: %s", strings.Join(filters, ", ")))
	}
	markdown.WriteString("\n\n")

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
			markdown.WriteString(fmt.Sprintf("### %s %s Packages (%d)\n\n", IconDiamondPlain(), strings.ToUpper(provider), len(packages)))
			markdown.WriteString("| Package ID | Version | Status | Description |\n")
			markdown.WriteString("|------------|---------|--------|-------------|\n")

			for _, pkg := range packages {
				installedVersion, isInstalled := installedMap[pkg.Source.ID]

				// Build status text
				statusText := ""
				if isInstalled {
					updateInfo, hasUpdate := ls.checkUpdateAvailability(pkg.Source.ID, installedVersion)
					if hasUpdate {
						// Clean up update info for table display
						statusText = strings.ReplaceAll(updateInfo, IconRefresh(), "")
						statusText = strings.TrimSpace(statusText)
						if statusText == "" {
							// Use ANSI yellow color directly for update status
							statusText = fmt.Sprintf("\033[33m%s Update available\033[0m", IconRefreshPlain())
						} else {
							// Wrap existing text in yellow ANSI codes
							statusText = fmt.Sprintf("\033[33m%s\033[0m", statusText)
						}
					} else {
						statusText = fmt.Sprintf("%s Installed, up to date", IconCheckCirclePlain())
					}
				} else {
					statusText = fmt.Sprintf("%s Not installed", IconEmptyPlain())
				}

				// Escape pipe characters in description for markdown table
				description := pkg.Description
				if description != "" {
					description = strings.ReplaceAll(description, "|", "\\|")
				} else {
					description = "—"
				}

				markdown.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", pkg.Source.ID, pkg.Version, statusText, description))
			}
			markdown.WriteString("\n")
		}
	}

	ls.renderMarkdown(markdown.String())
}

// renderMarkdown renders markdown content using glamour
func (ls *ListService) renderMarkdown(markdown string) {
	// Get terminal width, default to 80 if not available
	width := 80
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		width = w
	}

	// Create a renderer with terminal width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// Fallback to plain render
		rendered, renderErr := glamour.Render(markdown, "dark")
		if renderErr != nil {
			fmt.Print(markdown)
			return
		}
		fmt.Print(rendered)
		return
	}

	rendered, err := r.Render(markdown)
	if err != nil {
		// Fallback to plain text if rendering fails
		fmt.Print(markdown)
		return
	}
	fmt.Print(rendered)
}

// listAllPackagesPlain lists all packages in plain text format
func (ls *ListService) listAllPackagesPlain(filteredRegistry []registry_parser.RegistryItem, filters []string) {
	fmt.Printf("%s All Available Packages\n\n", IconBook())

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

	providers := []string{"npm", "golang", "pypi", "cargo", "github", "gitlab", "codeberg", "gem", "composer", "luarocks", "nuget", "opam", "openvsx", "generic"}
	for _, provider := range providers {
		if packages, exists := packagesByProvider[provider]; exists {
			fmt.Printf("%s %s Packages (%d):\n", IconDiamond(), strings.ToUpper(provider), len(packages))
			for _, pkg := range packages {
				fmt.Printf("   %s %s (v%s)", getProviderIcon(provider), pkg.Source.ID, pkg.Version)
				if pkg.Description != "" {
					fmt.Printf("\n      %s", pkg.Description)
				}
				fmt.Println()
			}
			fmt.Println()
		}
	}
}

// listAllPackagesJSON lists all packages in JSON format
func (ls *ListService) listAllPackagesJSON(filteredRegistry []registry_parser.RegistryItem, filters []string) {
	result := make(map[string]any)
	result["type"] = "all"
	if len(filters) > 0 {
		result["filters"] = filters
	}

	if len(filteredRegistry) == 0 {
		result["count"] = 0
		result["packages"] = []any{}
		PrintJSON(result)
		return
	}

	// Get installed packages to check status
	installedPackages := ls.localPackages.GetData(false).Packages
	installedMap := make(map[string]string) // sourceID -> version
	for _, pkg := range installedPackages {
		installedMap[pkg.SourceID] = pkg.Version
	}

	packagesData := make([]map[string]any, 0, len(filteredRegistry))
	for _, pkg := range filteredRegistry {
		packageName := getPackageNameFromSourceID(pkg.Source.ID)
		provider := getProviderFromSourceID(pkg.Source.ID)
		installedVersion, isInstalled := installedMap[pkg.Source.ID]

		pkgData := map[string]any{
			"source_id": pkg.Source.ID,
			"name":      packageName,
			"provider":  provider,
			"version":   pkg.Version,
			"installed": isInstalled,
		}

		if isInstalled {
			pkgData["installed_version"] = installedVersion
			_, hasUpdate := ls.checkUpdateAvailability(pkg.Source.ID, installedVersion)
			pkgData["has_update"] = hasUpdate
		}

		if pkg.Description != "" {
			pkgData["description"] = pkg.Description
		}

		packagesData = append(packagesData, pkgData)
	}

	result["count"] = len(filteredRegistry)
	result["packages"] = packagesData
	PrintJSON(result)
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
