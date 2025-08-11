package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update [pkgId]",
	Short:   "Update a package to the latest version",
	Aliases: []string{"up"},
	Long: `Update a package to its latest version from a supported provider.

Supported package ID formats:
  pkg:npm/@prisma/language-server
  pkg:golang/golang.org/x/tools/gopls
  pkg:pypi/black

Examples:
  zana update pkg:npm/@prisma/language-server
  zana up pkg:golang/golang.org/x/tools/gopls
  zana update pkg:pypi/black
  zana update --all (update all installed packages)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")

		if allFlag {
			// Update all installed packages
			fmt.Println("Updating all installed packages to latest versions...")
			success := updateAllPackages()
			if success {
				fmt.Println("Successfully updated all packages")
			} else {
				fmt.Println("Failed to update some packages")
			}
			return
		}

		// Check if package ID is provided
		if len(args) == 0 {
			fmt.Println("Error: Please provide a package ID or use --all flag")
			cmd.Help()
			return
		}

		pkgId := args[0]

		// Validate package ID format
		if !strings.HasPrefix(pkgId, "pkg:") {
			fmt.Printf("Error: Invalid package ID format. Must start with 'pkg:'\n")
			return
		}

		// Parse provider from package ID
		parts := strings.Split(strings.TrimPrefix(pkgId, "pkg:"), "/")
		if len(parts) < 2 {
			fmt.Printf("Error: Invalid package ID format. Expected 'pkg:provider/package-name'\n")
			return
		}

		provider := parts[0]
		if provider != "npm" && provider != "golang" && provider != "pypi" {
			fmt.Printf("Error: Unsupported provider '%s'. Supported providers: npm, golang, pypi\n", provider)
			return
		}

		fmt.Printf("Updating %s to latest version...\n", pkgId)

		// Update the package
		success := updater.Update(pkgId)
		if success {
			fmt.Printf("Successfully updated %s\n", pkgId)
		} else {
			fmt.Printf("Failed to update %s\n", pkgId)
		}
	},
}

func init() {
	updateCmd.Flags().BoolP("all", "A", false, "Update all installed packages to their latest versions")
}

func updateAllPackages() bool {
	// Get all installed packages
	localPackages := local_packages_parser.GetData(true).Packages

	if len(localPackages) == 0 {
		fmt.Println("No packages are currently installed")
		return true
	}

	fmt.Printf("Found %d installed packages\n", len(localPackages))

	allSuccess := true
	successCount := 0
	failedCount := 0

	for _, pkg := range localPackages {
		fmt.Printf("Updating %s...\n", pkg.SourceID)

		success := updater.Update(pkg.SourceID)
		if success {
			successCount++
			fmt.Printf("✓ Successfully updated %s\n", pkg.SourceID)
		} else {
			failedCount++
			fmt.Printf("✗ Failed to update %s\n", pkg.SourceID)
			allSuccess = false
		}
	}

	fmt.Printf("\nUpdate Summary:\n")
	fmt.Printf("  Successfully updated: %d\n", successCount)
	fmt.Printf("  Failed to update: %d\n", failedCount)

	return allSuccess
}
