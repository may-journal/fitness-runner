package toolchain

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestModulesSkipsFixtures(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module a\n")
	write(t, root, "tools/go.mod", "module b\n")
	write(t, root, "x/testdata/go.mod", "module c\n")
	write(t, root, "vendor/d/go.mod", "module d\n")
	write(t, root, "readme.txt", "")
	got := Modules(root)
	want := []string{".", "tools"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Modules = %v, want %v", got, want)
	}
}

func TestLabelAndLines(t *testing.T) {
	if Label(".", "m") != "m" || Label("tools", "m") != "tools: m" {
		t.Error("Label must prefix only non-root modules")
	}
	got := Lines("a\n\n  \r\nb\r\n")
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("Lines = %q", got)
	}
}

func TestCleanEnvDropsGitHookVars(t *testing.T) {
	got := CleanEnv([]string{"PATH=/bin", "GIT_DIR=/repo/.git", "GIT_INDEX_FILE=/repo/.git/index", "GIT_AUTHOR_NAME=bot", "GOFLAGS=-mod=mod"})
	want := []string{"PATH=/bin", "GIT_AUTHOR_NAME=bot", "GOFLAGS=-mod=mod"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CleanEnv = %v, want %v", got, want)
	}
}

func TestCleanEnvDropsRunnerContext(t *testing.T) {
	got := CleanEnv([]string{"PATH=/bin", "FITNESS_CHANGED_FILES=a.md", "FITNESS_STAGED_FILES=a.md"})
	if want := []string{"PATH=/bin"}; !reflect.DeepEqual(got, want) {
		t.Errorf("CleanEnv = %v, want %v", got, want)
	}
}

func TestUnchanged(t *testing.T) {
	cases := []struct {
		name, changed string
		want          bool
	}{
		{"unscoped run", "", false},
		{"docs only", "README.md\ndocs/a.md", true},
		{"go source", "README.md\ngo/cmd/x/main.go", false},
		{"go.mod", "go/go.mod", false},
		{"go.sum", "go/go.sum", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FITNESS_CHANGED_FILES", tc.changed)
			if got := Unchanged(); got != tc.want {
				t.Fatalf("Unchanged() = %v, want %v", got, tc.want)
			}
		})
	}
}
