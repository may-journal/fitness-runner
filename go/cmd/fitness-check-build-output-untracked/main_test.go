package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// initRepo creates a temp git repo; when ignoreDist is set, .gitignore holds
// a dist rule so `git check-ignore -q dist` succeeds.
func initRepo(t *testing.T, ignoreDist bool) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	if ignoreDist {
		writeFile(t, dir, ".gitignore", "dist\n")
	}
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// trackFile writes a file and force-adds it past any ignore rule.
func trackFile(t *testing.T, dir, rel string) {
	t.Helper()
	writeFile(t, dir, rel, "tracked\n")
	git(t, dir, "add", "-f", rel)
}

func TestScanFileForDistImports(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		content string
		want    []string
	}{
		{
			name:    "flags from/import/require specifiers into dist with path:line",
			relPath: "src/x.ts",
			content: "import ok from './local.js';\n" +
				"import bad from '../dist/foo.js';\n" +
				"const z = require('pkg/dist/z.js');\n" +
				"export * from 'dist/root.js';\n" +
				"const w = await import('normal-pkg');",
			want: []string{
				`src/x.ts:2 imports build output: "../dist/foo.js"`,
				`src/x.ts:3 imports build output: "pkg/dist/z.js"`,
				`src/x.ts:4 imports build output: "dist/root.js"`,
			},
		},
		{
			name:    "returns nothing for clean source",
			relPath: "a.ts",
			content: "import x from './x.js';\n",
			want:    nil,
		},
		{
			name:    "bare ../dist with no trailing slash is flagged",
			relPath: "a.ts",
			content: "import x from '../dist';\n",
			want:    []string{`a.ts:1 imports build output: "../dist"`},
		},
		{
			name:    "dist as a name fragment is not flagged",
			relPath: "a.ts",
			content: "import x from 'mydist/x.js';\nimport y from './distant/y.js';\n",
			want:    nil,
		},
		{
			name:    "keyword needs a word boundary",
			relPath: "a.ts",
			content: "const reimport = 'dist/x.js';\n",
			want:    nil,
		},
		{
			name:    "two dist specifiers on one line report in order",
			relPath: "a.ts",
			content: "import a from 'dist/a.js'; const b = require('x/dist/b.js');\n",
			want: []string{
				`a.ts:1 imports build output: "dist/a.js"`,
				`a.ts:1 imports build output: "x/dist/b.js"`,
			},
		},
		{
			name:    "dynamic import into dist is flagged",
			relPath: "a.ts",
			content: "const m = await import('../dist/mod.js');\n",
			want:    []string{`a.ts:1 imports build output: "../dist/mod.js"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scanFileForDistImports(tc.relPath, tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("errors = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestCollectSourceFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"a.ts", "b.mts", "c.cts", "d.txt",
		"a.test.ts", "b.spec.mts", "c.test.cts",
		"dist/generated.ts", "nested/deep.ts",
	} {
		writeFile(t, dir, name, "")
	}
	got := collectSourceFiles(dir)
	want := []string{"a.ts", "b.mts", "c.cts", "nested/deep.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestDedupeSorted(t *testing.T) {
	got := dedupeSorted([]string{"dist/a.js"}, []string{"dist/a.js", "sub/dist/b.js"})
	want := []string{"dist/a.js", "sub/dist/b.js"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deduped = %#v, want %#v", got, want)
	}
}

func TestDistTrackingErrors(t *testing.T) {
	t.Run("empty when dist is ignored and untracked", func(t *testing.T) {
		dir := initRepo(t, true)
		if got := distTrackingErrors(dir); len(got) != 0 {
			t.Fatalf("errors = %#v, want none", got)
		}
	})

	t.Run("flags a missing gitignore rule", func(t *testing.T) {
		dir := initRepo(t, false)
		if got := distTrackingErrors(dir); !reflect.DeepEqual(got, []string{msgNotIgnored}) {
			t.Fatalf("errors = %#v, want %#v", got, []string{msgNotIgnored})
		}
	})

	t.Run("flags nested tracked dist files with a git rm hint", func(t *testing.T) {
		dir := initRepo(t, true)
		trackFile(t, dir, "sub/dist/b.js")
		want := []string{"Remove tracked files: sub/dist/b.js (git rm --cached)"}
		if got := distTrackingErrors(dir); !reflect.DeepEqual(got, want) {
			t.Fatalf("errors = %#v, want %#v", got, want)
		}
	})

	t.Run("reports both rules at once, sorted and combined", func(t *testing.T) {
		dir := initRepo(t, false)
		trackFile(t, dir, "sub/dist/b.js")
		trackFile(t, dir, "dist/a.js")
		want := []string{
			msgNotIgnored,
			"Remove tracked files: dist/a.js, sub/dist/b.js (git rm --cached)",
		}
		if got := distTrackingErrors(dir); !reflect.DeepEqual(got, want) {
			t.Fatalf("errors = %#v, want %#v", got, want)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("passes on a clean repo and counts files + 1", func(t *testing.T) {
		dir := initRepo(t, true)
		writeFile(t, dir, "clean.ts", "import x from './local.js';\n")
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Ok || len(res.Errors) != 0 || res.FilesChecked != 2 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("fails on a dist import in source", func(t *testing.T) {
		dir := initRepo(t, true)
		writeFile(t, dir, "bad.ts", "import x from '../dist/x.js';\n")
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{`bad.ts:1 imports build output: "../dist/x.js"`}
		if res.Ok || !reflect.DeepEqual(res.Errors, want) || res.FilesChecked != 2 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("orders tracking errors before import errors", func(t *testing.T) {
		dir := initRepo(t, false)
		trackFile(t, dir, "dist/build.js")
		writeFile(t, dir, "bad.ts", "import x from '../dist/x.js';\n")
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			msgNotIgnored,
			"Remove tracked files: dist/build.js (git rm --cached)",
			`bad.ts:1 imports build output: "../dist/x.js"`,
		}
		if res.Ok || !reflect.DeepEqual(res.Errors, want) || res.FilesChecked != 2 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}
