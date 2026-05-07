package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

func IsTreeSitterCategory(categories []string) bool {
	for _, c := range categories {
		if strings.EqualFold(strings.TrimSpace(c), "Tree-sitter-parser") {
			return true
		}
	}
	return false
}

func SharedLibExt() string {
	switch runtime.GOOS {
	case "darwin":
		return ".dylib"
	case "windows":
		return ".dll"
	default:
		return ".so"
	}
}

// injectable for tests
var (
	treeSitterHasCommand = shell_out.HasCommand
	treeSitterShellOut   = shell_out.ShellOut
	osMkdirAll           = func(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
)

func HasTreeSitterCLI() bool {
	return treeSitterHasCommand("tree-sitter", []string{"--version"}, nil)
}

func SafeArtifactPackageID(sourceID string) string {
	// sourceID is like "github:tree-sitter/tree-sitter-typescript"
	s := strings.TrimSpace(sourceID)
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func TreeSitterArtifactPath(sourceID, version, language string) string {
	base := filepath.Join(files.GetAppDataSharePath(), "artifacts", "treesitter", SafeArtifactPackageID(sourceID), version)
	return filepath.Join(base, language+SharedLibExt())
}

// buildTreeSitterParsersToCache builds tree-sitter parser shared libraries from
// upstream source into Zana's artifact cache and returns the list of languages
// that were built.
func BuildTreeSitterParsersToCache(
	repoPath string,
	sourceID string,
	version string,
	build []registry_parser.RegistryItemTreeSitterBuild,
) ([]string, error) {
	if len(build) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	if !HasTreeSitterCLI() {
		return nil, fmt.Errorf("tree-sitter CLI not found in PATH (required to build parsers from source)")
	}

	var built []string
	for _, b := range build {
		lang := strings.TrimSpace(b.Language)
		grammarDir := strings.TrimSpace(b.GrammarDir)
		if lang == "" || grammarDir == "" {
			continue
		}

		outPath := TreeSitterArtifactPath(sourceID, version, lang)
		if err := osMkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return nil, fmt.Errorf("create artifact dir: %w", err)
		}

		fullGrammarDir := filepath.Join(repoPath, filepath.FromSlash(grammarDir))
		code, err := treeSitterShellOut("tree-sitter", []string{"build", "-o", outPath, fullGrammarDir}, "", nil)
		if err != nil || code != 0 {
			return nil, fmt.Errorf("tree-sitter build failed for %s in %s", lang, grammarDir)
		}

		built = append(built, lang)
	}

	return built, nil
}

