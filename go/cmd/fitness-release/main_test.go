package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("### 2026.10.04.0037\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_REF_NAME", "go/v0.20261004.37")
	for _, command := range []string{"tag", "verify-tag"} {
		if err := run([]string{command, "--root", root}); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
	}
}

func TestReleaseCommandErrors(t *testing.T) {
	root := t.TempDir()
	cases := [][]string{nil, {"unknown"}, {"tag", "--invalid"}, {"tag", "--root", root}, {"verify-tag", "--root", root}, {"publish", "--root", root}, {"smoke", "--root", root}, {"verify-download", "--root", root}, {"build", "--root", root}}
	for _, args := range cases {
		if err := run(args); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
