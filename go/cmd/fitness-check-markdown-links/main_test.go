package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"target.md", "docs/other.md", "docs/sub/deep.md", "docs/with space.txt"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(f)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func scanDoc(t *testing.T, dir, rel, content string) []string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return scan(dir, rel, content)
}

func TestLinkResolution(t *testing.T) {
	cases := []struct {
		name    string
		rel     string
		content string
		want    []string
	}{
		{"file resolves", "a.md", "[t](target.md)", nil},
		{"percent-encoded space resolves", "a.md", "[t](docs/with%20space.txt)", nil},
		{"percent-encoded missing file is broken", "a.md", "[t](docs/no%20such.md)", []string{"a.md:1: broken relative link: docs/no%20such.md"}},
		{"dir resolves", "a.md", "[t](docs/)", nil},
		{"dot-slash resolves", "a.md", "[t](./target.md)", nil},
		{"parent traversal resolves", "docs/b.md", "[t](../target.md)", nil},
		{"sibling resolves", "docs/b.md", "[t](sub/deep.md)", nil},
		{"fragment stripped", "a.md", "[t](target.md#section)", nil},
		{"missing flagged", "a.md", "[t](gone.md)", []string{"a.md:1: broken relative link: gone.md"}},
		{"missing with fragment flagged", "a.md", "[t](gone.md#x)", []string{"a.md:1: broken relative link: gone.md#x"}},
		{"missing from subdir names file", "docs/b.md", "\n[t](nope.md)", []string{"docs/b.md:2: broken relative link: nope.md"}},
		{"image checked", "a.md", "![alt](gone.png)", []string{"a.md:1: broken relative link: gone.png"}},
		{"reference definition checked", "a.md", "[ref]: gone.md", []string{"a.md:1: broken relative link: gone.md"}},
		{"title suffix parsed", "a.md", `[t](gone.md "a title")`, []string{`a.md:1: broken relative link: gone.md`}},
		{"two links one line", "a.md", "[a](target.md) [b](gone.md)", []string{"a.md:1: broken relative link: gone.md"}},
		{"https skipped", "a.md", "[t](https://example.com/gone)", nil},
		{"mailto skipped", "a.md", "[t](mailto:x@example.com)", nil},
		{"protocol-relative skipped", "a.md", "[t](//example.com/x)", nil},
		{"fragment-only skipped", "a.md", "[t](#local-anchor)", nil},
		{"fenced code ignored", "a.md", "```\n[t](gone.md)\n```\n", nil},
		{"code span ignored", "a.md", "rewrite `[text](url)` to text", nil},
		{"empty target ignored", "a.md", "[t](#)", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := repo(t)
			got := scanDoc(t, dir, tc.rel, tc.content)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestRunEndToEnd(t *testing.T) {
	dir := repo(t)
	res := writeAndRun(t, dir, "ok.md", "[t](target.md)")
	expectCounts(t, res, true, 4, 0, "clean repo")
	res = writeAndRun(t, dir, "bad.md", "[t](gone.md)")
	expectCounts(t, res, false, 5, 1, "one broken link expected")
	if !strings.Contains(res.Errors[0], "bad.md:1: broken relative link: gone.md") {
		t.Fatalf("error format: %v", res.Errors)
	}
}

// writeAndRun adds one file to dir and runs the check.
func writeAndRun(t *testing.T, dir, name, content string) checkkit.Result {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

// expectCounts requires the verdict, file count and error count.
func expectCounts(t *testing.T, res checkkit.Result, ok bool, files, errorCount int, label string) {
	t.Helper()
	if res.Ok != ok || res.FilesChecked != files || len(res.Errors) != errorCount {
		t.Fatalf("%s: %+v", label, res)
	}
}
