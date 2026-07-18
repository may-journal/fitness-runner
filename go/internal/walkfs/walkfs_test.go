package walkfs

import (
	"os"
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

func TestSkipDirsDefaultsAndCspell(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cspell.json", `{"ignorePaths": ["node_modules", "custom-skip", "**/glob/**", "with/slash"]}`)
	set := SkipDirs(root)
	for _, want := range []string{"node_modules", "dist", "coverage", ".git", "githooks", "custom-skip"} {
		if !set[want] {
			t.Errorf("missing skip dir %q", want)
		}
	}
	if set["**/glob/**"] || set["with/slash"] {
		t.Error("glob/slash entries must not join the skip set")
	}
}

func TestSkipDirsBadCspellIgnored(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cspell.json", "not json at all")
	if len(SkipDirs(root)) != 5 {
		t.Error("bad cspell.json must contribute nothing")
	}
}

func TestFilesByExt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "root.md", "")
	write(t, root, "src/a.md", "")
	write(t, root, "src/deep/b.md", "")
	write(t, root, "src/c.ts", "")
	write(t, root, "node_modules/pkg/x.md", "")
	write(t, root, "dist/y.md", "")
	write(t, root, "custom-skip/z.md", "")
	write(t, root, "cspell.json", `{"ignorePaths": ["custom-skip"]}`)

	got := FilesByExt(root, ".md")
	want := []string{"root.md", "src/a.md", "src/deep/b.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	both := FilesByExt(root, ".md", ".ts")
	wantBoth := []string{"root.md", "src/a.md", "src/c.ts", "src/deep/b.md"}
	if !reflect.DeepEqual(both, wantBoth) {
		t.Fatalf("got %v, want %v", both, wantBoth)
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
