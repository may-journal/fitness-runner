package walkfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestFilesByExtOutsideGitWalksEverything(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"root.md", "src/a.md", "src/c.ts", "node_modules/pkg/x.md",
		"dist/y.md", "coverage/z.md", "githooks/h.md", ".git/HEAD.md", "sub/.git/k.md"} {
		write(t, root, rel, "")
	}
	write(t, root, ".fitnessrc.json", `{"ignore": ["src"]}`)
	write(t, root, "cspell.json", `{"ignorePaths": ["dist"]}`)
	cases := []struct {
		exts []string
		want []string
	}{
		{[]string{".md"}, []string{"coverage/z.md", "dist/y.md", "githooks/h.md", "node_modules/pkg/x.md", "root.md", "src/a.md"}},
		{[]string{".ts"}, []string{"src/c.ts"}},
		{[]string{"x.custom.md", ".ts"}, []string{"src/c.ts"}},
	}
	for _, tc := range cases {
		if got := FilesByExt(root, tc.exts...); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("FilesByExt(%v) = %v, want %v", tc.exts, got, tc.want)
		}
	}
}

func TestFilesByExtSuffixMatch(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.custom.md", "")
	got := FilesByExt(root, ".md")
	if !reflect.DeepEqual(got, []string{"x.custom.md"}) {
		t.Fatalf("suffix match failed: %v", got)
	}
}

func TestFilesByExtInGitListsTrackedFiles(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, ".gitignore", "dist\nforced.md\n")
	for _, rel := range []string{"a.md", "node_modules/p/b.md", "githooks/c.md", "gone.md", "untracked.md", "dist/d.md", "forced.md"} {
		write(t, root, rel, "")
	}
	git(t, root, "add", ".gitignore", "a.md", "node_modules", "githooks", "gone.md")
	git(t, root, "add", "-f", "forced.md")
	must(t, os.Symlink("a.md", filepath.Join(root, "link.md")))
	git(t, root, "add", "link.md")
	must(t, os.Remove(filepath.Join(root, "gone.md")))
	want := []string{"a.md", "forced.md", "githooks/c.md", "node_modules/p/b.md"}
	if got := FilesByExt(root, ".md"); !reflect.DeepEqual(got, want) {
		t.Errorf("FilesByExt = %v, want %v", got, want)
	}
	if got := FilesByExt(filepath.Join(root, "githooks"), ".md"); !reflect.DeepEqual(got, []string{"c.md"}) {
		t.Errorf("FilesByExt(subdir) = %v, want [c.md]", got)
	}
}

func TestInScope(t *testing.T) {
	files := []string{"a.md", "docs/b.md", "c.md"}
	t.Setenv("FITNESS_CHANGED_FILES", "")
	if got := InScope(files); !reflect.DeepEqual(got, files) {
		t.Fatalf("unscoped = %v, want every file", got)
	}
	t.Setenv("FITNESS_CHANGED_FILES", "docs/b.md\ngone.md\nmain.go")
	if got := InScope(files); !reflect.DeepEqual(got, []string{"docs/b.md"}) {
		t.Fatalf("scoped = %v, want [docs/b.md]", got)
	}
}

func TestScanFilesScoped(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", "x")
	write(t, root, "b.md", "x")
	t.Setenv("FITNESS_CHANGED_FILES", "b.md")
	var seen []string
	_, n, err := ScanFiles(root, []string{".md"}, func(rel, _ string) []string {
		seen = append(seen, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || !reflect.DeepEqual(seen, []string{"b.md"}) {
		t.Fatalf("scanned %v (n=%d), want only b.md", seen, n)
	}
}

// must fails the test on a setup error.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
