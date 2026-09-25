package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()

	// No docs/plans directory: passes with zero files.
	if res, err := run(dir, nil); err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("clean repo: %+v %v", res, err)
	}

	// A file under docs/plans: fails, one error per file, with guidance.
	plans := filepath.Join(dir, "docs", "plans", "archive")
	if err := os.MkdirAll(plans, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"03-publish.md", "archive/old.md"} {
		p := filepath.Join(dir, "docs", "plans", filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 2 {
		t.Fatalf("expected 2 offenders, got %+v", res)
	}
	if !strings.Contains(res.Errors[0], "docs/plans/") || !strings.Contains(res.Errors[0], "open a Plan Issue") {
		t.Fatalf("error should name the file and the fix: %q", res.Errors[0])
	}
}
