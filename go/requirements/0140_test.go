package requirements

import (
	"strings"
	"testing"
)

// viewsPartner is a file check that passes on happyRepo. A run of it and
// one more check reads both views the runner built for them.
const viewsPartner = "no-plans-dir"

// withPartner runs check and viewsPartner together on repo.
func withPartner(t *testing.T, repo, check string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, "--policy=external", "--checks="+check+","+viewsPartner)
}

func Test0140_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		".gitignore": buildOutputIgnore,
		"src/app.ts": "import ok from './local.js';\n// import bad from '../dist/foo.js';\n" +
			"/*\nconst z = require('pkg/dist/z.js');\n*/\nexport const app = ok;\n",
	})
	out, code := withPartner(t, repo, buildOutputCheck)
	sees(t, out, code, 0, "All 2 checks passed")
}

func Test0140_2(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"vitest": {vcfFullRun}})
	config := strings.Replace(vcfConfig("90"), "      thresholds:",
		"      // thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 },\n      thresholds:", 1)
	out, code := vcfRun(t, vcfRepo(t, map[string]string{"vitest.config.mjs": config}), rp.env)
	sees(t, out, code, 1, "Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.")
}

func Test0140_3(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{
		"pom.xml":       "<project>\n  <!-- " + strings.TrimSpace(javaVersionRelease("25")) + " -->\n</project>\n",
		".java-version": "# 17\n",
	})
	sees(t, out, code, 1, "no JDK version is pinned; add a .java-version file naming the release")
}

func Test0140_4(t *testing.T) {
	t.Parallel()
	out, code := withPartner(t, example(t, "happyRepo", inComments(strings.TrimSpace(longSentence))), "prose-budget")
	sees(t, out, code, 1, atEachFile("a sentence has 30 words (max 23)")...)
}

func Test0140_5(t *testing.T) {
	t.Parallel()
	marked := "// duplicate-code:ignore-start\n" + dupClone + "// duplicate-code:ignore-end\n"
	repo := example(t, "happyRepo", map[string]string{
		"src/a.js": marked + dupBlock("fa", 24, 3),
		"src/b.js": marked + dupBlock("fb", 24, 3),
	})
	out, code := withPartner(t, repo, "duplicate-code")
	sees(t, out, code, 0, "All 2 checks passed")
}

func Test0140_6(t *testing.T) {
	t.Parallel()
	workflow := "name: CI\non: push\njobs: # cspell:ignore foo\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: make test # cspell:disable-line\n"
	out, code := withPartner(t, example(t, "happyRepo", map[string]string{".github/workflows/ci.yml": workflow}), "workflow-reporting")
	sees(t, out, code, 1, ".github/workflows/ci.yml:4: job test has no step")
}
