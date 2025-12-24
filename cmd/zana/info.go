package zana

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Aliases: []string{"show", "details"},
	Use:     "info [package-id]...",
	Short:   "Show detailed information about packages",
	Long: `Show detailed information about one or more packages from the registry.

Examples:
  zana info npm:eslint
  zana info pypi:black
  zana info golang:golang.org/x/tools/gopls
  zana info eslint (will prompt for provider selection if multiple matches)
  zana info npm:eslint pypi:black`,
	Args: cobra.MinimumNArgs(1),
	// Enable shell completion for package IDs based on the local registry.
	ValidArgsFunction: packageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		// Ensure registry is available
		_ = downloadAndUnzipRegistryFn()

		parser := newRegistryParserFn()

		// Process all packages
		packagesToShow := make([]string, 0, len(args))

		for _, userPkgID := range args {
			baseID, _ := parsePackageIDAndVersion(userPkgID)

			// Check if this is a package name without provider
			if !strings.Contains(baseID, ":") && !strings.HasPrefix(baseID, "pkg:") {
				// Package name without provider - search registry and prompt user
				matches := findPackagesByName(baseID)
				if len(matches) == 0 {
					fmt.Printf("%s No packages found matching '%s'\n", IconClose(), baseID)
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

				// Prompt for selection
				selectedSourceIDs, err := promptForProviderSelection(baseID, matchesToShow, false, "view")
				if err != nil {
					fmt.Printf("%s Error selecting provider for '%s': %v\n", IconClose(), baseID, err)
					continue
				}

				packagesToShow = append(packagesToShow, selectedSourceIDs...)
			} else {
				// Package with provider - parse normally
				provider, pkgName, err := parseUserPackageID(baseID)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}
				if !providers.IsSupportedProvider(provider) {
					fmt.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, userPkgID, strings.Join(providers.AvailableProviders, ", "))
					continue
				}

				sourceID := toInternalPackageID(provider, pkgName)
				packagesToShow = append(packagesToShow, sourceID)
			}
		}

		// Display info for each package
		for i, sourceID := range packagesToShow {
			if i > 0 {
				fmt.Println() // Add spacing between multiple packages
			}

			item := parser.GetBySourceId(sourceID)
			if item.Source.ID == "" {
				fmt.Printf("%s Package '%s' not found in registry\n", IconClose(), sourceID)
				continue
			}

			displayPackageInfo(item, sourceID)
		}
	},
}

// displayPackageInfo renders package information as markdown using glamour
func displayPackageInfo(item registry_parser.RegistryItem, sourceID string) {
	// Build markdown content
	var markdown strings.Builder

	// Package name and ID
	markdown.WriteString(fmt.Sprintf("# %s\n\n", item.Name))
	markdown.WriteString(fmt.Sprintf("**Package ID:** `%s`\n\n", sourceID))

	// Version
	if item.Version != "" {
		markdown.WriteString(fmt.Sprintf("**Version:** `%s`\n\n", item.Version))
	}

	// Description
	if item.Description != "" {
		markdown.WriteString(fmt.Sprintf("## Description\n\n%s\n\n", item.Description))
	}

	// Homepage
	if item.Homepage != "" {
		markdown.WriteString(fmt.Sprintf("**Homepage:** %s\n\n", item.Homepage))
	}

	// Provider
	if strings.Contains(sourceID, ":") {
		parts := strings.SplitN(sourceID, ":", 2)
		if len(parts) == 2 {
			markdown.WriteString(fmt.Sprintf("**Provider:** `%s`\n\n", parts[0]))
		}
	}

	// Licenses
	if len(item.Licenses) > 0 {
		markdown.WriteString(fmt.Sprintf("**Licenses:** %s\n\n", strings.Join(item.Licenses, ", ")))
	}

	// Languages
	if len(item.Languages) > 0 {
		markdown.WriteString(fmt.Sprintf("**Languages:** %s\n\n", strings.Join(item.Languages, ", ")))
	}

	// Categories
	if len(item.Categories) > 0 {
		markdown.WriteString(fmt.Sprintf("**Categories:** %s\n\n", strings.Join(item.Categories, ", ")))
	}

	// Installation status
	localPackagesRoot := newLocalPackagesParserFn()
	installedPackages := localPackagesRoot.Packages
	isInstalled := false
	installedVersion := ""
	for _, pkg := range installedPackages {
		if pkg.SourceID == sourceID {
			isInstalled = true
			installedVersion = pkg.Version
			break
		}
	}

	if isInstalled {
		if installedVersion != "" {
			markdown.WriteString(fmt.Sprintf("**Status:** ✅ Installed (version: `%s`)\n\n", installedVersion))
		} else {
			markdown.WriteString(fmt.Sprintf("**Status:** ✅ Installed\n\n"))
		}
	} else {
		markdown.WriteString(fmt.Sprintf("**Status:** ⬜ Not installed\n\n"))
	}

	// Aliases
	if len(item.Aliases) > 0 {
		markdown.WriteString(fmt.Sprintf("**Aliases:** %s\n\n", strings.Join(item.Aliases, ", ")))
	}

	// Binaries
	if len(item.Bin) > 0 {
		markdown.WriteString("## Binaries\n\n")
		for binName, binPath := range item.Bin {
			markdown.WriteString(fmt.Sprintf("- **%s:** `%s`\n", binName, binPath))
		}
		markdown.WriteString("\n")
	}

	// Render markdown with glamour
	rendered, err := glamour.Render(markdown.String(), "dark")
	if err != nil {
		// Fallback to plain text if rendering fails
		fmt.Println(markdown.String())
		return
	}

	fmt.Print(rendered)
}