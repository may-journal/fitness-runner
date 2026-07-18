package clonedetect

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T, gitignore string) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	if gitignore != "" {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGitIgnored(t *testing.T) {
	t.Run("filters ignored paths in batch", func(t *testing.T) {
		dir := initRepo(t, "ignored.txt\nbuild/\n")
		got := GitIgnored(dir, []string{"ignored.txt", "kept.txt", "build/out.js", "src/kept.go"})
		want := map[string]bool{"ignored.txt": true, "build/out.js": true}
		if len(got) != len(want) {
			t.Fatalf("ignored = %v, want %v", got, want)
		}
		for p := range want {
			if !got[p] {
				t.Fatalf("expected %q ignored; got %v", p, got)
			}
		}
	})

	t.Run("nothing ignored yields empty map", func(t *testing.T) {
		dir := initRepo(t, "ignored.txt\n")
		got := GitIgnored(dir, []string{"kept.txt", "also-kept.go"})
		if got == nil || len(got) != 0 {
			t.Fatalf("want empty non-nil map, got %v", got)
		}
	})

	t.Run("outside a git repo returns nil", func(t *testing.T) {
		if got := GitIgnored(t.TempDir(), []string{"a.txt"}); got != nil {
			t.Fatalf("want nil outside a repo, got %v", got)
		}
	})

	t.Run("no paths asks git nothing", func(t *testing.T) {
		if got := GitIgnored(t.TempDir(), nil); got == nil || len(got) != 0 {
			t.Fatalf("want empty map for no paths, got %v", got)
		}
	})
}
