package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const allowedList = "**/*.bench.ts, **/*.d.ts, **/*.types.ts, **/*.test.ts, " +
	"**/*.spec.ts, **/index.ts, **/run-one-check-worker.ts"

func TestJudge(t *testing.T) {
	cases := []struct {
		name    string
		exclude []string
		ok      bool
		errors  []string
	}{
		{"empty exclude passes", nil, true, nil},
		{"non-.ts patterns pass", []string{"node_modules", "dist"}, true, nil},
		{"all allowed patterns pass", allowedCoverageExcludePatterns, true, nil},
		{"src-prefixed conventional passes", []string{"src/**/*.types.ts"}, true, nil},
		{"test glob passes", []string{"src/**/*.test.ts"}, true, nil},
		{"spec glob passes", []string{"**/*.spec.ts"}, true, nil},
		{"plain .ts path fails", []string{"src/config/load.ts"}, false, []string{
			"Vitest coverage exclude only allows " + allowedList +
				`; disallowed: "src/config/load.ts"`,
		}},
		{"directory glob fails", []string{"src/types/**"}, false, []string{
			"Vitest coverage exclude only allows " + allowedList +
				`; disallowed: "src/types/**"`,
		}},
		{"padded pattern trims for judgment but reports verbatim",
			[]string{"  src/foo.ts  "}, false, []string{
				"Vitest coverage exclude only allows " + allowedList +
					`; disallowed: "  src/foo.ts  "`,
			}},
		{"padded allowed pattern passes", []string{"  **/*.d.ts  "}, true, nil},
		{"all disallowed patterns reported", []string{"src/foo.ts", "src/types/**"}, false,
			[]string{
				"Vitest coverage exclude only allows " + allowedList +
					`; disallowed: "src/foo.ts"`,
				"Vitest coverage exclude only allows " + allowedList +
					`; disallowed: "src/types/**"`,
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.exclude)
			assertResult(t, res, tc.ok, tc.errors)
		})
	}
}

func TestAllowedSuffixes(t *testing.T) {
	cases := []struct {
		pattern string
		want    bool
	}{
		{"foo.d.ts", true},
		{"bar.types.ts", true},
		{"baz.test.ts", true},
		{"baz.spec.ts", true},
		{"qux.bench.ts", true},
		{"index.ts", true},
		{"run-one-check-worker.ts", true},
		{"baz.ts", false},
	}
	for _, tc := range cases {
		if got := allowedSuffixes.MatchString(tc.pattern); got != tc.want {
			t.Errorf("allowedSuffixes.MatchString(%q) = %v, want %v", tc.pattern, got, tc.want)
		}
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		files      map[string]string
		ok         bool
		wantErrors int
		wantInErr  string
	}{
		{"no config passes", nil, true, 0, ""},
		{"empty exclude passes", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: [] } } };`,
		}, true, 0, ""},
		{"non-.ts excludes pass", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["node_modules", "dist"] } } };`,
		}, true, 0, ""},
		{"conventional excludes pass", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["**/*.bench.ts", "**/*.d.ts", "**/*.types.ts", "**/*.test.ts", "**/*.spec.ts", "**/index.ts", "**/run-one-check-worker.ts"] } } };`,
		}, true, 0, ""},
		{"plain .ts path fails", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["src/config/load.ts"] } } };`,
		}, false, 1, "load.ts"},
		{"directory glob fails", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["src/types/**"] } } };`,
		}, false, 1, "src/types/**"},
		{"both disallowed reported", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["src/foo.ts", "src/types/**"] } } };`,
		}, false, 2, ""},
		{"unresolvable identifiers skipped, literal judged", map[string]string{
			"vitest.config.mjs": `import { DTS_GLOB } from "./globs.js";
export default { test: { coverage: { exclude: [DTS_GLOB, "src/foo.ts"] } } };`,
		}, false, 1, `"src/foo.ts"`},
		{"package.json vitest allowed passes", map[string]string{
			"package.json": `{"vitest":{"test":{"coverage":{"exclude":["**/*.types.ts"]}}}}`,
		}, true, 0, ""},
		{"package.json top-level coverage disallowed fails", map[string]string{
			"package.json": `{"vitest":{"coverage":{"exclude":["src/foo.ts"]}}}`,
		}, false, 1, `"src/foo.ts"`},
		{"package.json vitest non-object ignored", map[string]string{
			"package.json": `{"vitest":"invalid"}`,
		}, true, 0, ""},
		{"package.json invalid JSON ignored", map[string]string{
			"package.json": `not json`,
		}, true, 0, ""},
		{"package.json non-array exclude passes", map[string]string{
			"package.json": `{"vitest":{"coverage":{"exclude":"not-array"}}}`,
		}, true, 0, ""},
		{"throwing config yields no excludes", map[string]string{
			"vitest.config.js": `throw new Error("bad");`,
		}, true, 0, ""},
		{"cjs wins over ts (allowed shadows disallowed)", map[string]string{
			"vitest.config.cjs": `module.exports = { test: { coverage: { exclude: ["**/*.d.ts"] } } };`,
			"vitest.config.ts":  `export default { test: { coverage: { exclude: ["src/foo.ts"] } } };`,
		}, true, 0, ""},
		{"cjs wins over js (disallowed shadows allowed)", map[string]string{
			"vitest.config.cjs": `module.exports = { test: { coverage: { exclude: ["src/foo.ts"] } } };`,
			"vitest.config.js":  `module.exports = { test: { coverage: { exclude: ["**/*.d.ts"] } } };`,
		}, false, 1, `"src/foo.ts"`},
		{"config file wins over package.json", map[string]string{
			"vitest.config.js": `module.exports = { test: { coverage: { exclude: ["**/*.d.ts"] } } };`,
			"package.json":     `{"vitest":{"coverage":{"exclude":["src/foo.ts"]}}}`,
		}, true, 0, ""},
		{"config without coverage exclude passes", map[string]string{
			"vitest.config.mjs": `export default { test: {} };`,
		}, true, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			res, err := run(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if len(res.Errors) != tc.wantErrors {
				t.Fatalf("errors = %v, want %d of them", res.Errors, tc.wantErrors)
			}
			if tc.wantInErr != "" && !strings.Contains(res.Errors[0], tc.wantInErr) {
				t.Fatalf("error %q does not contain %q", res.Errors[0], tc.wantInErr)
			}
		})
	}
}

// TestRunFallbackRoot pins the fallback chain: a root without any config
// source is judged against the installed @mayjournal/fitness-shared config
// directory (walking up from root), and a local config source shadows that
// fallback entirely.
func TestRunFallbackRoot(t *testing.T) {
	tmp := t.TempDir()
	proj := filepath.Join(tmp, "outer", "proj")
	configDir := filepath.Join(tmp, "outer", "node_modules", "@mayjournal", "fitness-shared", "config")
	for _, dir := range []string{proj, configDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(configDir, "vitest.config.js"),
		[]byte(`module.exports = { test: { coverage: { exclude: ["src/foo.ts"] } } };`), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := run(proj, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || len(res.Errors) != 1 || !strings.Contains(res.Errors[0], `"src/foo.ts"`) {
		t.Fatalf("expected installed-config fallback failure, got %+v", res)
	}

	// A local config source shadows the fallback entirely.
	if err := os.WriteFile(filepath.Join(proj, "vitest.config.js"),
		[]byte(`module.exports = { test: { coverage: { exclude: ["**/*.d.ts"] } } };`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = run(proj, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected local config to shadow fallback, got %+v", res)
	}
}

// TestLoadExcludeEmbeddedFallback pins the npm-free path: with no local
// config and no node_modules anywhere up the tree, the exclude list comes
// from the embedded vitest.config.mjs materialized out of the binary (its
// two clean string literals; identifier entries are unresolvable).
func TestLoadExcludeEmbeddedFallback(t *testing.T) {
	got := loadExclude(t.TempDir())
	want := []string{
		"**/*.bench.ts",
		"**/index.ts",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadExclude = %#v, want the embedded config's literals %#v", got, want)
	}
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("embedded exclude list should pass the judgment: %+v", res)
	}
}

func assertResult(t *testing.T, res checkkit.Result, ok bool, errors []string) {
	t.Helper()
	if res.Ok != ok {
		t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, ok, res.Errors)
	}
	if res.FilesChecked != 1 {
		t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
	}
	if len(res.Errors) != len(errors) {
		t.Fatalf("errors = %v, want %v", res.Errors, errors)
	}
	for i, want := range errors {
		if res.Errors[i] != want {
			t.Fatalf("error[%d] = %q, want %q", i, res.Errors[i], want)
		}
	}
}
