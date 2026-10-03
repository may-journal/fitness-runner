package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// runOn writes each named markdown file into a temp root and runs the check.
func runOn(t *testing.T, names []string) checkkit.Result {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# doc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRunWiresCamelConvention(t *testing.T) {
	cases := []struct {
		name       string
		files      []string
		ok         bool
		filesCount int
		wantErrors []string
	}{
		{"flags the nonconforming file", []string{"releaseNotes.md", "installGuide.md", "README.md", "api-design.md"}, false, 4, []string{"api-design.md: filename must be camelCase"}},
		{"passes on a conforming repo", []string{"releaseNotes.md"}, true, 1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runOn(t, tc.files)
			if res.Ok != tc.ok || res.FilesChecked != tc.filesCount || !slices.Equal(res.Errors, tc.wantErrors) {
				t.Fatalf("unexpected result: %+v, want errors %q", res, tc.wantErrors)
			}
		})
	}
}
