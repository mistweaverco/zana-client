package zana

import (
	"fmt"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

// parseUserPackageID parses a user-facing package ID and returns
// the provider and raw package identifier (without version).
//
// It supports both the legacy format:
//
//	pkg:<provider>/<package-name>
//
// and the new simplified format:
//
//	<provider>:<package-name>
func parseUserPackageID(arg string) (string, string, error) {
	// Legacy format: pkg:provider/name
	if strings.HasPrefix(arg, "pkg:") {
		parts := strings.Split(strings.TrimPrefix(arg, "pkg:"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid package ID format '%s': expected 'pkg:provider/package-name[@version]'", arg)
		}
		provider := parts[0]
		if provider == "" {
			return "", "", fmt.Errorf("invalid package ID format '%s': provider cannot be empty", arg)
		}
		packageName := parts[1]
		if packageName == "" {
			return "", "", fmt.Errorf("invalid package ID format '%s': package name cannot be empty", arg)
		}
		return provider, packageName, nil
	}

	// New format: provider:package-name
	if !strings.Contains(arg, ":") {
		return "", "", fmt.Errorf("invalid package ID format '%s': expected '<provider>:<package-id>[@version]'", arg)
	}

	parts := strings.SplitN(arg, ":", 2)
	provider := parts[0]
	packageName := parts[1]

	if provider == "" {
		return "", "", fmt.Errorf("invalid package ID format '%s': provider cannot be empty", arg)
	}
	if packageName == "" {
		return "", "", fmt.Errorf("invalid package ID format '%s': package id cannot be empty", arg)
	}

	return provider, packageName, nil
}

// toInternalPackageID normalizes a user-facing package ID to the
// internal representation "<provider>:<package-id>".
// This is the format used in zana-lock.json and throughout the codebase.
func toInternalPackageID(provider, pkgID string) string {
	return provider + ":" + pkgID
}

// normalizePackageID converts a package ID from legacy format (pkg:provider/pkg)
// to the new format (provider:pkg), or returns it unchanged if already in new format.
// This is used for backward compatibility when reading zana-lock.json.
func normalizePackageID(sourceID string) string {
	if strings.HasPrefix(sourceID, "pkg:") {
		rest := strings.TrimPrefix(sourceID, "pkg:")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			return parts[0] + ":" + parts[1]
		}
	}
	return sourceID
}

// validatePackageArgs validates the package arguments (for cobra)
// and ensures that all providers are supported.
func validatePackageArgs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("requires at least 1 argument")
	}

	for _, arg := range args {
		provider, _, err := parseUserPackageID(arg)
		if err != nil {
			return err
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
  npm:@prisma/language-server
  npm:@prisma/language-server@latest
  golang:golang.org/x/tools/gopls@v0.14.0
  pypi:black@22.3.0
  cargo:ripgrep@13.0.0

Examples:
  zana install npm:@prisma/language-server
  zana install golang:golang.org/x/tools/gopls@latest
  zana install npm:eslint pypi:black@22.3.0
  zana install cargo:ripgrep@13.0.0 npm:prettier`,
	Args: func(cmd *cobra.Command, args []string) error {
		return validatePackageArgs(args)
	},
	// Enable shell completion for package IDs based on the local registry.
	ValidArgsFunction: packageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		// Install all packages
		successCount := 0
		failureCount := 0
		var failures []string

		for _, userPkgID := range args {
			// Parse package ID and version from the user-facing ID
			baseID, version := parsePackageIDAndVersion(userPkgID)
			provider, pkgName, err := parseUserPackageID(baseID)
			if err != nil {
				// This shouldn't happen because Args validation already ran,
				// but guard just in case.
				fmt.Printf("Error: %v\n", err)
				return
			}

			internalID := toInternalPackageID(provider, pkgName)

			if installPackageFn(internalID, version) {
				successCount++
				fmt.Printf("✓ Successfully installed %s@%s\n", userPkgID, version)
			} else {
				failureCount++
				failures = append(failures, userPkgID)
				fmt.Printf("✗ Failed to install %s@%s\n", userPkgID, version)
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
