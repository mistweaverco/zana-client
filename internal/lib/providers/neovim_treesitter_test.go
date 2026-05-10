package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNeovimTreeSitterQueriesDir_PrefersGrammarLocal(t *testing.T) {
	repo := t.TempDir()
	gram := filepath.Join(repo, "g")
	if err := os.MkdirAll(filepath.Join(gram, "queries"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gram, "queries", "highlights.scm"), []byte("(a)"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "queries"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "queries", "highlights.scm"), []byte("(b)"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolveNeovimTreeSitterQueriesDir(repo, gram)
	want := filepath.Join(gram, "queries")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCopyNeovimTreeSitterQueriesDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "highlights.scm"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "injections.scm"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyNeovimTreeSitterQueriesDir(src, dst, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "nested", "injections.scm"))
	if err != nil || string(b) != "y" {
		t.Fatalf("nested copy: %v %q", err, b)
	}
}

func TestCopyNeovimTreeSitterQueriesDir_InheritsModelineAndNotEq(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "highlights.scm"), []byte(`(#is-not? @x "y")`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyNeovimTreeSitterQueriesDir(src, dst, []string{"javascript"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "highlights.scm"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.HasPrefix(got, "; inherits: javascript\n") {
		t.Fatalf("want inherits modeline first, got %q", got)
	}
	if !strings.Contains(got, "#not-eq?") || strings.Contains(got, "#is-not?") {
		t.Fatalf("replace #is-not?: %q", got)
	}
}

func TestCopyNeovimTreeSitterQueriesDir_SkipsModelineWhenPresent(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	orig := "; inherits: ecma\n(#is-not? @a \"b\")\n"
	if err := os.WriteFile(filepath.Join(src, "highlights.scm"), []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyNeovimTreeSitterQueriesDir(src, dst, []string{"javascript"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "highlights.scm"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Count(got, "inherits:") != 1 {
		t.Fatalf("should not duplicate inherits modeline: %q", got)
	}
	if !strings.Contains(got, "#not-eq?") {
		t.Fatal("expected #is-not? rewritten")
	}
}
