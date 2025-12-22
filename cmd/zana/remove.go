package zana

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh/spinner"
	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <pkgId> [pkgId...]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove one or more packages",
	Long: `Remove one or more packages from supported providers.

Supported package ID formats:
  npm:@prisma/language-server
  golang:golang.org/x/tools/gopls
  pypi:black
  cargo:ripgrep
  github:user/repo
  gitlab:group/subgroup/project
  codeberg:user/repo

Examples:
  zana remove npm:@prisma/language-server
  zana rm golang:golang.org/x/tools/gopls npm:eslint
  zana delete pypi:black cargo:ripgrep
  zana remove npm:prettier golang:golang.org/x/tools/gopls
  zana remove github:sharkdp/bat
  zana remove gitlab:group/subgroup/myproject`,
	Args: cobra.MinimumNArgs(1),
	// Enable shell completion for installed package IDs only.
	ValidArgsFunction: installedPackageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		packages := args

		// Process all packages
		// Use slices instead of fixed-size arrays to handle multi-select results
		internalIDs := make([]string, 0, len(packages))
		displayIDs := make([]string, 0, len(packages))

		for _, userPkgID := range packages {
			// Parse package ID and version from the user-facing ID
			baseID, _ := parsePackageIDAndVersion(userPkgID)

			var internalID string
			var displayID string

			// Check if this is a package name without provider
			if !strings.Contains(baseID, ":") && !strings.HasPrefix(baseID, "pkg:") {
				// Package name without provider - search installed packages and prompt user
				matches := findInstalledPackagesByName(baseID)
				if len(matches) == 0 {
					fmt.Printf("%s No installed packages found matching '%s'\n", IconClose(), baseID)
					return
				}

				// Always show confirmation for partial names (user didn't provide full provider:package-id)
				selectedSourceIDs, err := promptForProviderSelection(baseID, matches, false, "remove")
				if err != nil {
					fmt.Printf("%s Error selecting provider for '%s': %v\n", IconClose(), baseID, err)
					return
				}

				// Process all selected packages
				for _, selectedSourceID := range selectedSourceIDs {
					internalIDs = append(internalIDs, selectedSourceID)
					displayIDs = append(displayIDs, selectedSourceID)
				}
				continue // Skip the single package processing below
			} else {
				// Package with provider - parse normally
				provider, pkgName, err := parseUserPackageID(baseID)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return
				}
				if !isSupportedProviderFn(provider) {
					fmt.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, userPkgID, strings.Join(availableProvidersFn(), ", "))
					return
				}

				internalID = toInternalPackageID(provider, pkgName)
				// Construct displayID from provider and package name (full provider:package-id format)
				displayID = fmt.Sprintf("%s:%s", provider, pkgName)
			}

			internalIDs = append(internalIDs, internalID)
			displayIDs = append(displayIDs, displayID)
		}

		// Remove all packages
		fmt.Printf("Removing %d package(s)...\n", len(internalIDs))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for i := range internalIDs {
			internalID := internalIDs[i]
			displayID := displayIDs[i]

			// Remove the package with spinner showing package name
			var success bool
			action := func() {
				success = removePackageFn(internalID)
			}

			title := fmt.Sprintf("Removing %s...", displayID)
			if err := spinner.New().Title(title).Action(action).Run(); err != nil {
				fmt.Printf("%s Failed to remove %s: %v\n", IconClose(), displayID, err)
				failedCount++
				allSuccess = false
				continue
			}

			if success {
				fmt.Printf("%s Successfully removed %s\n", IconCheck(), displayID)
				successCount++
			} else {
				fmt.Printf("%s Failed to remove %s\n", IconClose(), displayID)
				failedCount++
				allSuccess = false
			}
		}

		// Print summary
		fmt.Printf("\nRemove Summary:\n")
		fmt.Printf("  Successfully removed: %d\n", successCount)
		fmt.Printf("  Failed to remove: %d\n", failedCount)

		if allSuccess {
			fmt.Printf("All packages removed successfully!\n")
		} else {
			fmt.Printf("Some packages failed to remove.\n")
		}
	},
}

// findInstalledPackagesByName searches installed packages for packages matching the given name
// (substring match, case-insensitive) and returns matches.
func findInstalledPackagesByName(packageName string) []PackageMatch {
	localPackagesRoot := newLocalPackagesParserFn()
	installedPackages := localPackagesRoot.Packages

	matches := []PackageMatch{}
	packageNameLower := strings.ToLower(packageName)

	for _, pkg := range installedPackages {
		sourceID := strings.TrimSpace(pkg.SourceID)
		if sourceID == "" {
			continue
		}

		// Get package name without provider
		displayID := displayPackageNameFromRegistryID(sourceID)
		if displayID == "" {
			continue
		}

		displayIDLower := strings.ToLower(displayID)

		// Check if package name contains the search term (substring match)
		if strings.Contains(displayIDLower, packageNameLower) {
			// Extract provider
			var provider string
			if strings.Contains(sourceID, ":") {
				parts := strings.SplitN(sourceID, ":", 2)
				if len(parts) == 2 {
					provider = parts[0]
				}
			}

			matches = append(matches, PackageMatch{
				SourceID:    sourceID,
				Provider:    provider,
				PackageName: displayID,
				Name:        displayID, // Use package name as display name
				Description: "",
				Version:     pkg.Version,
			})
		}
	}

	return matches
}

// indirections for testability
var (
	removePackageFn = providers.Remove
)
