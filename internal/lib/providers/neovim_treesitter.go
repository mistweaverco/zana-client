package providers

import (
	"fmt"
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
	neovimUserHomeDir     = os.UserHomeDir
	neovimGetenv          = os.Getenv
	neovimReadFile        = os.ReadFile
	neovimWriteFile       = os.WriteFile
)

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
	}

	AddIntegrationReportLine(sourceID, version, fmt.Sprintf("Integrated into Neovim: %s", destDir))
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

	for lang := range langs {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}
		for _, ext := range exts {
			_ = neovimRemove(filepath.Join(destDir, lang+ext))
		}
	}
	return nil
}

