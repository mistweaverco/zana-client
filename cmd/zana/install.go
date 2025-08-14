package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

// validatePackageArgs validates the package arguments
func validatePackageArgs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("requires at least 1 argument")
	}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "pkg:") {
			return fmt.Errorf("invalid package ID format '%s': must start with 'pkg:'", arg)
		}

		// Parse provider from package ID
		parts := strings.Split(strings.TrimPrefix(arg, "pkg:"), "/")
		if len(parts) < 2 {
			return fmt.Errorf("invalid package ID format '%s': expected 'pkg:provider/package-name[@version]'", arg)
		}

		provider := parts[0]
		if provider == "" {
			return fmt.Errorf("invalid package ID format '%s': provider cannot be empty", arg)
		}

		packageName := parts[1]
		if packageName == "" {
			return fmt.Errorf("invalid package ID format '%s': package name cannot be empty", arg)
		}

		if !providers.IsSupportedProvider(provider) {
			return fmt.Errorf("unsupported provider '%s' for package '%s'. Supported providers: %s",
				provider, arg, strings.Join(providers.AvailableProviders, ", "))
		}
	}

	return nil
}

var installCmd = &cobra.Command{
	Use:     "install <pkgId> [pkgId...]",
	Aliases: []string{"add"},
	Short:   "Install one or more packages",
	Long: `Install one or more packages from supported providers.

Supported package ID formats:
  pkg:npm/@prisma/language-server
  pkg:npm/@prisma/language-server@latest
  pkg:golang/golang.org/x/tools/gopls@v0.14.0
  pkg:pypi/black@22.3.0
  pkg:cargo/ripgrep@13.0.0

Examples:
  zana install pkg:npm/@prisma/language-server
  zana install pkg:golang/golang.org/x/tools/gopls@latest
  zana install pkg:npm/eslint pkg:pypi/black@22.3.0
  zana install pkg:cargo/ripgrep@13.0.0 pkg:npm/prettier`,
	Args: func(cmd *cobra.Command, args []string) error {
		return validatePackageArgs(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Install all packages
		successCount := 0
		failureCount := 0
		var failures []string

		for _, pkgId := range args {
			// Parse package ID and version
			pkgId, version := parsePackageIDAndVersion(pkgId)

			if installPackageFn(pkgId, version) {
				successCount++
				fmt.Printf("✓ Successfully installed %s@%s\n", pkgId, version)
			} else {
				failureCount++
				failures = append(failures, pkgId)
				fmt.Printf("✗ Failed to install %s@%s\n", pkgId, version)
			}
		}

		// Print summary
		fmt.Printf("\nInstallation Summary:\n")
		fmt.Printf("  Successfully installed: %d\n", successCount)
		if failureCount > 0 {
			fmt.Printf("  Failed to install: %d\n", failureCount)
			fmt.Printf("  Failed packages: %s\n", strings.Join(failures, ", "))
		}
	},
}

// indirections for testability
var (
	isSupportedProviderFn = providers.IsSupportedProvider
	availableProvidersFn  = func() []string { return providers.AvailableProviders }
	installPackageFn      = providers.Install
)

// isValidVersionString checks if a string looks like a valid version
func isValidVersionString(version string) bool {
	// Common version patterns: "1.0.0", "latest", "v1.0.0", "1.0.0-beta", etc.
	// A version should contain at least one digit or be "latest"
	if version == "latest" {
		return true
	}

	// Check if it contains at least one digit
	for _, char := range version {
		if char >= '0' && char <= '9' {
			return true
		}
	}

	return false
}

// parsePackageIDAndVersion extracts the package ID and version from a full package ID string.
// It handles the format pkg:provider/name[@version] where name can contain @ symbols.
func parsePackageIDAndVersion(pkgId string) (string, string) {
	// Split by @ and check if the last part looks like a version
	parts := strings.Split(pkgId, "@")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		// Check if the last part looks like a version (contains digits or is "latest")
		if isValidVersionString(lastPart) {
			// Reconstruct the package name without the version
			packageName := strings.Join(parts[:len(parts)-1], "@")
			return packageName, lastPart
		}
	}
	// No valid version found, return the full package ID with "latest"
	return pkgId, "latest"
}
