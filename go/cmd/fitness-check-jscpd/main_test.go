package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cloneBlock builds lines of distinct identifiers so a shared prefix
// yields an exact duplicate and different prefixes never match: 6 lines
// with 10 identifiers plus ";" is 66 tokens — over the 50-token minimum.
func cloneBlock(prefix string, lines, perLine int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < perLine; j++ {
			fmt.Fprintf(&b, "%s_%d_%d ", prefix, i, j)
		}
		b.WriteString(";\n")
	}
	return b.String()
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// verdict is what a case expects: an empty wantError means a pass.
type verdict struct {
	wantError    string
	filesChecked int
}

// tooMany is the failure message for a duplicate percentage.
func tooMany(pct string) string {
	return "ERROR: jscpd found too many duplicates (" + pct + "%) over threshold (1.0%)"
}

// assertVerdict runs the check on root and compares it with want.
func assertVerdict(t *testing.T, root string, want verdict) {
	t.Helper()
	res, err := run(root, nil)
	if err != nil || res.FilesChecked != want.filesChecked {
		t.Fatalf("filesChecked = %d (err %v), want %d", res.FilesChecked, err, want.filesChecked)
	}
	got := strings.Join(res.Errors, "\n")
	if res.Ok != (want.wantError == "") || got != want.wantError {
		t.Fatalf("ok = %v, errors = %q, want %q", res.Ok, got, want.wantError)
	}
}

func TestRun(t *testing.T) {
	clone := cloneBlock("c", 6, 10)
	marked := "// jscpd:ignore-start\n" + clone + "// jscpd:ignore-end\n"
	cases := []struct {
		name  string
		files map[string]string
		want  verdict
	}{
		{
			// 6 duplicated of 60 total lines = 10.0% > 1.0%.
			"fat cross-file clone fails in the TS format",
			map[string]string{
				"src/a.js": clone + cloneBlock("fa", 24, 3),
				"src/b.js": clone + cloneBlock("fb", 24, 3),
			},
			verdict{tooMany("10.0"), 2},
		},
		{
			"clone inside ignore markers passes",
			map[string]string{
				"src/a.js": marked + cloneBlock("fa", 24, 3),
				"src/b.js": marked + cloneBlock("fb", 24, 3),
			},
			verdict{"", 2},
		},
		{
			"duplicate under min tokens passes",
			map[string]string{"src/a.js": cloneBlock("s", 6, 3), "src/b.js": cloneBlock("s", 6, 3)},
			verdict{"", 2},
		},
		{
			"duplicate under min lines passes",
			map[string]string{"src/a.js": cloneBlock("w", 3, 20), "src/b.js": cloneBlock("w", 3, 20)},
			verdict{"", 2},
		},
		{
			"tests, markdown, json, and locks are compared too",
			map[string]string{
				"src/a_test.go": clone + cloneBlock("ta", 24, 3),
				"docs/b.md":     clone + cloneBlock("tb", 24, 3),
				"conf/c.json":   cloneBlock("tc", 30, 3),
				"deps/d.lock":   cloneBlock("td", 30, 3),
			},
			verdict{tooMany("5.0"), 4},
		},
		{
			"binary files are skipped",
			map[string]string{
				"blob.dat":    "\x00\x01\x02 binary",
				"src/only.js": "const onlyOne = someValue;\n",
			},
			verdict{"", 1},
		},
		{"empty repo passes", map[string]string{}, verdict{"", 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertVerdict(t, writeTree(t, tc.files), tc.want)
		})
	}
}

// TestRunScansTrackedFilesOnly proves a git repo's tracked files are
// compared even when gitignored, while untracked files and files marked
// linguist-generated are left out.
func TestRunScansTrackedFilesOnly(t *testing.T) {
	clone := cloneBlock("c", 6, 10)
	root := writeTree(t, map[string]string{
		".gitignore":     "vendor/\n",
		".gitattributes": "*.lock linguist-generated\n",
		"deps/a.lock":    clone + cloneBlock("la", 24, 3),
		"deps/b.lock":    clone + cloneBlock("lb", 24, 3),
		"vendor/a.js":    clone + cloneBlock("va", 24, 3),
		"src/b.js":       clone + cloneBlock("vb", 24, 3),
		"untracked/c.js": clone,
		"untracked/d.js": clone,
		"untracked/e.js": clone,
	})
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", ".gitignore", ".gitattributes", "deps", "vendor", "src"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	// .gitignore, .gitattributes, and the two tracked clones; the generated
	// lock files are left out: 6 of 62 lines duplicated.
	assertVerdict(t, root, verdict{tooMany("9.7"), 4})
}
