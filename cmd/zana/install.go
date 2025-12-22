package zana

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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
// It allows package names without providers (they will be handled in Run function).
func validatePackageArgs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("requires at least 1 argument")
	}

	for _, arg := range args {
		// Check if it's a package name without provider (no colon and no pkg: prefix)
		if !strings.Contains(arg, ":") && !strings.HasPrefix(arg, "pkg:") {
			// Allow it - will be handled in Run function
			continue
		}

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
  github:user/repo
  github:user/repo@v1.0.0
  gitlab:group/subgroup/project
  gitlab:group/subgroup/project@v1.0.0
  codeberg:user/repo
  codeberg:user/repo@v1.0.0

Examples:
  zana install npm:@prisma/language-server
  zana install golang:golang.org/x/tools/gopls@latest
  zana install npm:eslint pypi:black@22.3.0
  zana install cargo:ripgrep@13.0.0 npm:prettier
  zana install github:sharkdp/bat
  zana install gitlab:group/subgroup/myproject@v1.0.0
  zana install codeberg:user/repo`,
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

			var internalID string
			var displayID string

			// Check if this is a package name without provider
			if !strings.Contains(baseID, ":") && !strings.HasPrefix(baseID, "pkg:") {
				// Package name without provider - search registry and prompt user
				matches := findPackagesByName(baseID)
				if len(matches) == 0 {
					fmt.Printf("✗ No packages found matching '%s'\n", baseID)
					failureCount++
					failures = append(failures, userPkgID)
					continue
				}

				// Filter matches to exact package name matches first (for better UX)
				exactMatches := []PackageMatch{}
				partialMatches := []PackageMatch{}
				baseIDLower := strings.ToLower(baseID)

				for _, match := range matches {
					matchNameLower := strings.ToLower(match.PackageName)
					if matchNameLower == baseIDLower {
						exactMatches = append(exactMatches, match)
					} else {
						partialMatches = append(partialMatches, match)
					}
				}

				// Use exact matches if available, otherwise use partial matches
				matchesToShow := exactMatches
				if len(exactMatches) == 0 {
					matchesToShow = partialMatches
				}

				selectedSourceID, err := promptForProviderSelection(baseID, matchesToShow)
				if err != nil {
					fmt.Printf("✗ Error selecting provider for '%s': %v\n", baseID, err)
					failureCount++
					failures = append(failures, userPkgID)
					continue
				}

				internalID = selectedSourceID
				displayID = userPkgID
			} else {
				// Package with provider - parse normally
				provider, pkgName, err := parseUserPackageID(baseID)
				if err != nil {
					// This shouldn't happen because Args validation already ran,
					// but guard just in case.
					fmt.Printf("Error: %v\n", err)
					return
				}

				internalID = toInternalPackageID(provider, pkgName)
				displayID = userPkgID
			}

			if installPackageFn(internalID, version) {
				successCount++
				fmt.Printf("✓ Successfully installed %s@%s\n", displayID, version)
			} else {
				failureCount++
				failures = append(failures, displayID)
				fmt.Printf("✗ Failed to install %s@%s\n", displayID, version)
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

// PackageMatch represents a package found in the registry
type PackageMatch struct {
	SourceID    string
	Provider    string
	PackageName string
	Name        string
	Description string
	Version     string
}

// findPackagesByName searches the registry for packages matching the given name
// (substring match, case-insensitive) and returns matches grouped by provider.
func findPackagesByName(packageName string) []PackageMatch {
	parser := newRegistryParserFn()
	items := parser.GetData(false)

	matches := []PackageMatch{}
	packageNameLower := strings.ToLower(packageName)

	for _, item := range items {
		// Extract provider and package name from source ID
		sourceID := strings.TrimSpace(item.Source.ID)
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
				Name:        item.Name,
				Description: item.Description,
				Version:     strings.TrimSpace(item.Version),
			})
		}
	}

	return matches
}

// promptForProviderSelection prompts the user to select a provider when multiple
// packages with the same name are found across different providers.
func promptForProviderSelection(packageName string, matches []PackageMatch) (string, error) {
	if len(matches) == 0 {
		return "", fmt.Errorf("no packages found matching '%s'", packageName)
	}

	// Group matches by provider to show unique providers
	providerMatches := make(map[string][]PackageMatch)
	for _, match := range matches {
		providerMatches[match.Provider] = append(providerMatches[match.Provider], match)
	}

	// Create a list of unique providers with their first match
	uniqueProviders := []string{}
	providerToMatch := make(map[string]PackageMatch)
	for provider, providerMatches := range providerMatches {
		uniqueProviders = append(uniqueProviders, provider)
		providerToMatch[provider] = providerMatches[0] // Use first match as representative
	}

	if len(uniqueProviders) == 1 {
		// Only one provider, auto-select and show what was selected
		selected := matches[0]
		fmt.Printf("✓ Found '%s' in %s provider: %s\n", packageName, selected.Provider, selected.SourceID)
		return selected.SourceID, nil
	}

	if len(matches) == 1 {
		// Only one match total, auto-select
		fmt.Printf("✓ Found '%s': %s\n", packageName, matches[0].SourceID)
		return matches[0].SourceID, nil
	}

	// Multiple providers found, prompt user
	fmt.Printf("\n🔍 Found '%s' in multiple providers:\n\n", packageName)

	// Show options
	for i, provider := range uniqueProviders {
		match := providerToMatch[provider]
		fmt.Printf("  %d. %s:%s", i+1, provider, match.PackageName)
		if match.Name != "" && match.Name != match.PackageName {
			fmt.Printf(" (%s)", match.Name)
		}
		if match.Description != "" {
			// Truncate description if too long
			desc := match.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf(" - %s", desc)
		}
		fmt.Println()
	}

	fmt.Printf("\nSelect provider (1-%d): ", len(uniqueProviders))

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(uniqueProviders) {
		return "", fmt.Errorf("invalid selection: please choose a number between 1 and %d", len(uniqueProviders))
	}

	selectedProvider := uniqueProviders[choice-1]
	return providerToMatch[selectedProvider].SourceID, nil
}
