package requirements

import (
	"testing"
)

// prettierJS is a formatted project with its own Prettier config, which keeps
// happyRepo's single-quoted front matter.
func prettierJS(files map[string]string) map[string]string {
	return with(map[string]string{
		"package.json":     "{}\n",
		".prettierrc.json": `{ "singleQuote": true }` + "\n",
		"app.js":           prettierFormatted,
	}, files)
}

// prettierFormatted and prettierMessy are one statement, formatted and not.
const (
	prettierFormatted = "const answer = { value: 42 };\n"
	prettierMessy     = "const answer={value:42}\n"
)

// prettierShared is formatted to the shared config's single quotes, which
// Prettier's own defaults would reformat.
const prettierShared = "const greeting = 'hi';\n"

// prettierPackageField is a formatted package.json holding the Prettier
// config, in place of a config file.
const prettierPackageField = "{\n  \"prettier\": {\n    \"singleQuote\": true\n  }\n}\n"

// What Prettier 3 prints: `--check` on formatted files, `--check` naming
// app.js, the same in a CI job that colors the [warn] prefix, and `--write`.
var (
	prettierClean = response{Stdout: "Checking formatting...\nAll matched files use Prettier code style!\n"}
	prettierWarn  = response{
		Stdout: "Checking formatting...\n",
		Stderr: "[warn] app.js\n[warn] Code style issues found in the above file. Run Prettier with --write to fix.\n",
		Exit:   1,
	}
	prettierWarnColored = response{
		Stdout: "Checking formatting...\n",
		Stderr: "\x1b[33m[warn]\x1b[39m app.js\n\x1b[33m[warn]\x1b[39m Code style issues found in the above file. Run Prettier with --write to fix.\n",
		Exit:   1,
	}
	prettierWritten = response{Stdout: "CHANGELOG.md 9ms (unchanged)\nREADME.md 4ms (unchanged)\napp.js 12ms\npackage.json 1ms (unchanged)\n"}
)

// prettierAnswers matches a response to calls holding every match string.
func prettierAnswers(r response, match ...string) response {
	r.Match = match
	return r
}

// prettierRun runs `fitness-install -- prettier <args>` on happyRepo with
// files written over it and a Prettier stand-in answering from saved.
func prettierRun(t *testing.T, files map[string]string, saved []response, args ...string) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	rp := standIn(t, map[string][]response{"prettier": saved})
	out, code := fitness(t, repo, rp.env, append([]string{"prettier"}, args...)...)
	return out, code, rp
}

// prettierCalledWith fails t unless fitness called Prettier with want and
// without each of not.
func prettierCalledWith(t *testing.T, rp replays, want []string, not ...string) {
	t.Helper()
	if !rp.called("prettier", want...) {
		t.Errorf("prettier was not called with %q; calls: %q", want, rp.calls("prettier"))
	}
	for _, n := range not {
		if rp.called("prettier", n) {
			t.Errorf("prettier was called with %q; calls: %q", n, rp.calls("prettier"))
		}
	}
}

func Test0052_1(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, prettierJS(map[string]string{".prettierrc.json": `{ "singleQuote": true, "semi": false }` + "\n"}), []response{prettierWarnColored})
	sees(t, out, code, 1, "✖ app.js")
	prettierCalledWith(t, rp, []string{"--check", "app.js"}, "--config")
}

func Test0052_2(t *testing.T) {
	t.Parallel()
	files := prettierJS(map[string]string{".prettierrc.json": "", "package.json": prettierPackageField})
	out, code, rp := prettierRun(t, files, []response{prettierAnswers(prettierWarn, "--config"), prettierClean})
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--check", "package.json"}, "--config")
}

func Test0052_3(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, map[string]string{"app.js": prettierMessy}, []response{prettierWarn})
	sees(t, out, code, 0, "All 1 checks passed")
	if calls := rp.calls("prettier"); len(calls) > 0 {
		t.Errorf("prettier was called without a package.json: %q", calls)
	}
}

func Test0052_4(t *testing.T) {
	t.Parallel()
	shared := prettierAnswers(prettierClean, "--config", "prettier.config.cjs")
	out, code, rp := prettierRun(t, prettierJS(map[string]string{".prettierrc.json": "", "app.js": prettierShared}), []response{shared, prettierWarn})
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--config", "prettier.config.cjs", "--check", "app.js"})
}

func Test0052_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", prettierJS(nil))
	write(t, repo, map[string]string{"app.js": prettierMessy, "lib.js": prettierFormatted})
	git(t, repo, "add", "lib.js")
	rp := standIn(t, map[string][]response{"prettier": {prettierAnswers(prettierWarn, "app.js"), prettierClean}})
	out, code := fitness(t, repo, rp.env, "prettier")
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--check lib.js"}, "app.js")
}

func Test0052_6(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, prettierJS(map[string]string{"app.js": prettierMessy}), []response{prettierAnswers(prettierWritten, "--write ."), prettierWarn}, "--write", ".")
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--write ."}, "--check")
}

func Test0052_7(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", map[string]string{".prettierignore": "README.md\n"}), nil, "prettier")
	sees(t, out, code, 1, ".prettierignore exists; Prettier checks every tracked file, so delete it and fix the findings instead")
}

func Test0052_8(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", prettierJS(nil)), []string{pathWithout(t, "prettier")}, "prettier")
	sees(t, out, code, 1, "Prettier not installed: npm install --save-dev prettier")
}
