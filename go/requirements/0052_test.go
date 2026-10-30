package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// prettierJS is a formatted project with its own Prettier config, which
// keeps single quotes.
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

func Test0052_9(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, prettierJS(map[string]string{".prettierrc.json": `{ "singleQuote": true, "semi": false }` + "\n"}), []response{prettierWarnColored})
	sees(t, out, code, 1, "✖ app.js")
	prettierCalledWith(t, rp, []string{"--check", "app.js"}, "--config")
	files := prettierJS(map[string]string{".prettierrc.json": "", "package.json": prettierPackageField})
	out, code, rp = prettierRun(t, files, []response{prettierAnswers(prettierWarn, "--config"), prettierClean})
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--check", "package.json"}, "--config")
}

// biomeReformat is real `biome format --reporter=github` output naming a
// file it would reformat.
const biomeReformat = "::error title=format,file=%s,line=1,endLine=1,col=2,endColumn=2::Formatter would have printed the following content:\n"

func Test0052_11(t *testing.T) {
	t.Parallel()
	files := map[string]string{"biome.jsonc": "{ \"formatter\": { \"enabled\": true } }\n"}
	var report strings.Builder
	for i := 1; i <= biomeMany; i++ {
		name := fmt.Sprintf("app%d.js", i)
		files[name] = prettierMessy
		fmt.Fprintf(&report, biomeReformat, name)
	}
	for _, major := range biomeMajors {
		files["package.json"] = fmt.Sprintf(biomePackage, major[0])
		repo := example(t, "happyRepo", files)
		reformat := response{
			Match:  []string{"format " + fmt.Sprintf(biomeFlags, major[2]), "app1.js"},
			Stdout: report.String(),
			Stderr: "format ━━━━━━━━━━\n\n  × Some errors were emitted while running checks.\n",
			Exit:   1,
		}
		rp := standIn(t, map[string][]response{"biome": {biomeVersion(major[1]), reformat}})
		out, code := fitness(t, repo, rp.env, "prettier")
		sees(t, out, code, 1, "✖ app1.js", "✖ app25.js")
		if !rp.called("biome", "format", "README.md") || strings.Contains(out, "Prettier not installed") {
			t.Errorf("Biome must check every tracked file in place of Prettier; calls %q:\n%s", rp.calls("biome"), out)
		}
	}
	biomeRejects(t, files, "prettier")
}

func Test0052_3(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, map[string]string{"app.js": prettierMessy}, []response{prettierWarn})
	sees(t, out, code, 0, "All 1 checks passed")
	if calls := rp.calls("prettier"); len(calls) > 0 {
		t.Errorf("prettier was called without a package.json: %q", calls)
	}
}

// prettierSynced are the files may-journal/.github syncs into every repo,
// formatted with Prettier's defaults: double-quoted front matter.
var prettierSynced = map[string]string{
	".github/ISSUE_TEMPLATE/plan.md": "---\nname: Plan\ntitle: \"[Plan] Short name of the work\"\n---\n\n> REPLACE-ME\n",
	".github/fitness-setup.md":       "# Fitness setup\n\nRun \"fitness\" before each push.\n",
}

// configFrom returns the file a Prettier call names after --config.
func configFrom(t *testing.T, call string) string {
	t.Helper()
	args := strings.Fields(call)
	i := slices.Index(args, "--config")
	if i < 0 || i+1 == len(args) {
		t.Fatalf("prettier was called without --config: %q", call)
	}
	raw, err := os.ReadFile(args[i+1])
	mustDo(t, err)
	return string(raw)
}

func Test0052_12(t *testing.T) {
	t.Parallel()
	files := prettierJS(with(map[string]string{".prettierrc.json": "", "app.js": prettierShared}, prettierSynced))
	shared := prettierAnswers(prettierClean, "--config", "prettier.config.cjs", "--check", "app.js", ".github/ISSUE_TEMPLATE/plan.md", ".github/fitness-setup.md")
	out, code, rp := prettierRun(t, files, []response{shared, prettierWarn})
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--config", "prettier.config.cjs", "--ignore-unknown", "--ignore-path", "--with-node-modules", "--check", "app.js"})
	calls := rp.calls("prettier")
	if len(calls) != 1 {
		t.Fatalf("want one prettier call, got %q", calls)
	}
	config := configFrom(t, calls[0])
	for _, want := range []string{"singleQuote: true", "'.github/ISSUE_TEMPLATE/**/*.md', '.github/fitness-setup.md'", "singleQuote: false", "fromRepoRoot"} {
		if !strings.Contains(config, want) {
			t.Errorf("the shared config must use single quotes and keep the synced files at Prettier defaults; missing %q in:\n%s", want, config)
		}
	}
}

// prettierLinkRefused is what Prettier 3 prints when a symbolic link is
// named on its command line.
var prettierLinkRefused = response{
	Match:  []string{"ci_scripts/ci_post_clone.sh"},
	Stderr: "[error] Explicitly specified pattern \"ci_scripts/ci_post_clone.sh\" is a symbolic link.\n",
	Exit:   2,
}

func Test0052_5(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, prettierJS(map[string]string{".fitnessrc.json": `{"policy": "external", "checks": ["prettier"]}` + "\n"}))
	write(t, repo, map[string]string{"app.js": prettierMessy, "ci_scripts/run.sh": "#!/bin/sh\necho ready\n"})
	mustDo(t, os.Symlink("run.sh", filepath.Join(repo, "ci_scripts", "ci_post_clone.sh")))
	rp := standIn(t, map[string][]response{"prettier": {prettierLinkRefused, prettierAnswers(prettierWarn, "app.js"), prettierClean}})
	env = append(env, rp.env...)
	user(t, repo, env, "add", "ci_scripts")
	out, code := user(t, repo, env, "commit", "-m", "ci(scripts): share one post-clone script")
	sees(t, out, code, 0, "ci(scripts): share one post-clone script")
	prettierCalledWith(t, rp, []string{"--check", "ci_scripts/run.sh"}, "app.js", "ci_scripts/ci_post_clone.sh")
}

func Test0052_6(t *testing.T) {
	t.Parallel()
	out, code, rp := prettierRun(t, prettierJS(map[string]string{"app.js": prettierMessy}), []response{prettierAnswers(prettierWritten, "--write ."), prettierWarn}, "--write", ".")
	sees(t, out, code, 0, "All 1 checks passed")
	prettierCalledWith(t, rp, []string{"--write ."}, "--check")
	files := prettierJS(map[string]string{".prettierrc.json": "", "biome.jsonc": "{}\n", "package.json": fmt.Sprintf(biomePackage, "2.4.15")})
	written := response{Match: []string{"format " + fmt.Sprintf(biomeFlags, "none") + " --write app.js"}, Stdout: "Formatted 1 file in 2ms. Fixed 1 file.\n"}
	biomeRun := standIn(t, map[string][]response{"biome": {biomeVersion("2.4.15"), written}})
	out, code = fitness(t, example(t, "happyRepo", files), biomeRun.env, "prettier", "--write", "--ignore-unknown", "app.js")
	sees(t, out, code, 0, "All 1 checks passed")
	for _, call := range biomeRun.calls("biome") {
		if strings.Contains(call, "--ignore-unknown") || strings.Contains(call, "README.md") {
			t.Errorf("Biome must get only its own flags and the given path; got %q", call)
		}
	}
	out, code = fitness(t, example(t, "happyRepo", files), biomeRun.env, "prettier", "--write", "--print-width", "100", "app.js")
	sees(t, out, code, 1, "Biome formats this repo and has no counterpart for Prettier's --print-width")
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
