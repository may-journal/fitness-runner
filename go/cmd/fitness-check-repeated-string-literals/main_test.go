package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// values collects just the values scanned from source, in order.
func values(src string) []string {
	var out []string
	for _, lit := range scanStringLiterals(src) {
		out = append(out, lit.value)
	}
	return out
}

// tempRepo builds a fresh temp dir and writes files (relPath -> content)
// into it.
func tempRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScanStringLiterals(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"captures single- and double-quoted literals",
			"const a = 'pending';\nconst b = \"active\";", []string{"pending", "active"}},
		{"ignores strings inside line and block comments",
			"// 'pending' here\n/* 'active' too */\nconst x = 'kept';", []string{"kept"}},
		{"skips module specifiers (import/require/from)",
			"import x from 'node:fs';\n" +
				"export { y } from 'node:path';\n" +
				"const z = require('node:os');\n" +
				"const d = await import('node:url');\n" +
				"const keep = 'pending';",
			[]string{"pending"}},
		{"does not mistake a regex literal for a string",
			// The quote chars live inside a regex character class, not a string.
			"const re = /['\"]/g;\nconst keep = 'active';", []string{"active"}},
		{"treats / after a value as division, not a regex",
			"const r = a / b / c;\nconst keep = 'active';", []string{"active"}},
		{"skips template literals in v1",
			"const t = `hello ${name}`;\nconst keep = 'active';", []string{"active"}},
		{"honors backslash escapes inside strings",
			"const a = 'it\\'s fine';", []string{"it's fine"}},
		{"drops literals shorter than minLength",
			"const a = 'ok'; const b = 'yes';", []string{"yes"}},
		{"drops idiomatic values (encodings, stdio modes, typeof results)",
			"readFileSync(p, 'utf8');\n" +
				"spawnSync(cmd, { stdio: 'inherit' });\n" +
				"if (typeof x === 'object') noop();\n" +
				"const keep = 'active';",
			[]string{"active"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := values(tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("values = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScanStringLiteralsLines(t *testing.T) {
	got := scanStringLiterals("const a = 'pending';\nconst b = \"active\";")
	want := []literal{{line: 1, value: "pending"}, {line: 2, value: "active"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("literals = %v, want %v", got, want)
	}
}

func TestFindDuplicates(t *testing.T) {
	t.Run("flags a value at or above the occurrence threshold and lists locations", func(t *testing.T) {
		errs := findDuplicates([]scannedFile{
			{file: "a.ts", literals: []literal{{line: 1, value: "active"}, {line: 9, value: "active"}}},
			{file: "b.ts", literals: []literal{{line: 2, value: "active"}}},
		}, nil)
		want := []string{`"active" appears 3 times (a.ts:1, a.ts:9, b.ts:2) — extract a shared constant`}
		if !reflect.DeepEqual(errs, want) {
			t.Fatalf("errors = %v, want %v", errs, want)
		}
	})

	t.Run("does not flag a value below the threshold", func(t *testing.T) {
		errs := findDuplicates([]scannedFile{
			{file: "a.ts", literals: []literal{{line: 1, value: "active"}, {line: 2, value: "active"}}},
		}, nil)
		if len(errs) != 0 {
			t.Fatalf("errors = %v, want none", errs)
		}
	})

	t.Run("caps listed locations and appends a +N more suffix", func(t *testing.T) {
		lits := make([]literal, 7)
		for i := range lits {
			lits[i] = literal{line: i + 1, value: "active"}
		}
		errs := findDuplicates([]scannedFile{{file: "a.ts", literals: lits}}, nil)
		want := `"active" appears 7 times (a.ts:1, a.ts:2, a.ts:3, a.ts:4, a.ts:5, +2 more) — extract a shared constant`
		if len(errs) != 1 || errs[0] != want {
			t.Fatalf("errors = %v, want [%s]", errs, want)
		}
	})

	t.Run("orders most-repeated first", func(t *testing.T) {
		errs := findDuplicates([]scannedFile{{file: "a.ts", literals: []literal{
			{line: 1, value: "twice"}, {line: 2, value: "twice"}, {line: 3, value: "twice"},
			{line: 4, value: "thrice"}, {line: 5, value: "thrice"},
			{line: 6, value: "thrice"}, {line: 7, value: "thrice"},
		}}}, nil)
		if len(errs) != 2 {
			t.Fatalf("errors = %v, want 2", errs)
		}
		if !strings.HasPrefix(errs[0], `"thrice" appears 4 times`) || !strings.HasPrefix(errs[1], `"twice" appears 3 times`) {
			t.Fatalf("errors = %q, want thrice (4) before twice (3)", errs)
		}
	})

	t.Run("never flags allowed values", func(t *testing.T) {
		errs := findDuplicates([]scannedFile{{file: "a.ts", literals: []literal{
			{line: 1, value: "active"}, {line: 2, value: "active"}, {line: 3, value: "active"},
		}}}, map[string]bool{"active": true})
		if len(errs) != 0 {
			t.Fatalf("errors = %v, want none", errs)
		}
	})
}

func TestRun(t *testing.T) {
	thrice := "const a = 'active';\nconst b = 'active';\nconst c = 'active';\n"
	cases := []struct {
		name   string
		files  map[string]string
		errors []string
		count  int
	}{
		{"passes when no literal repeats past the threshold",
			map[string]string{"a.ts": "const a = 'alpha';\nconst b = 'beta';\n"}, nil, 1},
		{"fails on a literal repeated across files",
			map[string]string{"a.ts": "const a = 'active';\nconst b = 'active';\n", "b.ts": "const c = 'active';\n"},
			[]string{`"active" appears 3 times (a.ts:1, a.ts:2, b.ts:1) — extract a shared constant`}, 2},
		{"scans test and bench files like any source",
			map[string]string{"a.test.ts": thrice, "b.bench.ts": thrice},
			[]string{`"active" appears 6 times (a.test.ts:1, a.test.ts:2, a.test.ts:3, b.bench.ts:1, b.bench.ts:2, +1 more) — extract a shared constant`}, 2},
		{"never flags values allowed via .fitnessrc.json repeatedStringLiterals.allow",
			map[string]string{".fitnessrc.json": `{"repeatedStringLiterals": {"allow": ["active"]}}`, "a.ts": thrice}, nil, 1},
		// Deviation from TS (which evaluates .fitnessrc.js): the Go check
		// cannot run JS, so the allow list is empty and the value is flagged;
		// the config file is itself scanned as source.
		{"a legacy .fitnessrc.js degrades to no allow list",
			map[string]string{".fitnessrc.js": "module.exports = { repeatedStringLiterals: { allow: ['active'] } };\n", "a.ts": thrice},
			[]string{`"active" appears 4 times (.fitnessrc.js:1, a.ts:1, a.ts:2, a.ts:3) — extract a shared constant`}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := run(tempRepo(t, tc.files), nil)
			if err != nil || res.Ok != (len(tc.errors) == 0) || res.FilesChecked != tc.count {
				t.Fatalf("unexpected result: %+v (err %v)", res, err)
			}
			if !slices.Equal(res.Errors, tc.errors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.errors)
			}
		})
	}
}
