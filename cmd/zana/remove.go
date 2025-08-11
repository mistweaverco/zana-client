package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/updater"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <pkgId>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a package",
	Long: `Remove a package from a supported provider.

Supported package ID formats:
  pkg:npm/@prisma/language-server
  pkg:golang/golang.org/x/tools/gopls
  pkg:pypi/black

Examples:
  zana remove pkg:npm/@prisma/language-server
  zana rm pkg:golang/golang.org/x/tools/gopls
  zana delete pkg:pypi/black`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		fmt.Printf("Removing %s...\n", pkgId)

		// Remove the package
		success := updater.Remove(pkgId)
		if success {
			fmt.Printf("Successfully removed %s\n", pkgId)
		} else {
			fmt.Printf("Failed to remove %s\n", pkgId)
		}
	},
}
