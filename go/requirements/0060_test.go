package requirements

import "testing"

// vcfConfig is a local Vitest config holding the given coverage thresholds
// over src/**/*.ts, the way a consumer repo writes one.
func vcfConfig(branches string) string {
	return "export default {\n  test: {\n    coverage: {\n      include: ['src/**/*.ts'],\n      provider: 'v8',\n" +
		"      thresholds: { branches: " + branches + ", functions: 100, lines: 100, statements: 100 },\n" +
		"    },\n    include: ['src/**/*.test.ts'],\n  },\n};\n"
}

// vcfProject is a small JavaScript project whose one function is fully tested.
var vcfProject = map[string]string{
	"package.json":      `{"type": "module"}` + "\n",
	"vitest.config.mjs": vcfConfig("100"),
	"src/add.ts":        "export const add = (a: number, b: number): number => a + b;\n",
	"src/add.test.ts": "import { expect, test } from 'vitest';\nimport { add } from './add';\n\n" +
		"test('adds', () => {\n  expect(add(1, 2)).toBe(3);\n});\n",
}

// vcfRepo is happyRepo with vcfProject and then files written over it.
func vcfRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{}
	for _, set := range []map[string]string{vcfProject, files} {
		for name, body := range set {
			all[name] = body
		}
	}
	return example(t, "happyRepo", all)
}

func vcfRun(t *testing.T, repo string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, "vitest-coverage-full")
}

func Test0060_1(t *testing.T) {
	t.Parallel()
	out, code := vcfRun(t, example(t, "happyRepo", nil))
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "vitest-coverage-full")
}

func Test0060_2(t *testing.T) {
	t.Parallel()
	repo := vcfRepo(t, map[string]string{"vitest.config.mjs": vcfConfig("90")})
	withJSTools(t, repo)
	out, code := vcfRun(t, repo)
	sees(t, out, code, 1, "Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.")
}

func Test0060_3(t *testing.T) {
	t.Parallel()
	shared := "node_modules/@mayjournal/fitness-shared/config/vitest.config.mjs"
	out, code := vcfRun(t, vcfRepo(t, map[string]string{shared: vcfConfig("90")}))
	sees(t, out, code, 1, "Fitness-runner package must have Vitest coverage thresholds set to 100 for branches, functions, lines, and statements.")
}

func Test0060_4(t *testing.T) {
	t.Parallel()
	repo := vcfRepo(t, nil)
	withJSTools(t, repo)
	out, code := vcfRun(t, repo)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0060_5(t *testing.T) {
	t.Parallel()
	repo := vcfRepo(t, map[string]string{"src/sub.ts": "export const sub = (a: number, b: number): number => a - b;\n"})
	withJSTools(t, repo)
	out, code := vcfRun(t, repo)
	sees(t, out, code, 1, "Coverage for statements (50%) does not meet global threshold (100%)")
}

func Test0060_6(t *testing.T) {
	t.Parallel()
	out, code := vcfRun(t, vcfRepo(t, nil))
	sees(t, out, code, 1, "vitest not found in node_modules/.bin (walking up from root) or on PATH. Install it (npm install -D vitest)")
}
