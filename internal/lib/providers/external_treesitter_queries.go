package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
	"golang.org/x/mod/semver"
)

// External tree-sitter query repositories (e.g. neovim-treesitter/nvim-treesitter-queries-*)
// are optional community sources. Policy controls prompts and non-interactive behavior.

type externalTreeSitterQueriesPolicyKind int

const (
	externalTreeSitterQueriesAsk externalTreeSitterQueriesPolicyKind = iota
	externalTreeSitterQueriesAlways
	externalTreeSitterQueriesNever
)

var externalTreeSitterQueriesPolicyValue = externalTreeSitterQueriesAsk

var (
	externalQueriesShellOutCapture = shell_out.ShellOutCapture
	externalQueriesGitHas          = shell_out.HasCommand
)

// externalTreeSitterQueriesConfirmHook is swapped in tests.
var externalTreeSitterQueriesConfirmHook = defaultExternalTreeSitterQueriesConfirm

func parseExternalTreeSitterQueriesPolicy(s string) (externalTreeSitterQueriesPolicyKind, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "ask":
		return externalTreeSitterQueriesAsk, nil
	case "always", "yes", "true", "1":
		return externalTreeSitterQueriesAlways, nil
	case "never", "no", "false", "0":
		return externalTreeSitterQueriesNever, nil
	default:
		return externalTreeSitterQueriesAsk, fmt.Errorf("invalid policy %q (use ask, always, or never)", s)
	}
}

// SetExternalTreeSitterQueriesPolicy configures how optional external query git clones are handled.
func SetExternalTreeSitterQueriesPolicy(s string) error {
	p, err := parseExternalTreeSitterQueriesPolicy(s)
	if err != nil {
		return err
	}
	externalTreeSitterQueriesPolicyValue = p
	return nil
}

// ConfigureExternalTreeSitterQueriesFromCLI applies the CLI flag when set; otherwise reads
// ZANA_EXTERNAL_TREESITTER_QUERIES (ask | always | never).
func ConfigureExternalTreeSitterQueriesFromCLI(flagExplicit bool, flagValue string) error {
	val := strings.TrimSpace(flagValue)
	if !flagExplicit {
		if e := strings.TrimSpace(os.Getenv("ZANA_EXTERNAL_TREESITTER_QUERIES")); e != "" {
			val = e
		}
	}
	return SetExternalTreeSitterQueriesPolicy(val)
}

type externalQueryNeed struct {
	Lang string
	URL  string
}

// plannedTreeSitterBuildLanguages returns every non-empty language from the registry build list
// (used before parsers are built, e.g. external-query preflight).
func plannedTreeSitterBuildLanguages(build []registry_parser.RegistryItemTreeSitterBuild) []string {
	var out []string
	for _, b := range build {
		if s := strings.TrimSpace(b.Language); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func collectExternalTreeSitterQueryNeeds(
	repoPath string,
	build []registry_parser.RegistryItemTreeSitterBuild,
	builtLangs []string,
) []externalQueryNeed {
	want := map[string]struct{}{}
	for _, l := range builtLangs {
		l = strings.TrimSpace(l)
		if l != "" {
			want[l] = struct{}{}
		}
	}
	var out []externalQueryNeed
	seen := map[string]struct{}{}
	for _, b := range build {
		lang := strings.TrimSpace(b.Language)
		grammarDir := strings.TrimSpace(b.GrammarDir)
		if lang == "" || grammarDir == "" {
			continue
		}
		if !TreeSitterBuildDeclaresNeovimIntegration(b) {
			continue
		}
		if _, ok := want[lang]; !ok {
			continue
		}
		if b.ExternalQueries == nil {
			continue
		}
		url := strings.TrimSpace(b.ExternalQueries.RepoURL)
		if url == "" {
			continue
		}
		// Registry external_queries always participates in confirm / policy. Upstream repos often
		// ship a minimal queries/ tree; that must not hide the optional nvim-treesitter-queries-* clone.
		if _, dup := seen[lang]; dup {
			continue
		}
		seen[lang] = struct{}{}
		out = append(out, externalQueryNeed{Lang: lang, URL: url})
	}
	return out
}

func batchConfirmExternalTreeSitterQueries(sourceID string, needs []externalQueryNeed) (bool, error) {
	if len(needs) == 0 {
		return true, nil
	}
	switch externalTreeSitterQueriesPolicyValue {
	case externalTreeSitterQueriesNever:
		return false, nil
	case externalTreeSitterQueriesAlways:
		return true, nil
	default:
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Package %s can fetch Neovim tree-sitter queries from these separate repositories (not the parser upstream):\n\n", sourceID)
	for _, n := range needs {
		fmt.Fprintf(&b, "• language %q → %s\n", n.Lang, n.URL)
	}
	b.WriteString("\nAllow Zana to git clone these repositories on your machine?")

	title := "External tree-sitter query sources"
	return externalTreeSitterQueriesConfirmHook(title, b.String())
}

func defaultExternalTreeSitterQueriesConfirm(title, description string) (bool, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stderr.Fd()) {
		return false, fmt.Errorf(
			"%s\n%s\n\nNon-interactive session: set ZANA_EXTERNAL_TREESITTER_QUERIES=always to allow these clones, or never to skip without prompting",
			title,
			description,
		)
	}
	// Default to Download (affirmative) so Enter selects the primary action.
	proceed := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Value(&proceed).
				Affirmative("Download").
				Negative("Skip (no external queries)"),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return proceed, nil
}

func canonicalSemverTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	if semver.IsValid(tag) {
		return semver.Canonical(tag)
	}
	v := "v" + strings.TrimPrefix(strings.TrimPrefix(tag, "V"), "v")
	if semver.IsValid(v) {
		return semver.Canonical(v)
	}
	return ""
}

func pickLatestSemverTag(lsRemoteOutput string) (tag string, ok bool) {
	var bestTag string
	var bestCanon string
	for _, line := range strings.Split(lsRemoteOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		const prefix = "refs/tags/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		t := strings.TrimPrefix(ref, prefix)
		if strings.HasSuffix(t, "^{}") {
			continue
		}
		canon := canonicalSemverTag(t)
		if canon == "" {
			continue
		}
		if bestCanon == "" || semver.Compare(canon, bestCanon) > 0 {
			bestCanon = canon
			bestTag = t
		}
	}
	if bestTag == "" {
		return "", false
	}
	return bestTag, true
}

func shallowCloneExternalQueriesRepo(repoURL, destDir, ref string, wantSemver bool) error {
	if !externalQueriesGitHas("git", []string{"--version"}, nil) {
		return fmt.Errorf("git not found in PATH (required to clone external tree-sitter query repositories)")
	}
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return fmt.Errorf("external queries repo_url is empty")
	}
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("remove old external queries clone: %w", err)
	}
	parent := filepath.Dir(destDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("mkdir external queries parent: %w", err)
	}

	checkout := strings.TrimSpace(ref)
	if checkout == "" && wantSemver {
		code, out, err := externalQueriesShellOutCapture("git", []string{"ls-remote", "--tags", repoURL}, "", nil)
		if err != nil || code != 0 {
			return fmt.Errorf("git ls-remote --tags %q: %w %s", repoURL, err, strings.TrimSpace(out))
		}
		var semverOK bool
		checkout, semverOK = pickLatestSemverTag(out)
		if !semverOK {
			// Many nvim-treesitter-queries-* repos only move forward on main with no semver tags;
			// fall back to the remote default branch (shallow clone without -b).
			checkout = ""
		}
	}

	args := []string{"clone", "--depth", "1"}
	if checkout != "" {
		args = append(args, "-b", checkout)
	}
	args = append(args, repoURL, destDir)

	code, out, err := externalQueriesShellOutCapture("git", args, parent, nil)
	if err != nil || code != 0 {
		return fmt.Errorf("git clone %q: %w %s", repoURL, err, strings.TrimSpace(out))
	}
	return nil
}
