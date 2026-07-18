package vitestconf

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCoverageExclude(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		want  []string
		found bool
	}{
		{"double and single quotes",
			`module.exports = { test: { coverage: { exclude: ["a", 'b'] } } };`,
			[]string{"a", "b"}, true},
		{"defineConfig wrapper with identifiers mixed in",
			`import { defineConfig } from 'vitest/config';
import { DTS_GLOB, TEST_GLOB } from './constants.cjs';
export default defineConfig({
  test: {
    coverage: {
      exclude: [
        '**/*.bench.ts',
        DTS_GLOB,
        TEST_GLOB,
        'packages/runner/src/index.ts',
      ],
      include: ['packages/runner/src/**/*.ts'],
      provider: 'v8',
    },
  },
});`,
			[]string{"**/*.bench.ts", "packages/runner/src/index.ts"}, true},
		{"plain template literal collected",
			"module.exports = { coverage: { exclude: [`**/*.d.ts`] } };",
			[]string{"**/*.d.ts"}, true},
		{"template with expression skipped",
			"const d = 'd'; module.exports = { coverage: { exclude: [`**/*.${d}.ts`, \"x\"] } };",
			[]string{"x"}, true},
		{"comments stripped string-aware",
			`module.exports = { coverage: { exclude: [ // line comment "not a string"
        "src//x.d.ts", /* block "comment" */ "z" ] } };`,
			[]string{"src//x.d.ts", "z"}, true},
		{"escaped quotes decoded",
			`module.exports = { coverage: { exclude: ["sr\"c.ts", 'a\'b'] } };`,
			[]string{`sr"c.ts`, "a'b"}, true},
		{"unicode escapes decoded",
			`module.exports = { coverage: { exclude: ["**/*.d.ts", "\x2a.ts", "\u{2a}"] } };`,
			[]string{"**/*.d.ts", "*.ts", "*"}, true},
		{"regex literal with quote does not derail lexing",
			`const re = /["']+/g; module.exports = { coverage: { exclude: ["x"] } };`,
			[]string{"x"}, true},
		{"division is not a regex",
			`const half = total / 2; module.exports = { coverage: { exclude: ["x"] } };`,
			[]string{"x"}, true},
		{"spread entries skipped",
			`module.exports = { coverage: { exclude: [...base, "x"] } };`,
			[]string{"x"}, true},
		{"call-wrapped strings skipped",
			`module.exports = { coverage: { exclude: [glob("y"), "x"] } };`,
			[]string{"x"}, true},
		{"concatenation halves skipped",
			`module.exports = { coverage: { exclude: ["a" + "b", "x"] } };`,
			[]string{"x"}, true},
		{"quoted keys accepted",
			`module.exports = { "coverage": { "exclude": ["x"] } };`,
			[]string{"x"}, true},
		{"nested exclude is not coverage's",
			`module.exports = { coverage: { thresholds: { exclude: ["x"] } } };`,
			nil, false},
		{"test.exclude without coverage not matched",
			`module.exports = { test: { exclude: ["x"] } };`,
			nil, false},
		{"shorthand coverage not matched",
			`const coverage = { exclude: ["x"] }; module.exports = { test: { coverage } };`,
			nil, false},
		{"coverage boolean skipped, later block found",
			`module.exports = { a: { coverage: true }, b: { coverage: { exclude: ["x"] } } };`,
			[]string{"x"}, true},
		{"coverage without exclude then coverage with one",
			`module.exports = { a: { coverage: { provider: "v8" } }, b: { coverage: { exclude: ["x"] } } };`,
			[]string{"x"}, true},
		{"empty array found",
			`module.exports = { test: { coverage: { exclude: [] } } };`,
			nil, true},
		{"no coverage at all",
			`export default { test: {} };`,
			nil, false},
		{"throwing config has no excludes",
			`throw new Error("bad");`,
			nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := CoverageExclude(tc.src)
			if found != tc.found {
				t.Fatalf("found = %v, want %v (got %v)", found, tc.found, got)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("exclude = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestLoadExcludeFromRoot(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []string
		found bool
	}{
		{"no config source", nil, nil, false},
		{"cjs probed before js",
			map[string]string{
				"vitest.config.cjs": `module.exports = { coverage: { exclude: ["from-cjs.d.ts"] } };`,
				"vitest.config.js":  `module.exports = { coverage: { exclude: ["from-js.d.ts"] } };`,
			},
			[]string{"from-cjs.d.ts"}, true},
		{"config file beats package.json",
			map[string]string{
				"vitest.config.ts": `export default { coverage: { exclude: ["from-ts.d.ts"] } };`,
				"package.json":     `{"vitest":{"coverage":{"exclude":["from-pkg.d.ts"]}}}`,
			},
			[]string{"from-ts.d.ts"}, true},
		{"package.json test.coverage wins over coverage",
			map[string]string{
				"package.json": `{"vitest":{"test":{"coverage":{"exclude":["t.ts"]}},"coverage":{"exclude":["c.ts"]}}}`,
			},
			[]string{"t.ts"}, true},
		{"package.json null test.coverage falls back to coverage",
			map[string]string{
				"package.json": `{"vitest":{"test":{"coverage":null},"coverage":{"exclude":["c.ts"]}}}`,
			},
			[]string{"c.ts"}, true},
		{"package.json non-object test.coverage does not fall back",
			map[string]string{
				"package.json": `{"vitest":{"test":{"coverage":5},"coverage":{"exclude":["c.ts"]}}}`,
			},
			nil, true},
		{"package.json non-string entries filtered",
			map[string]string{
				"package.json": `{"vitest":{"coverage":{"exclude":["x", 1, null]}}}`,
			},
			[]string{"x"}, true},
		{"package.json non-array exclude", map[string]string{
			"package.json": `{"vitest":{"coverage":{"exclude":"not-array"}}}`,
		}, nil, true},
		{"package.json vitest non-object", map[string]string{
			"package.json": `{"vitest":"invalid"}`,
		}, nil, false},
		{"package.json invalid JSON", map[string]string{
			"package.json": `not json`,
		}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, found := LoadExcludeFromRoot(dir)
			if found != tc.found {
				t.Fatalf("found = %v, want %v (got %v)", found, tc.found, got)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("exclude = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestFitnessRunnerRoot(t *testing.T) {
	t.Run("no shared install falls back to root's cspell ancestor", func(t *testing.T) {
		tmp := t.TempDir()
		root := filepath.Join(tmp, "a", "b")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "a", "cspell.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := FitnessRunnerRoot(root); got != filepath.Join(tmp, "a") {
			t.Fatalf("FitnessRunnerRoot = %q, want %q", got, filepath.Join(tmp, "a"))
		}
	})

	t.Run("no cspell anywhere returns root", func(t *testing.T) {
		root := t.TempDir()
		if got := FitnessRunnerRoot(root); got != root {
			t.Fatalf("FitnessRunnerRoot = %q, want %q", got, root)
		}
	})

	t.Run("shared install resolves outermost cspell ancestor of its config dir", func(t *testing.T) {
		tmp := t.TempDir()
		outer := filepath.Join(tmp, "outer")
		root := filepath.Join(outer, "proj")
		configDir := filepath.Join(root, "node_modules", "@mayjournal", "fitness-shared", "config")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{
			filepath.Join(configDir, "..", "package.json"): `{"exports":{"./cspell":"./config/cspell.json"}}`,
			filepath.Join(configDir, "cspell.json"):        "{}",
			filepath.Join(outer, "cspell.json"):            "{}",
		}
		for path, content := range files {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if got := FitnessRunnerRoot(root); got != outer {
			t.Fatalf("FitnessRunnerRoot = %q, want %q", got, outer)
		}
	})

	t.Run("conditional export object resolves via default", func(t *testing.T) {
		root := t.TempDir()
		configDir := filepath.Join(root, "node_modules", "@mayjournal", "fitness-shared", "config")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		pkg := `{"exports":{"./cspell":{"default":"./config/cspell.json"}}}`
		if err := os.WriteFile(filepath.Join(configDir, "..", "package.json"), []byte(pkg), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "cspell.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := FitnessRunnerRoot(root); got != configDir {
			t.Fatalf("FitnessRunnerRoot = %q, want %q", got, configDir)
		}
	})

	t.Run("install without cspell export falls back to root walk", func(t *testing.T) {
		root := t.TempDir()
		pkgDir := filepath.Join(root, "node_modules", "@mayjournal", "fitness-shared")
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"exports":{}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "cspell.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := FitnessRunnerRoot(root); got != root {
			t.Fatalf("FitnessRunnerRoot = %q, want %q", got, root)
		}
	})
}

func TestHasFullThresholds(t *testing.T) {
	const fullJS = "module.exports = { test: { coverage: { thresholds: " +
		"{ branches: 100, functions: 100, lines: 100, statements: 100 } } } };"
	const partialJS = "module.exports = { test: { coverage: { thresholds: " +
		"{ branches: 90, functions: 100, lines: 100, statements: 100 } } } };"
	cases := []struct {
		name          string
		rootFiles     map[string]string
		rootDirs      []string
		fallbackFiles map[string]string
		noFallback    bool
		want          bool
	}{
		{name: "all literal 100 pass", want: true,
			rootFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "one threshold 90 fails", want: false,
			rootFiles: map[string]string{"vitest.config.js": partialJS}},
		{name: "no space after colon still matches", want: true,
			rootFiles: map[string]string{"vitest.config.js": "branches:100, functions:100, lines:100, statements:100"}},
		{name: "1000 is not 100", want: false,
			rootFiles: map[string]string{"vitest.config.js": "branches: 1000, functions: 100, lines: 100, statements: 100"}},
		{name: "raw fast path reads comments like the TS did", want: true,
			rootFiles: map[string]string{"vitest.config.js": "module.exports = null; // branches: 100, functions: 100, lines: 100, statements: 100"}},
		{name: "variable-built thresholds are unresolvable", want: false,
			rootFiles: map[string]string{"vitest.config.mjs": "const t = 100; export default { test: { coverage: { thresholds: { branches: t, functions: t, lines: t, statements: t } } } };"}},
		{name: "empty coverage block fails", want: false,
			rootFiles: map[string]string{"vitest.config.js": "module.exports = { test: { coverage: {} } };"}},
		{name: "fallback used when root has no source", want: true,
			fallbackFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "root config decides over a full fallback", want: false,
			rootFiles:     map[string]string{"vitest.config.js": partialJS},
			fallbackFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "any raw-matching config file wins, not just the first", want: true,
			rootFiles: map[string]string{
				"vitest.config.cjs": partialJS,
				"vitest.config.js":  fullJS,
			}},
		{name: "unreadable config candidate skipped", want: true,
			rootDirs:  []string{"vitest.config.cjs"},
			rootFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "config file beats full package.json vitest", want: false,
			rootFiles: map[string]string{
				"vitest.config.js": partialJS,
				"package.json":     `{"vitest":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":100}}}}`,
			}},
		{name: "package.json top-level coverage full", want: true,
			rootFiles: map[string]string{"package.json": `{"vitest":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":100}}}}`}},
		{name: "package.json test.coverage full", want: true,
			rootFiles: map[string]string{"package.json": `{"vitest":{"test":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":100}}}}}`}},
		{name: "package.json thresholds 99 decide false over full fallback", want: false,
			rootFiles:     map[string]string{"package.json": `{"vitest":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":99}}}}`},
			fallbackFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "package.json vitest null falls through to fallback", want: true,
			rootFiles:     map[string]string{"package.json": `{"vitest":null}`},
			fallbackFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "invalid package.json falls through to fallback", want: true,
			rootFiles:     map[string]string{"package.json": `{ invalid }`},
			fallbackFiles: map[string]string{"vitest.config.js": fullJS}},
		{name: "missing statements key fails", want: false,
			rootFiles: map[string]string{"package.json": `{"vitest":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100}}}}`}},
		{name: "string threshold values fail", want: false,
			rootFiles: map[string]string{"package.json": `{"vitest":{"coverage":{"thresholds":{"branches":"100","functions":"100","lines":"100","statements":"100"}}}}`}},
		{name: "non-object test.coverage does not fall back to coverage", want: false,
			rootFiles: map[string]string{"package.json": `{"vitest":{"test":{"coverage":5},"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":100}}}}`}},
		{name: "no source anywhere", want: false},
		{name: "no source and no fallback", noFallback: true, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range tc.rootDirs {
				if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for name, content := range tc.rootFiles {
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			fallback := ""
			if !tc.noFallback {
				fallback = t.TempDir()
				for name, content := range tc.fallbackFiles {
					if err := os.WriteFile(filepath.Join(fallback, name), []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			if got := HasFullThresholds(root, fallback); got != tc.want {
				t.Fatalf("HasFullThresholds = %v, want %v", got, tc.want)
			}
		})
	}
}
