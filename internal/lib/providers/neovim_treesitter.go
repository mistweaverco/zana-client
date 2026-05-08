package providers

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
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

func copyNeovimTreeSitterQueriesDir(src, dst string) error {
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
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func cacheNeovimTreeSitterQueriesAfterBuild(repoPath, fullGrammarDir, sourceID, version, lang string) error {
	dest := neovimTreeSitterQueriesCacheDir(sourceID, version, lang)
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clear cached queries for %s: %w", lang, err)
	}
	if src := resolveNeovimTreeSitterQueriesDir(repoPath, fullGrammarDir); src != "" {
		if err := copyNeovimTreeSitterQueriesDir(src, dest); err != nil {
			return fmt.Errorf("cache tree-sitter queries for %s: %w", lang, err)
		}
	}
	return nil
}

func cacheNeovimTreeSitterQueriesForBuiltLangs(
	repoPath, sourceID, version string,
	build []registry_parser.RegistryItemTreeSitterBuild,
	builtLangs []string,
) error {
	want := map[string]struct{}{}
	for _, l := range builtLangs {
		l = strings.TrimSpace(l)
		if l != "" {
			want[l] = struct{}{}
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
		if err := cacheNeovimTreeSitterQueriesAfterBuild(repoPath, fullGrammarDir, sourceID, version, lang); err != nil {
			return err
		}
	}
	return nil
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
				if err := copyNeovimTreeSitterQueriesDir(cacheQueries, destQueries); err != nil {
					return fmt.Errorf("install neovim queries %s: %w", lang, err)
				}
			}
		}
	}

	AddIntegrationReportLine(sourceID, version, fmt.Sprintf("Integrated into Neovim: parsers %s, queries %s", destDir, queriesRoot))
	return nil
}

func buildAndMaybeIntegrateTreeSitter(repoPath string, registryItem registry_parser.RegistryItem, version string) error {
	if !IsTreeSitterCategory(registryItem.Categories) {
		return nil
	}
	if registryItem.TreeSitter == nil || len(registryItem.TreeSitter.Build) == 0 {
		return nil
	}

	langs, err := BuildTreeSitterParsersToCache(repoPath, registryItem.Source.ID, version, registryItem.TreeSitter.Build)
	if err != nil {
		return err
	}
	if err := cacheNeovimTreeSitterQueriesForBuiltLangs(repoPath, registryItem.Source.ID, version, registryItem.TreeSitter.Build, langs); err != nil {
		return err
	}
	return installNeovimParsersFromCache(registryItem.Source.ID, version, langs)
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
