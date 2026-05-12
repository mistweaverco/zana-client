package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/stretchr/testify/require"
)

func TestPickLatestSemverTag(t *testing.T) {
	in := "" +
		"a1\trefs/tags/v0.1.0\n" +
		"a2\trefs/tags/v0.2.0\n" +
		"a3\trefs/tags/0.3.0\n" +
		"x\trefs/tags/not-semver\n"
	tag, ok := pickLatestSemverTag(in)
	require.True(t, ok)
	require.Equal(t, "0.3.0", tag)

	_, ok = pickLatestSemverTag("dead\trefs/tags/not-semver\n")
	require.False(t, ok)
}

func TestCollectExternalTreeSitterQueryNeeds_IncludesWhenRegistryDeclaresExternalDespiteUpstreamQueries(t *testing.T) {
	repo := t.TempDir()
	gram := filepath.Join(repo, "g")
	require.NoError(t, os.MkdirAll(filepath.Join(gram, "queries"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gram, "queries", "highlights.scm"), []byte("()"), 0o644))

	build := []registry_parser.RegistryItemTreeSitterBuild{
		{
			Language:     "hcl",
			GrammarDir:   "g",
			Integrations: []string{"neovim"},
			ExternalQueries: &registry_parser.RegistryItemTreeSitterExternalQueries{
				RepoURL: "https://github.com/neovim-treesitter/nvim-treesitter-queries-hcl",
				Semver:  true,
			},
		},
	}
	got := collectExternalTreeSitterQueryNeeds(repo, build, []string{"hcl"})
	require.Len(t, got, 1)
	require.Equal(t, "hcl", got[0].Lang)
	require.Contains(t, got[0].URL, "nvim-treesitter-queries-hcl")
}

func TestCollectExternalTreeSitterQueryNeeds_IncludesWhenMissingQueries(t *testing.T) {
	repo := t.TempDir()
	gram := filepath.Join(repo, "g")
	require.NoError(t, os.MkdirAll(gram, 0o755))

	build := []registry_parser.RegistryItemTreeSitterBuild{
		{
			Language:     "hcl",
			GrammarDir:   "g",
			Integrations: []string{"neovim"},
			ExternalQueries: &registry_parser.RegistryItemTreeSitterExternalQueries{
				RepoURL: "https://example.com/nvim-treesitter-queries-hcl",
			},
		},
	}
	got := collectExternalTreeSitterQueryNeeds(repo, build, []string{"hcl"})
	require.Len(t, got, 1)
	require.Equal(t, "hcl", got[0].Lang)
	require.Equal(t, "https://example.com/nvim-treesitter-queries-hcl", got[0].URL)
}

func TestCollectExternalTreeSitterQueryNeeds_SkipsWhenBuildDoesNotTargetNeovim(t *testing.T) {
	repo := t.TempDir()
	gram := filepath.Join(repo, "g")
	require.NoError(t, os.MkdirAll(gram, 0o755))

	build := []registry_parser.RegistryItemTreeSitterBuild{
		{
			Language:     "hcl",
			GrammarDir:   "g",
			Integrations: []string{"vscode"},
			ExternalQueries: &registry_parser.RegistryItemTreeSitterExternalQueries{
				RepoURL: "https://example.com/nvim-treesitter-queries-hcl",
			},
		},
	}
	got := collectExternalTreeSitterQueryNeeds(repo, build, []string{"hcl"})
	require.Empty(t, got)
}

func TestBatchConfirmExternalTreeSitterQueries_PolicyNever(t *testing.T) {
	t.Cleanup(func() { _ = SetExternalTreeSitterQueriesPolicy("ask") })
	require.NoError(t, SetExternalTreeSitterQueriesPolicy("never"))

	prev := externalTreeSitterQueriesConfirmHook
	externalTreeSitterQueriesConfirmHook = func(_, _ string) (bool, error) {
		t.Fatal("confirm hook should not run when policy is never")
		return false, nil
	}
	t.Cleanup(func() { externalTreeSitterQueriesConfirmHook = prev })

	ok, err := batchConfirmExternalTreeSitterQueries("github:demo/pkg", []externalQueryNeed{{Lang: "hcl", URL: "https://example.com/q"}})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestParseExternalTreeSitterQueriesPolicy_Invalid(t *testing.T) {
	_, err := parseExternalTreeSitterQueriesPolicy("sometimes")
	require.Error(t, err)
}

func TestExternalQueryNeedsStillRequiringConfirm_FiltersLockPinned(t *testing.T) {
	prev := externalQueryLockPinForConfirmFilter
	externalQueryLockPinForConfirmFilter = func(sourceID, version, lang string) (string, string, bool) {
		if lang == "hcl" {
			return "https://example.com/nvim-treesitter-queries-hcl", "abc123def456", true
		}
		return "", "", false
	}
	t.Cleanup(func() { externalQueryLockPinForConfirmFilter = prev })

	needs := []externalQueryNeed{
		{Lang: "hcl", URL: "https://example.com/nvim-treesitter-queries-hcl"},
		{Lang: "lua", URL: "https://example.com/other-queries"},
	}
	got := externalQueryNeedsStillRequiringConfirm("github:demo/grammar", "v1.0.0", needs)
	require.Len(t, got, 1)
	require.Equal(t, "lua", got[0].Lang)
}

func TestExternalQueryLockCoversNeed_URLMismatchStillRequiresConfirm(t *testing.T) {
	prev := externalQueryLockPinForConfirmFilter
	externalQueryLockPinForConfirmFilter = func(_, _, lang string) (string, string, bool) {
		if lang == "hcl" {
			return "https://example.com/different-repo", "abc123", true
		}
		return "", "", false
	}
	t.Cleanup(func() { externalQueryLockPinForConfirmFilter = prev })

	n := externalQueryNeed{Lang: "hcl", URL: "https://example.com/nvim-treesitter-queries-hcl"}
	require.False(t, externalQueryLockCoversNeed("github:demo/grammar", "v1.0.0", n))
}

func TestBatchConfirmExternalTreeSitterQueries_SkippedWhenNeedsEmptyAfterLockFilter(t *testing.T) {
	t.Cleanup(func() { _ = SetExternalTreeSitterQueriesPolicy("ask") })
	require.NoError(t, SetExternalTreeSitterQueriesPolicy("ask"))

	prev := externalTreeSitterQueriesConfirmHook
	externalTreeSitterQueriesConfirmHook = func(_, _ string) (bool, error) {
		t.Fatal("confirm hook should not run when every need is lock-pinned")
		return false, nil
	}
	t.Cleanup(func() { externalTreeSitterQueriesConfirmHook = prev })

	prevPin := externalQueryLockPinForConfirmFilter
	externalQueryLockPinForConfirmFilter = func(_, _, lang string) (string, string, bool) {
		if lang == "hcl" {
			return "https://example.com/q", "deadbeef", true
		}
		return "", "", false
	}
	t.Cleanup(func() { externalQueryLockPinForConfirmFilter = prevPin })

	needs := []externalQueryNeed{{Lang: "hcl", URL: "https://example.com/q"}}
	confirm := externalQueryNeedsStillRequiringConfirm("github:demo/pkg", "v1", needs)
	require.Empty(t, confirm)

	ok, err := batchConfirmExternalTreeSitterQueries("github:demo/pkg", confirm)
	require.NoError(t, err)
	require.True(t, ok)
}
