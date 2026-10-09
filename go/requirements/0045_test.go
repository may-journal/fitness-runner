package requirements

import "testing"

// noEslintDisableRun runs `fitness-install -- no-eslint-disable <args>` in repo.
func noEslintDisableRun(t *testing.T, repo string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, append([]string{"no-eslint-disable"}, args...)...)
}

// noEslintDisableCheck runs the check on happyRepo with files written over it.
func noEslintDisableCheck(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return noEslintDisableRun(t, example(t, "happyRepo", files))
}

// noEslintDisableNext is a next-line directive comment.
const noEslintDisableNext = "// eslint-disable-next-line no-console\n"

// noEslintDisableExts are the extensions the check scans.
var noEslintDisableExts = []string{".cts", ".mts", ".ts", ".cjs", ".js", ".mjs", ".tsx"}

func Test0045_1(t *testing.T) {
	legacy := "export const x = 1;\n" + noEslintDisableNext + "console.log(x);\n" +
		"const y = 2; // eslint-disable-line\n/* eslint-disable */\n"
	out, code := noEslintDisableCheck(t, map[string]string{"src/legacy.ts": legacy})
	sees(t, out, code, 1,
		"src/legacy.ts:2: eslint-disable-next-line",
		"src/legacy.ts:4: eslint-disable-line",
		"src/legacy.ts:5: eslint-disable")
}

func Test0045_2(t *testing.T) {
	out, code := noEslintDisableCheck(t, map[string]string{
		"src/a.ts": "export const a = 1;\n",
		"src/b.js": "export const b = 2;\n",
	})
	sees(t, out, code, 0, "All 1 checks passed · 2 files scanned")
}

func Test0045_3(t *testing.T) {
	files := map[string]string{}
	var want []string
	for _, ext := range noEslintDisableExts {
		files["src/f"+ext] = "export const hint = 'add eslint-disable-next-line to skip';\n"
		want = append(want, "src/f"+ext+":1: eslint-disable-next-line")
	}
	out, code := noEslintDisableCheck(t, files)
	sees(t, out, code, 1, append(want, "7 files scanned")...)
}

func Test0045_4(t *testing.T) {
	out, code := noEslintDisableCheck(t, map[string]string{
		"docs/lint.md": "Never write " + noEslintDisableNext,
		"src/a.ts":     "export const a = 1;\n",
	})
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0045_5(t *testing.T) {
	out, code := noEslintDisableCheck(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "no-eslint-disable")
}

func Test0045_6(t *testing.T) {
	out, code := noEslintDisableCheck(t, map[string]string{
		"node_modules/pkg/index.js": "/* eslint-disable */\n",
		"dist/out.js":               "/* eslint-disable */\n",
		"src/clean.ts":              "export const a = 1;\n",
	})
	sees(t, out, code, 1,
		"node_modules/pkg/index.js:1: eslint-disable",
		"dist/out.js:1: eslint-disable",
		"3 files scanned")
}

func Test0045_7(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{"src/a.ts": "export const a = 1;\n"})
	write(t, repo, map[string]string{"src/scratch.ts": noEslintDisableNext})
	out, code := noEslintDisableRun(t, repo)
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0045_8(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{"src/old.ts": noEslintDisableNext})
	write(t, repo, map[string]string{"src/new.ts": "export const a = 1;\n"})
	git(t, repo, "add", "src/new.ts")
	out, code := noEslintDisableRun(t, repo)
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}
