package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunWiresCamelConvention(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"releaseNotes.md", "README.md", "api-design.md"} {
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
	want := "api-design.md: filename must be camelCase"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %v, want [%q]", res.Errors, want)
	}
}

func TestRunPassesOnConformingRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "releaseNotes.md"), []byte("# doc\n"), 0o644); err != nil {
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
