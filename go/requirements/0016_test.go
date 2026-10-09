package requirements

import "testing"

// buildOutputCheck is the check these tests run.
const buildOutputCheck = "build-output-untracked"

// buildOutputIgnore is a .gitignore that ignores dist.
const buildOutputIgnore = "dist\n"

// buildOutputRun runs `fitness-install -- build-output-untracked` in repo.
func buildOutputRun(t *testing.T, repo string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, buildOutputCheck)
}

func Test0016_1(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{
		".gitignore": buildOutputIgnore,
		"src/app.test.ts": "import ok from './local.js';\n" +
			"import bad from '../dist/foo.js';\n" +
			"const z = require('pkg/dist/z.js');\n" +
			"const m = await import('../dist/mod.js');\n",
	})
	out, code := buildOutputRun(t, repo)
	sees(t, out, code, 1,
		`src/app.test.ts:2 imports build output: "../dist/foo.js"`,
		`src/app.test.ts:3 imports build output: "pkg/dist/z.js"`,
		`src/app.test.ts:4 imports build output: "../dist/mod.js"`)
}

func Test0016_2(t *testing.T) {
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"dist/app.js": "built\n"})
	out, code := buildOutputRun(t, repo)
	sees(t, out, code, 1, "Add dist/ to .gitignore (git check-ignore reports dist is not git-ignored)")
}

func Test0016_3(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{"sub/dist/b.js": "built\n", "dist/a.js": "built\n"})
	commit(t, repo, map[string]string{".gitignore": buildOutputIgnore}, "chore: ignore dist")
	out, code := buildOutputRun(t, repo)
	sees(t, out, code, 1, "Remove tracked files: dist/a.js, sub/dist/b.js (git rm --cached)")
}

func Test0016_4(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{
		".gitignore": buildOutputIgnore,
		"src/a.ts":   "import x from './b.js';\n",
		"src/b.mts":  "export const b = require('./c.cjs');\n",
	})
	out, code := buildOutputRun(t, repo)
	sees(t, out, code, 0, "All 1 checks passed · 3 files scanned")
}

func Test0016_5(t *testing.T) {
	out, code := buildOutputRun(t, example(t, "happyRepo", nil))
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, buildOutputCheck)
}

func Test0016_6(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{
		".gitignore": buildOutputIgnore,
		"src/a.ts":   "import x from './b.js';\n",
	})
	write(t, repo, map[string]string{"scratch.ts": "import x from '../dist/x.js';\n"})
	out, code := buildOutputRun(t, repo)
	sees(t, out, code, 0, "All 1 checks passed · 2 files scanned")
}
