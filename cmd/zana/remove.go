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

Examples:
  zana remove npm:@prisma/language-server
  zana rm golang:golang.org/x/tools/gopls npm:eslint
  zana delete pypi:black cargo:ripgrep
  zana remove npm:prettier golang:golang.org/x/tools/gopls`,
	Args: cobra.MinimumNArgs(1),
	// Enable shell completion for package IDs.
	ValidArgsFunction: packageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		packages := args

		// Validate and normalize all package IDs
		internalIDs := make([]string, len(packages))
		for i, userPkgID := range packages {
			provider, pkgName, err := parseUserPackageID(userPkgID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			if !isSupportedProviderFn(provider) {
				fmt.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, userPkgID, strings.Join(availableProvidersFn(), ", "))
				return
			}
			internalIDs[i] = toInternalPackageID(provider, pkgName)
		}

		// Remove all packages
		fmt.Printf("Removing %d package(s)...\n", len(packages))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for i, userPkgID := range packages {
			internalID := internalIDs[i]
			fmt.Printf("Removing %s...\n", userPkgID)

			// Remove the package
			success := removePackageFn(internalID)
			if success {
				fmt.Printf("✓ Successfully removed %s\n", userPkgID)
				successCount++
			} else {
				fmt.Printf("✗ Failed to remove %s\n", userPkgID)
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

// indirections for testability
var (
	removePackageFn = providers.Remove
)
