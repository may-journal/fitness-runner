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

func TestRun(t *testing.T) {
	clone := cloneBlock("c", 6, 10)
	marked := "// jscpd:ignore-start\n" + clone + "// jscpd:ignore-end\n"
	cases := []struct {
		name         string
		files        map[string]string
		ok           bool
		wantErrors   []string
		filesChecked int
	}{
		{
			// 6 duplicated of 60 total lines = 10.0% > 1.0%.
			"fat cross-file clone fails in the TS format",
			map[string]string{
				"src/a.js": clone + cloneBlock("fa", 24, 3),
				"src/b.js": clone + cloneBlock("fb", 24, 3),
			},
			false,
			[]string{"ERROR: jscpd found too many duplicates (10.0%) over threshold (1.0%)"},
			2,
		},
		{
			"clone inside ignore markers passes",
			map[string]string{
				"src/a.js": marked + cloneBlock("fa", 24, 3),
				"src/b.js": marked + cloneBlock("fb", 24, 3),
			},
			true, nil, 2,
		},
		{
			"duplicate under min tokens passes",
			map[string]string{
				"src/a.js": cloneBlock("s", 6, 3),
				"src/b.js": cloneBlock("s", 6, 3),
			},
			true, nil, 2,
		},
		{
			"duplicate under min lines passes",
			map[string]string{
				"src/a.js": cloneBlock("w", 3, 20),
				"src/b.js": cloneBlock("w", 3, 20),
			},
			true, nil, 2,
		},
		{
			"ignore globs exclude tests markdown json and locks",
			map[string]string{
				"src/a.test.js": clone,
				"src/b.test.js": clone,
				"docs/a.md":     clone,
				"docs/b.md":     clone,
				"conf/a.json":   clone,
				"conf/b.json":   clone,
				"deps/a.lock":   clone,
				"deps/b.lock":   clone,
				"src/only.js":   "const onlyOne = someValue;\n",
			},
			true, nil, 1,
		},
		{
			"binary files are skipped",
			map[string]string{
				"blob.dat":    "\x00\x01\x02binary",
				"src/only.js": "const onlyOne = someValue;\n",
			},
			true, nil, 1,
		},
		{
			"empty repo passes",
			map[string]string{},
			true, nil, 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != tc.filesChecked {
				t.Errorf("filesChecked = %d, want %d", res.FilesChecked, tc.filesChecked)
			}
			if !tc.ok {
				if len(res.Errors) != 1 || res.Errors[0] != tc.wantErrors[0] {
					t.Errorf("errors = %v, want %v", res.Errors, tc.wantErrors)
				}
			}
		})
	}
}

func TestRunRespectsGitignore(t *testing.T) {
	clone := cloneBlock("c", 6, 10)
	root := writeTree(t, map[string]string{
		".gitignore":   "vendor/\n",
		"vendor/a.js":  clone,
		"vendor/b.js":  clone,
		"src/only.js":  "const onlyOne = someValue;\n",
		"src/other.js": "const otherOne = otherValue;\n",
	})
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected pass with gitignored clone, got %+v", res)
	}
	// .gitignore + the two src files; the vendor clones are filtered out.
	if res.FilesChecked != 3 {
		t.Fatalf("filesChecked = %d, want 3", res.FilesChecked)
	}
}
