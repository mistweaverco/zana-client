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
  pkg:npm/@prisma/language-server
  pkg:golang/golang.org/x/tools/gopls
  pkg:pypi/black
  pkg:cargo/ripgrep

Examples:
  zana remove pkg:npm/@prisma/language-server
  zana rm pkg:golang/golang.org/x/tools/gopls pkg:npm/eslint
  zana delete pkg:pypi/black pkg:cargo/ripgrep
  zana remove pkg:npm/prettier pkg:golang/golang.org/x/tools/gopls`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		packages := args

		// Validate all package IDs
		for _, pkgId := range packages {
			if !strings.HasPrefix(pkgId, "pkg:") {
				fmt.Printf("Error: Invalid package ID format '%s'. Must start with 'pkg:'\n", pkgId)
				return
			}

			// Parse provider from package ID
			parts := strings.Split(strings.TrimPrefix(pkgId, "pkg:"), "/")
			if len(parts) < 2 {
				fmt.Printf("Error: Invalid package ID format '%s'. Expected 'pkg:provider/package-name'\n", pkgId)
				return
			}

			provider := parts[0]
			if !isSupportedProviderFn(provider) {
				fmt.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, pkgId, strings.Join(availableProvidersFn(), ", "))
				return
			}
		}

		// Remove all packages
		fmt.Printf("Removing %d package(s)...\n", len(packages))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for _, pkgId := range packages {
			fmt.Printf("Removing %s...\n", pkgId)

			// Remove the package
			success := removePackageFn(pkgId)
			if success {
				fmt.Printf("✓ Successfully removed %s\n", pkgId)
				successCount++
			} else {
				fmt.Printf("✗ Failed to remove %s\n", pkgId)
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
