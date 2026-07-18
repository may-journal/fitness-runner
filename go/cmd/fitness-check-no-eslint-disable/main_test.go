package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanContent(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		content string
		want    []string
	}{
		{"clean file", "a.ts", "const clean = 1;\nexport default clean;\n", nil},
		{"path and 1-based line per hit", "src/b.ts",
			"const ok = 1;\n// eslint-disable-next-line\nconst x = 1; // eslint-disable-line",
			[]string{
				"src/b.ts:2: eslint-disable-next-line",
				"src/b.ts:3: eslint-disable-line",
			}},
		{"block form", "a.ts", "/* eslint-disable no-console */",
			[]string{"a.ts:1: eslint-disable"}},
		{"file form", "a.ts", "/* eslint-disable */",
			[]string{"a.ts:1: eslint-disable"}},
		{"line token wins over bare disable", "a.ts", "const x = 1; // eslint-disable-line",
			[]string{"a.ts:1: eslint-disable-line"}},
		{"first match only on a multi-directive line", "a.ts",
			"// eslint-disable-next-line eslint-disable",
			[]string{"a.ts:1: eslint-disable-next-line"}},
		{"trailing newline adds no phantom line", "a.ts", "/* eslint-disable */\n",
			[]string{"a.ts:1: eslint-disable"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scanContent(tc.relPath, tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("errors = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRunPassesOnCleanDirectory(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.ts", "const a = 1;\n")
	write(t, dir, "b.js", "const b = 2;\n")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || len(res.Errors) != 0 || res.FilesChecked != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunFailsOncePerDirectiveForm(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "file.ts", "/* eslint-disable */\n")
	write(t, dir, "block.ts", "/* eslint-disable no-console */\n")
	write(t, dir, "line.ts", "const x = 1; // eslint-disable-line\n")
	write(t, dir, "next.ts", "// eslint-disable-next-line\n")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 4 {
		t.Fatalf("unexpected result: %+v", res)
	}
	want := []string{
		"block.ts:1: eslint-disable",
		"file.ts:1: eslint-disable",
		"line.ts:1: eslint-disable-line",
		"next.ts:1: eslint-disable-next-line",
	}
	if !reflect.DeepEqual(res.Errors, want) {
		t.Fatalf("errors = %v, want %v", res.Errors, want)
	}
}

func TestRunScansEveryExtensionAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range sourceExtensions {
		write(t, dir, "src/f"+ext, "// eslint-disable-next-line\n")
	}
	write(t, dir, "ignore.md", "eslint-disable everywhere\n")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != len(sourceExtensions) || len(res.Errors) != len(sourceExtensions) {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunPrunesSkipDirectories(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "node_modules/pkg/index.js", "/* eslint-disable */\n")
	write(t, dir, "dist/out.js", "/* eslint-disable */\n")
	write(t, dir, "src/clean.ts", "const a = 1;\n")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
