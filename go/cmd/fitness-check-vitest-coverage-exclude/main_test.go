package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// removeMsg is the failure line for one exclude entry, quoted as reported.
func removeMsg(quoted string) string {
	return "Vitest coverage exclude must be empty, so coverage judges every file; remove: " + quoted
}

func TestJudge(t *testing.T) {
	cases := []struct {
		name    string
		exclude []string
		errors  []string
	}{
		{"empty exclude passes", nil, nil},
		{"non-.ts patterns fail", []string{"node_modules", "dist"},
			[]string{removeMsg(`"node_modules"`), removeMsg(`"dist"`)}},
		{"test and type patterns fail", []string{"**/*.test.ts", "**/*.d.ts"},
			[]string{removeMsg(`"**/*.test.ts"`), removeMsg(`"**/*.d.ts"`)}},
		{"directory glob fails", []string{"src/types/**"}, []string{removeMsg(`"src/types/**"`)}},
		{"padded pattern reports verbatim", []string{"  src/foo.ts  "}, []string{removeMsg(`"  src/foo.ts  "`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertResult(t, judge(tc.exclude), nil, 1, tc.errors)
		})
	}
}

// writeFiles writes each name:content pair into dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRun(t *testing.T) {
	foo := []string{removeMsg(`"src/foo.ts"`)}
	cases := []struct {
		name   string
		files  map[string]string
		errors []string
	}{
		{"no config passes", nil, nil},
		{"empty exclude passes", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: [] } } };`,
		}, nil},
		{"conventional excludes fail too", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["**/*.test.ts", "dist"] } } };`,
		}, []string{removeMsg(`"**/*.test.ts"`), removeMsg(`"dist"`)}},
		{"unresolvable identifiers skipped, literal judged", map[string]string{
			"vitest.config.mjs": `import { DTS_GLOB } from "./globs.js";
export default { test: { coverage: { exclude: [DTS_GLOB, "src/foo.ts"] } } };`,
		}, foo},
		{"package.json vitest exclude fails", map[string]string{
			"package.json": `{"vitest":{"test":{"coverage":{"exclude":["src/foo.ts"]}}}}`,
		}, foo},
		{"package.json top-level coverage fails", map[string]string{
			"package.json": `{"vitest":{"coverage":{"exclude":["src/foo.ts"]}}}`,
		}, foo},
		{"package.json vitest non-object ignored", map[string]string{"package.json": `{"vitest":"invalid"}`}, nil},
		{"package.json invalid JSON ignored", map[string]string{"package.json": `not json`}, nil},
		{"package.json non-array exclude passes", map[string]string{
			"package.json": `{"vitest":{"coverage":{"exclude":"not-array"}}}`,
		}, nil},
		{"throwing config yields no excludes", map[string]string{"vitest.config.js": `throw new Error("bad");`}, nil},
		{"cjs wins over ts", map[string]string{
			"vitest.config.cjs": `module.exports = { test: { coverage: { exclude: [] } } };`,
			"vitest.config.ts":  `export default { test: { coverage: { exclude: ["src/foo.ts"] } } };`,
		}, nil},
		{"cjs wins over js", map[string]string{
			"vitest.config.cjs": `module.exports = { test: { coverage: { exclude: ["src/foo.ts"] } } };`,
			"vitest.config.js":  `module.exports = { test: { coverage: { exclude: [] } } };`,
		}, foo},
		{"config file wins over package.json", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: [] } } };`,
			"package.json":     `{"vitest":{"coverage":{"exclude":["src/foo.ts"]}}}`,
		}, nil},
		{"config without coverage exclude passes", map[string]string{
			"vitest.config.mjs": `export default { test: {} };`,
		}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			// Mark the root as a JS project so the package.json self-gate
			// lets the check run; a real package.json in tc.files overwrites
			// this bare one.
			writeFiles(t, dir, map[string]string{"package.json": "{}"})
			writeFiles(t, dir, tc.files)
			res, err := run(dir, nil)
			assertResult(t, res, err, 1, tc.errors)
		})
	}
}

// TestRunFallbackRoot pins the fallback chain: a root without any config
// source is judged against the installed @mayjournal/fitness-shared config
// directory (walking up from root), and a local config source shadows that
// fallback entirely.
func TestRunFallbackRoot(t *testing.T) {
	tmp := t.TempDir()
	writeFiles(t, tmp, map[string]string{
		"outer/proj/package.json": "{}",
		"outer/node_modules/@mayjournal/fitness-shared/config/vitest.config.js": `module.exports = { test: { coverage: { exclude: ["src/foo.ts"] } } };`,
	})
	proj := filepath.Join(tmp, "outer", "proj")
	res, err := run(proj, nil)
	assertResult(t, res, err, 1, []string{removeMsg(`"src/foo.ts"`)})

	writeFiles(t, proj, map[string]string{
		"vitest.config.js": `module.exports = { test: { coverage: { exclude: [] } } };`,
	})
	res, err = run(proj, nil)
	assertResult(t, res, err, 1, nil)
}

// TestLoadExcludeEmbeddedFallback pins the npm-free path: with no local
// config and no node_modules anywhere up the tree, the exclude list comes
// from the embedded vitest.config.mjs, which excludes nothing.
func TestLoadExcludeEmbeddedFallback(t *testing.T) {
	if got := loadExclude(t.TempDir()); len(got) != 0 {
		t.Fatalf("loadExclude = %#v, want the embedded config's empty list", got)
	}
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"package.json": "{}"})
	res, err := run(dir, nil)
	assertResult(t, res, err, 1, nil)
}

// TestNonJSProjectSkips pins the self-gate: a root without any package.json
// is not a JS project, so the check skips clean instead of judging a
// fallback config it never opted into.
func TestNonJSProjectSkips(t *testing.T) {
	res, err := run(t.TempDir(), nil)
	assertResult(t, res, err, 0, nil)
}

// assertResult checks a verdict: it passes exactly when errors is empty.
func assertResult(t *testing.T, res checkkit.Result, err error, files int, errors []string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok != (len(errors) == 0) || res.FilesChecked != files {
		t.Fatalf("ok = %v, files = %d, want files %d (errors: %v)", res.Ok, res.FilesChecked, files, res.Errors)
	}
	if !slices.Equal(res.Errors, errors) {
		t.Fatalf("errors = %v, want %v", res.Errors, errors)
	}
}
