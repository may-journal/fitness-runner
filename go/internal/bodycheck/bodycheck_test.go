package bodycheck

import (
	"os"
	"path/filepath"
	"testing"
)

// pass is a validator that accepts every body.
func pass(string) []string { return nil }

// flag is a validator that rejects every body with one message.
func flag(string) []string { return []string{"nope"} }

func TestRunReadsBodyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("some body"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run([]string{"--body-file", path}, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("expected pass with 1 file, got %+v", res)
	}
}

func TestRunReadsContextMessage(t *testing.T) {
	t.Setenv("FITNESS_CTX_MESSAGE", "some body")
	res, err := Run(nil, flag)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 1 {
		t.Fatalf("expected failure with 1 file, got %+v", res)
	}
}

func TestRunPassesWithNoInput(t *testing.T) {
	if err := os.Unsetenv("FITNESS_CTX_MESSAGE"); err != nil {
		t.Fatal(err)
	}
	res, err := Run(nil, flag)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("expected inert pass with 0 files, got %+v", res)
	}
}
