package zana

import (
	"fmt"
	"strings"

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
		internalIDs := make([]string, len(packages))
		displayIDs := make([]string, len(packages))

		for i, userPkgID := range packages {
			// Parse package ID and version from the user-facing ID
			baseID, _ := parsePackageIDAndVersion(userPkgID)

			var internalID string
			var displayID string

			// Check if this is a package name without provider
			if !strings.Contains(baseID, ":") && !strings.HasPrefix(baseID, "pkg:") {
				// Package name without provider - search installed packages and prompt user
				matches := findInstalledPackagesByName(baseID)
				if len(matches) == 0 {
					fmt.Printf("✗ No installed packages found matching '%s'\n", baseID)
					return
				}

				selectedSourceID, err := promptForProviderSelection(baseID, matches)
				if err != nil {
					fmt.Printf("✗ Error selecting provider for '%s': %v\n", baseID, err)
					return
				}

				internalID = selectedSourceID
				displayID = userPkgID
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
				displayID = userPkgID
			}

			internalIDs[i] = internalID
			displayIDs[i] = displayID
		}

		// Remove all packages
		fmt.Printf("Removing %d package(s)...\n", len(packages))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for i := range packages {
			internalID := internalIDs[i]
			displayID := displayIDs[i]
			fmt.Printf("Removing %s...\n", displayID)

			// Remove the package
			success := removePackageFn(internalID)
			if success {
				fmt.Printf("✓ Successfully removed %s\n", displayID)
				successCount++
			} else {
				fmt.Printf("✗ Failed to remove %s\n", displayID)
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