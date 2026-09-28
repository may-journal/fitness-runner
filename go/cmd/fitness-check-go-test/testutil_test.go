package main

import (
	"os"
	"os/exec"
	"path/filepath"
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

// needGo skips a test that shells out to the go toolchain when it is absent.
func needGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not installed")
	}
}

func TestNoGoPassesClean(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "# docs only\n")
	res, err := run(root, nil)
	if err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Errorf("run = %+v, %v; want a clean pass with 0 files", res, err)
	}
}
