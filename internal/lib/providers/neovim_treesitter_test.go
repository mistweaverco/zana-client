package providers

import (
	"os"
	"path/filepath"
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
	if err := copyNeovimTreeSitterQueriesDir(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "nested", "injections.scm"))
	if err != nil || string(b) != "y" {
		t.Fatalf("nested copy: %v %q", err, b)
	}
}
