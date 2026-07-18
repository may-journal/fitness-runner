package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// tempRepo writes a .gitignore with the given content into a fresh temp dir
// and returns that dir.
func tempRepo(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRun(t *testing.T) {
	cases := []struct {
		name         string
		content      string
		ok           bool
		filesChecked int
		wantErrors   []string
	}{
		{"passes when every pattern has an explanatory comment above it",
			"# dependencies\nnode_modules/\n\n# build output\ndist\n\n# keep this tracked\n!dist/keep.js\n",
			true, 1, []string{}},
		{"fails a pattern with a blank line above it",
			"# deps\nnode_modules/\n\ndist\n",
			false, 1, []string{
				`.gitignore:4: pattern "dist" has no explanatory # comment on the line above`}},
		{"fails a pattern with another pattern directly above it",
			"# deps\nnode_modules/\ndist\n",
			false, 1, []string{
				`.gitignore:3: pattern "dist" has no explanatory # comment on the line above`}},
		{"fails a leading pattern on the very first line",
			"node_modules/\n",
			false, 1, []string{
				`.gitignore:1: pattern "node_modules/" has no explanatory # comment on the line above`}},
		{"fails a pattern whose comment above is a bare # with no text",
			"#\nnode_modules/\n",
			false, 1, []string{
				`.gitignore:2: pattern "node_modules/" has no explanatory # comment on the line above`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := run(tempRepo(t, tc.content), nil)
			if err != nil {
				t.Fatal(err)
			}
			assertResult(t, res, tc.ok, tc.filesChecked, tc.wantErrors)
		})
	}
}

func TestRunMissingGitignore(t *testing.T) {
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertResult(t, res, true, 0, []string{})
}

func assertResult(t *testing.T, res checkkit.Result, ok bool, filesChecked int, wantErrors []string) {
	t.Helper()
	if res.Ok != ok {
		t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, ok, res.Errors)
	}
	if res.FilesChecked != filesChecked {
		t.Fatalf("filesChecked = %d, want %d", res.FilesChecked, filesChecked)
	}
	got := res.Errors
	if got == nil {
		got = []string{}
	}
	if !reflect.DeepEqual(got, wantErrors) {
		t.Fatalf("errors = %q, want %q", got, wantErrors)
	}
}

func TestClassifyLine(t *testing.T) {
	cases := []struct {
		line string
		want lineKind
	}{
		{"   ", kindBlank},
		{"# why", kindComment},
		{"node_modules/", kindPattern},
	}
	for _, tc := range cases {
		if got := classifyLine(tc.line); got != tc.want {
			t.Errorf("classifyLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestIsExplanatoryComment(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"# real reason", true},
		{"#", false},
		{"node_modules/", false},
	}
	for _, tc := range cases {
		if got := isExplanatoryComment(tc.line); got != tc.want {
			t.Errorf("isExplanatoryComment(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestFindViolationsCleanContent(t *testing.T) {
	if got := findViolations("# why\nnode_modules/\n"); len(got) != 0 {
		t.Fatalf("violations = %q, want none", got)
	}
}
