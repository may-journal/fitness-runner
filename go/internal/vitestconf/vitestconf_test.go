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
