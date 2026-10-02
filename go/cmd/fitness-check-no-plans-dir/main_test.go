package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()

	// No docs/plans directory: passes with zero files.
	runExpecting(t, dir, true, 0, "clean repo")

	// A file under docs/plans: fails, one error per file, with guidance.
	writePlans(t, dir, "03-publish.md", "archive/old.md")
	res := runExpecting(t, dir, false, 2, "expected 2 offenders")
	if !strings.Contains(res.Errors[0], "docs/plans/") || !strings.Contains(res.Errors[0], "open a Plan Issue") {
		t.Fatalf("error should name the file and the fix: %q", res.Errors[0])
	}
}

// writePlans creates each named file under docs/plans, making directories.
func writePlans(t *testing.T, dir string, names ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "plans", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		p := filepath.Join(dir, "docs", "plans", filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runExpecting runs the check and requires the verdict and file count.
func runExpecting(t *testing.T, dir string, ok bool, files int, label string) checkkit.Result {
	t.Helper()
	res, err := run(dir, nil)
	if err != nil || res.Ok != ok || res.FilesChecked != files {
		t.Fatalf("%s: %+v %v", label, res, err)
	}
	return res
}
