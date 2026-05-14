package providers

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/spinnerutil"
	"github.com/mistweaverco/zana-client/internal/lib/treesitterdeps"
)

// PreflightTreeSitterInjectionQueryPackages installs ambiguous Tree-sitter-queries registry packages
// referenced by treesitter.build[].injections (per Neovim integration) before the main install proceeds.
func PreflightTreeSitterInjectionQueryPackages(registryItem registry_parser.RegistryItem, resolvedVersion string) error {
	return ensureNeovimTreeSitterInjectionQueryPackages(registryItem, resolvedVersion)
}

func ensureNeovimTreeSitterInjectionQueryPackages(registryItem registry_parser.RegistryItem, resolvedVersion string) error {
	if registryItem.Source.ID == "" || registryItem.TreeSitter == nil {
		return nil
	}
	if !IsTreeSitterCategory(registryItem.Categories) {
		return nil
	}
	if !integrationEnabled("neovim") {
		return nil
	}
	reg := registry_parser.NewDefaultRegistryParser()
	langs := treesitterdeps.MergeInjectionLanguagesForEditor(registryItem, "neovim")
	if len(langs) == 0 {
		return nil
	}
	for _, lang := range langs {
		lang = strings.ToLower(strings.TrimSpace(lang))
		if lang == "" {
			continue
		}
		cands := treesitterdeps.QueryPackageCandidates(reg, lang, "neovim")
		if len(cands) == 0 {
			continue
		}
		sourceID, err := resolveQueryPackageSourceIDForLanguage(lang, "neovim", reg, registryItem.Source.ID, resolvedVersion)
		if err != nil {
			return err
		}
		if strings.TrimSpace(sourceID) == "" {
			continue
		}
		if local_packages_parser.IsPackageInstalled(sourceID) {
			continue
		}
		ver := strings.TrimSpace(reg.GetLatestVersion(sourceID))
		if ver == "" {
			ver = "latest"
		}
		title := fmt.Sprintf("Installing Neovim query package %s@%s (%s)...", sourceID, ver, lang)
		var installFailed bool
		action := func() {
			if !Install(sourceID, ver) {
				installFailed = true
			}
		}
		if err := spinnerutil.RunWithTTYOrPlain(title, func() {
			if NeovimInheritInstallNotifierStart != nil {
				NeovimInheritInstallNotifierStart(sourceID, ver)
			}
		}, action); err != nil {
			return err
		}
		if installFailed {
			return fmt.Errorf("failed to install Tree-sitter-queries package %s for injection language %q", sourceID, lang)
		}
		noteTreeSitterDependencyInstallSuccess()
		if NeovimInheritInstallNotifierDone != nil {
			instVer := strings.TrimSpace(local_packages_parser.GetBySourceId(sourceID).Version)
			if instVer == "" {
				instVer = ver
			}
			NeovimInheritInstallNotifierDone(sourceID, instVer)
		}
	}
	return nil
}

func resolveQueryPackageSourceIDForLanguage(
	lang, integration string,
	reg *registry_parser.RegistryParser,
	consumerSourceID, consumerResolvedVersion string,
) (string, error) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	integration = strings.ToLower(strings.TrimSpace(integration))
	if lang == "" || integration == "" {
		return "", fmt.Errorf("empty language or integration")
	}
	consumerSourceID = strings.TrimSpace(consumerSourceID)
	if lockID, ok := local_packages_parser.GetTreeSitterQueryLockChoice(consumerSourceID, lang, integration); ok && lockID != "" {
		item := reg.GetBySourceId(lockID)
		if item.Source.ID != "" && treesitterdeps.IsTreeSitterQueriesPackage(item.Categories) {
			return lockID, nil
		}
	}
	cands := treesitterdeps.QueryPackageCandidates(reg, lang, integration)
	if len(cands) == 0 {
		return "", nil
	}
	if len(cands) == 1 {
		return cands[0], nil
	}
	sort.Strings(cands)
	title := fmt.Sprintf("Multiple Tree-sitter-queries packages for %q (%s)", lang, integration)
	desc := fmt.Sprintf(
		"Choose which registry package supplies Neovim queries for this language when installing %s.\n\nThis choice is saved in zana-lock.json.",
		consumerSourceID,
	)
	if !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stderr.Fd()) {
		return "", fmt.Errorf(
			"%s\n%s\n\nNon-interactive session: add extras.treesitter_query_choices to the lock row for %s with {\"language\":%q,\"integration\":%q,\"sourceId\":\"...\"} (candidates: %s)",
			title, desc, consumerSourceID, lang, integration, strings.Join(cands, ", "),
		)
	}
	var chosen string
	opts := make([]huh.Option[string], 0, len(cands))
	for _, id := range cands {
		opts = append(opts, huh.NewOption(id, id))
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Description(desc).
				Options(opts...).
				Value(&chosen),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	chosen = strings.TrimSpace(chosen)
	if chosen == "" {
		return "", fmt.Errorf("no Tree-sitter-queries package selected for language %q", lang)
	}
	ver := lockVersionForQueryChoice(reg, consumerSourceID, consumerResolvedVersion)
	if err := local_packages_parser.MergePackageTreeSitterQueryChoice(consumerSourceID, lang, integration, chosen, ver); err != nil {
		return "", err
	}
	return chosen, nil
}

func lockVersionForQueryChoice(reg *registry_parser.RegistryParser, consumerSourceID, explicit string) string {
	v := strings.TrimSpace(explicit)
	if v != "" {
		return v
	}
	if v = strings.TrimSpace(local_packages_parser.GetBySourceId(consumerSourceID).Version); v != "" {
		return v
	}
	return strings.TrimSpace(reg.GetLatestVersion(consumerSourceID))
}
