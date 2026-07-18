package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunWiresKebabConvention(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"api-design.md", "README.md", "releaseNotes.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# doc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
	want := "releaseNotes.md: filename must be kebab-case"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %v, want [%q]", res.Errors, want)
	}
}

func TestRunPassesOnConformingRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-design.md"), []byte("# doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 || len(res.Errors) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
