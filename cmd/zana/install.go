package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/updater"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install <pkgId> [version]",
	Aliases: []string{"add"},
	Short:   "Install a package",
	Long: `Install a package from a supported provider.

Supported package ID formats:
  pkg:npm/@prisma/language-server
  pkg:golang/golang.org/x/tools/gopls
  pkg:pypi/black

Examples:
  zana install pkg:npm/@prisma/language-server
  zana install pkg:golang/golang.org/x/tools/gopls latest
  zana install pkg:pypi/black 22.3.0`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		pkgId := args[0]
		version := "latest"
		if len(args) > 1 {
			version = args[1]
		}

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

		fmt.Printf("Installing %s (version: %s)...\n", pkgId, version)

		// Install the package
		success := updater.Install(pkgId, version)
		if success {
			fmt.Printf("Successfully installed %s\n", pkgId)
		} else {
			fmt.Printf("Failed to install %s\n", pkgId)
		}
	},
}
