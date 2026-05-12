package providers

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
	"github.com/mistweaverco/zana-client/internal/lib/spinnerutil"
)

// Injectable helpers for tests
var (
	neovimShellOutCapture = shell_out.ShellOutCapture
	neovimMkdirAll        = os.MkdirAll
	neovimRemove          = os.Remove
	neovimRemoveAll       = os.RemoveAll
	neovimStat            = os.Stat
	neovimUserHomeDir     = os.UserHomeDir
	neovimGetenv          = os.Getenv
	neovimReadFile        = os.ReadFile
	neovimWriteFile       = os.WriteFile
)

// neovimInheritsPromptAction is returned by [neovimInheritsPrompt] (injectable for tests).
type neovimInheritsPromptAction string

const (
	neovimInheritsAbort    neovimInheritsPromptAction = "abort"
	neovimInheritsContinue neovimInheritsPromptAction = "continue"
	neovimInheritsInstall  neovimInheritsPromptAction = "install"
)

// neovimInheritsPrompt is swapped in tests to avoid interactive huh.
var neovimInheritsPrompt = defaultNeovimInheritsPrompt

func defaultNeovimInheritsPrompt(title, description string) (neovimInheritsPromptAction, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stderr.Fd()) {
		return neovimInheritsAbort, fmt.Errorf("%s\n%s", title, description)
	}
	var choice neovimInheritsPromptAction = neovimInheritsInstall
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[neovimInheritsPromptAction]().
				Title(title).
				Description(description).
				Options(
					huh.NewOption("Install missing dependencies", neovimInheritsInstall),
					huh.NewOption("Continue anyway", neovimInheritsContinue),
					huh.NewOption("Abort", neovimInheritsAbort),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return neovimInheritsAbort, err
	}
	return choice, nil
}

// NeovimInheritInstallNotifierStart, if non-nil, is called before each inherited tree-sitter
// grammar install (e.g. CLI prints progress while the parent install spinner is running).
var NeovimInheritInstallNotifierStart func(sourceID string, registryVersion string)

// NeovimInheritInstallNotifierDone, if non-nil, is called after each successful inherited install
// with the version from the lockfile (matches integration report keys).
var NeovimInheritInstallNotifierDone func(sourceID string, resolvedVersion string)

func neovimTreeSitterQueriesCacheDir(sourceID, version, lang string) string {
	return filepath.Join(TreeSitterArtifactVersionDir(sourceID, version), "queries", lang)
}

// resolveNeovimTreeSitterQueriesDir finds highlights/injections (etc.) for Neovim under a grammar checkout.
// Prefers grammar-local queries, then repo-root queries (for grammars in subdirectories).
func resolveNeovimTreeSitterQueriesDir(repoPath, fullGrammarDir string) string {
	candidates := []string{
		filepath.Join(fullGrammarDir, "queries"),
		filepath.Join(repoPath, "queries"),
	}
	seen := map[string]struct{}{}
	for _, c := range candidates {
		key := filepath.Clean(c)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

func hasTreeSitterInheritsModeline(content []byte) bool {
	nonEmpty := 0
	for _, line := range bytes.Split(content, []byte{'\n'}) {
		t := bytes.TrimSpace(line)
		if len(t) == 0 {
			continue
		}
		nonEmpty++
		if nonEmpty > 8 {
			break
		}
		if !bytes.HasPrefix(t, []byte(";")) {
			return false
		}
		low := bytes.ToLower(t)
		if bytes.Contains(low, []byte("inherits:")) {
			return true
		}
	}
	return false
}

func patchNeovimTreeSitterSCM(content []byte, inherits []string) []byte {
	s := strings.ReplaceAll(string(content), "#is-not?", "#not-eq?")
	b := []byte(s)
	if len(inherits) == 0 {
		return b
	}
	clean := make([]string, 0, len(inherits))
	seen := map[string]struct{}{}
	for _, in := range inherits {
		in = strings.TrimSpace(in)
		if in == "" {
			continue
		}
		k := strings.ToLower(in)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		clean = append(clean, in)
	}
	if len(clean) == 0 || hasTreeSitterInheritsModeline(b) {
		return b
	}
	prefix := "; inherits: " + strings.Join(clean, ", ") + "\n"
	return append([]byte(prefix), b...)
}

func copyNeovimTreeSitterQueriesDir(src, dst string, inherits []string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.EqualFold(filepath.Ext(path), ".scm") {
			b = patchNeovimTreeSitterSCM(b, inherits)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func cacheNeovimTreeSitterQueriesAfterBuild(
	repoPath, fullGrammarDir, sourceID, version string,
	build registry_parser.RegistryItemTreeSitterBuild,
	allowExternalQueryClone func(string) bool,
) (*local_packages_parser.TreeSitterExternalQueryPin, error) {
	lang := strings.TrimSpace(build.Language)
	if lang == "" {
		return nil, nil
	}
	if !TreeSitterBuildDeclaresNeovimIntegration(build) || !integrationEnabled("neovim") {
		return nil, nil
	}
	dest := neovimTreeSitterQueriesCacheDir(sourceID, version, lang)
	if err := os.RemoveAll(dest); err != nil {
		return nil, fmt.Errorf("clear cached queries for %s: %w", lang, err)
	}

	repoURL := ""
	if build.ExternalQueries != nil && allowExternalQueryClone(lang) {
		repoURL = strings.TrimSpace(build.ExternalQueries.RepoURL)
	}
	if repoURL != "" {
		lockRepo, lockRef, haveLock := externalQueryLockPinFromLocalLock(sourceID, version, lang)
		if haveLock && !externalQueryRepoURLsEqual(lockRepo, repoURL) {
			haveLock = false
		}
		var lockR, lockRefVal string
		if haveLock {
			lockR, lockRefVal = lockRepo, lockRef
		}
		cloneDir := filepath.Join(TreeSitterArtifactVersionDir(sourceID, version), "external-query-clones", lang)
		cloneTitle := fmt.Sprintf("Cloning external tree-sitter queries (%s)...", lang)
		var resolved string
		var cloneErr error
		if spinErr := spinnerutil.RunIfTTY(cloneTitle, func() {
			resolved, cloneErr = cloneExternalQueriesRepo(
				repoURL,
				cloneDir,
				build.ExternalQueries.Ref,
				build.ExternalQueries.Semver,
				lockR,
				lockRefVal,
			)
		}); spinErr != nil {
			return nil, fmt.Errorf("external queries for %s: %w", lang, spinErr)
		}
		if cloneErr != nil {
			return nil, fmt.Errorf("external queries for %s: %w", lang, cloneErr)
		}
		extSrc := resolveNeovimTreeSitterQueriesDir(cloneDir, cloneDir)
		if extSrc == "" {
			return nil, fmt.Errorf("external queries repo %s has no queries/ directory usable for language %s", repoURL, lang)
		}
		if err := copyNeovimTreeSitterQueriesDir(extSrc, dest, build.Inherits); err != nil {
			return nil, fmt.Errorf("cache external tree-sitter queries for %s: %w", lang, err)
		}
		return &local_packages_parser.TreeSitterExternalQueryPin{
			Language: lang,
			RepoURL:  repoURL,
			Ref:      resolved,
		}, nil
	}

	if src := resolveNeovimTreeSitterQueriesDir(repoPath, fullGrammarDir); src != "" {
		if err := copyNeovimTreeSitterQueriesDir(src, dest, build.Inherits); err != nil {
			return nil, fmt.Errorf("cache tree-sitter queries for %s: %w", lang, err)
		}
		return nil, nil
	}
	return nil, nil
}

func cacheNeovimTreeSitterQueriesForBuiltLangs(
	repoPath, sourceID, version string,
	build []registry_parser.RegistryItemTreeSitterBuild,
	builtLangs []string,
	externalPreflight *ExternalQueryPreflightChoice,
) ([]local_packages_parser.TreeSitterExternalQueryPin, error) {
	var pins []local_packages_parser.TreeSitterExternalQueryPin
	want := map[string]struct{}{}
	for _, l := range builtLangs {
		l = strings.TrimSpace(l)
		if l != "" {
			want[l] = struct{}{}
		}
	}
	var allowExternalQueryClone func(string) bool
	if !integrationEnabled("neovim") {
		allowExternalQueryClone = func(string) bool { return false }
	} else if externalPreflight != nil {
		needs := collectExternalTreeSitterQueryNeeds(repoPath, build, builtLangs)
		allowUnpinned := externalPreflight.AllowUnpinned
		pinned := make(map[string]struct{}, len(needs))
		for _, n := range needs {
			if externalQueryLockCoversNeed(sourceID, version, n) {
				pinned[strings.ToLower(strings.TrimSpace(n.Lang))] = struct{}{}
			}
		}
		allowExternalQueryClone = func(lang string) bool {
			lk := strings.ToLower(strings.TrimSpace(lang))
			if _, ok := pinned[lk]; ok {
				return true
			}
			return allowUnpinned
		}
	} else {
		needs := collectExternalTreeSitterQueryNeeds(repoPath, build, builtLangs)
		needsConfirm := externalQueryNeedsStillRequiringConfirm(sourceID, version, needs)
		allowUnpinned := true
		var err error
		if len(needsConfirm) > 0 {
			allowUnpinned, err = batchConfirmExternalTreeSitterQueries(sourceID, needsConfirm)
			if err != nil {
				return nil, err
			}
		}
		pinned := make(map[string]struct{}, len(needs))
		for _, n := range needs {
			if externalQueryLockCoversNeed(sourceID, version, n) {
				pinned[strings.ToLower(strings.TrimSpace(n.Lang))] = struct{}{}
			}
		}
		allowExternalQueryClone = func(lang string) bool {
			lk := strings.ToLower(strings.TrimSpace(lang))
			if _, ok := pinned[lk]; ok {
				return true
			}
			return allowUnpinned
		}
	}
	for _, b := range build {
		lang := strings.TrimSpace(b.Language)
		grammarDir := strings.TrimSpace(b.GrammarDir)
		if lang == "" || grammarDir == "" {
			continue
		}
		if _, ok := want[lang]; !ok {
			continue
		}
		fullGrammarDir := filepath.Join(repoPath, filepath.FromSlash(grammarDir))
		pin, err := cacheNeovimTreeSitterQueriesAfterBuild(repoPath, fullGrammarDir, sourceID, version, b, allowExternalQueryClone)
		if err != nil {
			return nil, err
		}
		if pin != nil {
			pins = append(pins, *pin)
		}
	}
	return pins, nil
}

func detectNeovimDataPath() (string, error) {
	// Prefer Neovim itself: this respects OS + user overrides.
	if code, out, err := neovimShellOutCapture("nvim", []string{
		"--headless",
		"+lua io.write(vim.fn.stdpath('data'))",
		"+q",
	}, "", nil); err == nil && code == 0 {
		p := strings.TrimSpace(out)
		if p != "" {
			return p, nil
		}
	}

	home, err := neovimUserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "linux":
		if xdg := strings.TrimSpace(neovimGetenv("XDG_DATA_HOME")); xdg != "" {
			return filepath.Join(xdg, "nvim"), nil
		}
		return filepath.Join(home, ".local", "share", "nvim"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "nvim"), nil
	case "windows":
		// Neovim uses $LOCALAPPDATA/nvim-data by default.
		if local := strings.TrimSpace(neovimGetenv("LOCALAPPDATA")); local != "" {
			return filepath.Join(local, "nvim-data"), nil
		}
		// Fallback: best-effort.
		return filepath.Join(home, "AppData", "Local", "nvim-data"), nil
	default:
		// Best-effort fallback.
		return filepath.Join(home, ".local", "share", "nvim"), nil
	}
}

func installNeovimParsersFromCache(sourceID, version string, languages []string) error {
	if !integrationEnabled("neovim") {
		return nil
	}
	dataDir, err := detectNeovimDataPath()
	if err != nil {
		return fmt.Errorf("detect neovim data path: %w", err)
	}
	destDir := filepath.Join(dataDir, "site", "parser")
	if err := neovimMkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create neovim parser dir: %w", err)
	}

	queriesRoot := filepath.Join(dataDir, "site", "queries")

	for _, lang := range languages {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}
		srcPath := TreeSitterArtifactPath(sourceID, version, lang)
		b, err := neovimReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read built parser %s: %w", lang, err)
		}
		destPath := filepath.Join(destDir, lang+SharedLibExt())
		if err := neovimWriteFile(destPath, b, 0o755); err != nil {
			return fmt.Errorf("write neovim parser %s: %w", lang, err)
		}

		cacheQueries := neovimTreeSitterQueriesCacheDir(sourceID, version, lang)
		if st, err := neovimStat(cacheQueries); err == nil && st.IsDir() {
			entries, rerr := os.ReadDir(cacheQueries)
			if rerr == nil && len(entries) > 0 {
				destQueries := filepath.Join(queriesRoot, lang)
				if err := neovimRemoveAll(destQueries); err != nil {
					return fmt.Errorf("remove stale neovim queries %s: %w", lang, err)
				}
				if err := copyNeovimTreeSitterQueriesDir(cacheQueries, destQueries, nil); err != nil {
					return fmt.Errorf("install neovim queries %s: %w", lang, err)
				}
			}
		}
	}

	AddIntegrationReportLine(sourceID, version, fmt.Sprintf("Integrated into Neovim: parsers %s, queries %s", destDir, queriesRoot))
	return nil
}

func installedTreeSitterGrammarLanguages(reg *registry_parser.RegistryParser) map[string]struct{} {
	out := map[string]struct{}{}
	for _, pkg := range local_packages_parser.GetData(false).Packages {
		item := reg.GetBySourceId(pkg.SourceID)
		if item.Source.ID == "" || !IsTreeSitterCategory(item.Categories) {
			continue
		}
		for _, l := range item.Languages {
			if s := strings.ToLower(strings.TrimSpace(l)); s != "" {
				out[s] = struct{}{}
			}
		}
		if item.TreeSitter != nil {
			for _, b := range item.TreeSitter.Build {
				if s := strings.ToLower(strings.TrimSpace(b.Language)); s != "" {
					out[s] = struct{}{}
				}
			}
		}
	}
	return out
}

func registrySourceIDsForTreeSitterLanguage(lang string, reg *registry_parser.RegistryParser) []string {
	want := strings.ToLower(strings.TrimSpace(lang))
	if want == "" {
		return nil
	}
	var out []string
	for _, item := range reg.GetData(false) {
		if item.Source.ID == "" || !IsTreeSitterCategory(item.Categories) {
			continue
		}
		ok := false
		for _, l := range item.Languages {
			if strings.ToLower(strings.TrimSpace(l)) == want {
				ok = true
				break
			}
		}
		if !ok && item.TreeSitter != nil {
			for _, b := range item.TreeSitter.Build {
				if strings.ToLower(strings.TrimSpace(b.Language)) == want {
					ok = true
					break
				}
			}
		}
		if ok {
			out = append(out, item.Source.ID)
		}
	}
	return out
}

func installRegistryTreeSitterPackagesForLanguages(
	missing []string,
	reg *registry_parser.RegistryParser,
	hint string,
) error {
	seenID := map[string]struct{}{}
	var ids []string
	var noRegistry []string
	for _, l := range missing {
		candidates := registrySourceIDsForTreeSitterLanguage(l, reg)
		if len(candidates) == 0 {
			noRegistry = append(noRegistry, l)
			continue
		}
		id := candidates[0]
		if _, dup := seenID[id]; dup {
			continue
		}
		seenID[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return fmt.Errorf(
			"no registry Tree-sitter-parser packages found for languages: %s%s",
			strings.Join(noRegistry, ", "),
			hint,
		)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ver := strings.TrimSpace(reg.GetLatestVersion(id))
		dispVer := ver
		if dispVer == "" {
			dispVer = "latest"
		}
		title := fmt.Sprintf("Installing inherited tree-sitter grammar %s@%s...", id, dispVer)

		var installFailed bool
		action := func() {
			if !Install(id, ver) {
				installFailed = true
			}
		}

		if err := spinnerutil.RunWithTTYOrPlain(title, func() {
			if NeovimInheritInstallNotifierStart != nil {
				NeovimInheritInstallNotifierStart(id, ver)
			}
		}, action); err != nil {
			return err
		}

		if installFailed {
			return fmt.Errorf("failed to install inherited grammar %s", id)
		}
		noteTreeSitterDependencyInstallSuccess()
		instVer := strings.TrimSpace(local_packages_parser.GetBySourceId(id).Version)
		if instVer == "" {
			instVer = ver
		}
		if NeovimInheritInstallNotifierDone != nil {
			NeovimInheritInstallNotifierDone(id, instVer)
		}
	}
	have := installedTreeSitterGrammarLanguages(reg)
	var still []string
	for _, l := range missing {
		if _, ok := have[strings.ToLower(strings.TrimSpace(l))]; !ok {
			still = append(still, l)
		}
	}
	if len(still) > 0 {
		msg := fmt.Sprintf(
			"after installing dependencies, still missing tree-sitter grammar(s): %s",
			strings.Join(still, ", "),
		)
		if len(noRegistry) > 0 {
			msg += fmt.Sprintf("; no registry package for: %s", strings.Join(noRegistry, ", "))
		}
		return fmt.Errorf("%s%s", msg, hint)
	}
	return nil
}

// neovimInheritContinueAnywaySourceID is set when the user chooses "Continue anyway" during
// PreflightNeovimTreeSitterInheritDeps so buildAndMaybeIntegrateTreeSitter does not prompt a second time.
var neovimInheritContinueAnywaySourceID string

// PreflightNeovimTreeSitterInheritDeps runs inherit dependency checks (prompt / nested install)
// before the CLI install spinner so prompts are not shown while the spinner claims work is in progress.
// It is a no-op unless Neovim integration is enabled and the registry item declares inherits.
func PreflightNeovimTreeSitterInheritDeps(registryItem registry_parser.RegistryItem) error {
	if registryItem.Source.ID == "" {
		return nil
	}
	if !integrationEnabled("neovim") {
		return nil
	}
	if !IsTreeSitterCategory(registryItem.Categories) {
		return nil
	}
	if registryItem.TreeSitter == nil {
		return nil
	}
	hasInherits := false
	for _, b := range registryItem.TreeSitter.Build {
		if len(b.Inherits) > 0 && TreeSitterBuildDeclaresNeovimIntegration(b) {
			hasInherits = true
			break
		}
	}
	if !hasInherits {
		return nil
	}
	if neovimInheritContinueAnywaySourceID != "" &&
		neovimInheritContinueAnywaySourceID != registryItem.Source.ID {
		neovimInheritContinueAnywaySourceID = ""
	}
	return resolveNeovimTreeSitterInheritDependencies(registryItem, func() {
		neovimInheritContinueAnywaySourceID = registryItem.Source.ID
	})
}

func resolveNeovimTreeSitterInheritDependencies(
	registryItem registry_parser.RegistryItem,
	onContinueAnyway func(),
) error {
	if !integrationEnabled("neovim") {
		return nil
	}
	if registryItem.TreeSitter == nil {
		return nil
	}
	need := map[string]struct{}{}
	for _, b := range registryItem.TreeSitter.Build {
		if !TreeSitterBuildDeclaresNeovimIntegration(b) {
			continue
		}
		for _, in := range b.Inherits {
			if s := strings.ToLower(strings.TrimSpace(in)); s != "" {
				need[s] = struct{}{}
			}
		}
	}
	if len(need) == 0 {
		return nil
	}
	reg := registry_parser.NewDefaultRegistryParser()
	have := installedTreeSitterGrammarLanguages(reg)
	var missing []string
	for l := range need {
		if _, ok := have[l]; !ok {
			missing = append(missing, l)
		}
	}
	if len(missing) == 0 {
		if registryItem.Source.ID == neovimInheritContinueAnywaySourceID {
			neovimInheritContinueAnywaySourceID = ""
		}
		return nil
	}
	sort.Strings(missing)
	var hint strings.Builder
	for _, l := range missing {
		if ids := registrySourceIDsForTreeSitterLanguage(l, reg); len(ids) > 0 {
			fmt.Fprintf(&hint, "\n• %s — e.g. zana install %s --integrate neovim", l, ids[0])
		} else {
			fmt.Fprintf(&hint, "\n• %s — install a Tree-sitter-parser package that lists this language in the registry", l)
		}
	}
	title := fmt.Sprintf("Missing base tree-sitter grammar(s) for Neovim: %s", strings.Join(missing, ", "))
	desc := "Queries may not resolve inherited captures until these are installed via Zana (Tree-sitter-parser packages whose languages include the names above)." + hint.String()
	action, err := neovimInheritsPrompt(title, desc)
	if err != nil {
		return err
	}
	switch action {
	case neovimInheritsAbort:
		return fmt.Errorf("aborted: install inherited grammar(s) first%s", hint.String())
	case neovimInheritsContinue:
		if onContinueAnyway != nil {
			onContinueAnyway()
		}
		return nil
	case neovimInheritsInstall:
		// Do not carry over a prior "continue anyway" skip from an aborted install attempt.
		neovimInheritContinueAnywaySourceID = ""
		return installRegistryTreeSitterPackagesForLanguages(missing, reg, hint.String())
	default:
		return fmt.Errorf("aborted: install inherited grammar(s) first%s", hint.String())
	}
}

func ensureNeovimTreeSitterInheritDependencies(registryItem registry_parser.RegistryItem) error {
	if registryItem.Source.ID != "" &&
		neovimInheritContinueAnywaySourceID == registryItem.Source.ID {
		neovimInheritContinueAnywaySourceID = ""
		return nil
	}
	return resolveNeovimTreeSitterInheritDependencies(registryItem, nil)
}

// buildTreeSitterOpts configures buildAndMaybeIntegrateTreeSitter for phased GitHub installs.
type buildTreeSitterOpts struct {
	SkipInheritDependencies bool
	// ExternalQueryPreflight, when non-nil, skips the external-query batch confirm in the cache step
	// and applies AllowUnpinned only to languages without a matching lock pin.
	ExternalQueryPreflight *ExternalQueryPreflightChoice
}

func buildAndMaybeIntegrateTreeSitter(repoPath string, registryItem registry_parser.RegistryItem, version string, opts *buildTreeSitterOpts) ([]local_packages_parser.TreeSitterExternalQueryPin, error) {
	if !IsTreeSitterCategory(registryItem.Categories) {
		return nil, nil
	}
	if registryItem.TreeSitter == nil || len(registryItem.TreeSitter.Build) == 0 {
		return nil, nil
	}

	if opts == nil || !opts.SkipInheritDependencies {
		if err := ensureNeovimTreeSitterInheritDependencies(registryItem); err != nil {
			return nil, err
		}
	}

	langs, err := BuildTreeSitterParsersToCache(repoPath, registryItem.Source.ID, version, registryItem.TreeSitter.Build)
	if err != nil {
		return nil, err
	}
	var pre *ExternalQueryPreflightChoice
	if opts != nil {
		pre = opts.ExternalQueryPreflight
	}
	pins, err := cacheNeovimTreeSitterQueriesForBuiltLangs(repoPath, registryItem.Source.ID, version, registryItem.TreeSitter.Build, langs, pre)
	if err != nil {
		return nil, err
	}
	neovimLangs := FilterLanguagesForNeovimTreeSitterIntegration(registryItem.TreeSitter.Build, langs)
	if err := installNeovimParsersFromCache(registryItem.Source.ID, version, neovimLangs); err != nil {
		return nil, err
	}
	return pins, nil
}

func removeNeovimTreeSitterParsers(registryItem registry_parser.RegistryItem) error {
	if !integrationEnabled("neovim") {
		return nil
	}
	if !IsTreeSitterCategory(registryItem.Categories) {
		return nil
	}
	if registryItem.TreeSitter == nil {
		return nil
	}
	if len(registryItem.TreeSitter.Build) == 0 {
		return nil
	}

	dataDir, err := detectNeovimDataPath()
	if err != nil {
		return fmt.Errorf("detect neovim data path: %w", err)
	}

	destDir := filepath.Join(dataDir, "site", "parser")
	exts := []string{".so", ".dylib", ".dll"}

	langs := map[string]struct{}{}
	for _, b := range registryItem.TreeSitter.Build {
		if !TreeSitterBuildDeclaresNeovimIntegration(b) {
			continue
		}
		if strings.TrimSpace(b.Language) != "" {
			langs[strings.TrimSpace(b.Language)] = struct{}{}
		}
	}

	queriesDir := filepath.Join(dataDir, "site", "queries")

	for lang := range langs {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}
		for _, ext := range exts {
			_ = neovimRemove(filepath.Join(destDir, lang+ext))
		}
		_ = neovimRemoveAll(filepath.Join(queriesDir, lang))
	}
	return nil
}
