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

func vcfRun(t *testing.T, repo string, env []string) (string, int) {
	t.Helper()
	return fitness(t, repo, env, "vitest-coverage-full")
}

// vcfTable is the coverage table Vitest's v8 provider prints after a run.
func vcfTable(rows string) string {
	rule := "----------|---------|----------|---------|---------|-------------------\n"
	return " % Coverage report from v8\n" + rule +
		"File      | % Stmts | % Branch | % Funcs | % Lines | Uncovered Line #s \n" + rule + rows + rule
}

// vcfFullRun is what `vitest run --coverage` prints when every line ran.
var vcfFullRun = response{
	Match: []string{"run --coverage"},
	Stdout: "\n RUN  v3.2.4 /repo\n\n \u2713 src/add.test.ts (1 test) 2ms\n\n" +
		" Test Files  1 passed (1)\n      Tests  1 passed (1)\n   Start at  10:15:02\n   Duration  412ms\n\n" +
		vcfTable("All files |     100 |      100 |     100 |     100 |                   \n"+
			" add.ts   |     100 |      100 |     100 |     100 |                   \n"),
}

// vcfShortRun is what `vitest run --coverage` prints when sub.ts never ran:
// the table on stdout, then one threshold error per metric on stderr.
var vcfShortRun = response{
	Match: []string{"run --coverage"},
	Stdout: "\n RUN  v3.2.4 /repo\n\n \u2713 src/add.test.ts (1 test) 2ms\n\n" +
		" Test Files  1 passed (1)\n      Tests  1 passed (1)\n   Start at  10:15:02\n   Duration  418ms\n\n" +
		vcfTable("All files |      50 |      100 |      50 |      50 |                   \n"+
			" add.ts   |     100 |      100 |     100 |     100 |                   \n"+
			" sub.ts   |       0 |      100 |       0 |       0 | 1                 \n"),
	Stderr: "ERROR: Coverage for lines (50%) does not meet global threshold (100%)\n" +
		"ERROR: Coverage for functions (50%) does not meet global threshold (100%)\n" +
		"ERROR: Coverage for statements (50%) does not meet global threshold (100%)\n",
	Exit: 1,
}

// vcfRanWithLocalConfig fails t unless fitness ran vitest once with
// `run --coverage` alone, leaving Vitest to load the repo's own config.
func vcfRanWithLocalConfig(t *testing.T, rp replays) {
	t.Helper()
	if calls := rp.calls("vitest"); len(calls) != 1 || calls[0] != "run --coverage" {
		t.Errorf("vitest calls = %q, want one \"run --coverage\"", calls)
	}
}

func Test0060_1(t *testing.T) {
	t.Parallel()
	out, code := vcfRun(t, example(t, "happyRepo", nil), nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "vitest-coverage-full")
}

func Test0060_2(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"vitest": {vcfFullRun}})
	out, code := vcfRun(t, vcfRepo(t, map[string]string{"vitest.config.mjs": vcfConfig("90")}), rp.env)
	sees(t, out, code, 1, "Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.")
}

func Test0060_3(t *testing.T) {
	t.Parallel()
	shared := "node_modules/@mayjournal/fitness-shared/config/vitest.config.mjs"
	out, code := vcfRun(t, vcfRepo(t, map[string]string{shared: vcfConfig("90")}), nil)
	sees(t, out, code, 1, "Fitness-runner package must have Vitest coverage thresholds set to 100 for branches, functions, lines, and statements.")
}

func Test0060_4(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"vitest": {vcfFullRun}})
	out, code := vcfRun(t, vcfRepo(t, nil), rp.env)
	sees(t, out, code, 0, "All 1 checks passed")
	vcfRanWithLocalConfig(t, rp)
}

func Test0060_5(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"vitest": {vcfShortRun}})
	repo := vcfRepo(t, map[string]string{"src/sub.ts": "export const sub = (a: number, b: number): number => a - b;\n"})
	out, code := vcfRun(t, repo, rp.env)
	sees(t, out, code, 1, "Coverage for statements (50%) does not meet global threshold (100%)")
	vcfRanWithLocalConfig(t, rp)
}

func Test0060_6(t *testing.T) {
	t.Parallel()
	out, code := vcfRun(t, vcfRepo(t, nil), nil)
	sees(t, out, code, 1, "vitest not found in node_modules/.bin (walking up from root) or on PATH. Install it (npm install -D vitest)")
}
