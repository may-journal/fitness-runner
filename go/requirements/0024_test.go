package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// eslintMissing is the install hint shown when no eslint binary resolves.
const eslintMissing = "eslint not found in node_modules/.bin (walking up from the repo root) or on PATH. Run: npm install --save-dev eslint"

// eslintPackage marks the repo as a JavaScript project.
const eslintPackage = "{\"name\": \"app\", \"private\": true}\n"

// eslintUnsorted breaks the shared config's `sort-keys` rule.
const eslintUnsorted = "export const o = { z: 1, a: 2 };\n"

// eslintSorted is eslintUnsorted with its keys in order.
const eslintSorted = "export const o = { a: 1, z: 2 };\n"

// eslintTyped is a clean TypeScript file.
const eslintTyped = "export const b = 2;\n"

// eslintArgs are the flags fitness passes eslint before the file list.
const eslintArgs = "--no-error-on-unmatched-pattern --format json"

// eslintClean is real `eslint --format json` output for a file with no
// findings; %s is its absolute path.
const eslintClean = `{"filePath":%s,"messages":[],"suppressedMessages":[],"errorCount":0,"fatalErrorCount":0,"warningCount":0,"fixableErrorCount":0,"fixableWarningCount":0,"usedDeprecatedRules":[]}`

// eslintSortKeys is real output for eslintUnsorted under the shared config.
const eslintSortKeys = `{"filePath":%s,"messages":[{"ruleId":"sort-keys","severity":2,"message":"Expected object keys to be in natural ascending order. 'a' should be before 'z'.","line":1,"column":26,"nodeType":"Property","messageId":"sortKeys","endLine":1,"endColumn":30}],"suppressedMessages":[],"errorCount":1,"fatalErrorCount":0,"warningCount":0,"fixableErrorCount":0,"fixableWarningCount":0,"source":"export const o = { z: 1, a: 2 };\n","usedDeprecatedRules":[]}`

// eslintConsole is real output for a `console.log` call under a config
// that only warns on it.
const eslintConsole = `{"filePath":%s,"messages":[{"ruleId":"no-console","severity":1,"message":"Unexpected console statement.","line":2,"column":1,"nodeType":"MemberExpression","messageId":"unexpected","endLine":2,"endColumn":12,"suggestions":[{"messageId":"removeConsole","data":{"propertyName":"log"},"fix":{"range":[33,48],"text":""},"desc":"Remove the console.log()."}]}],"suppressedMessages":[],"errorCount":0,"fatalErrorCount":0,"warningCount":1,"fixableErrorCount":0,"fixableWarningCount":0,"source":"export const o = { z: 1, a: 2 };\nconsole.log(o);\n","usedDeprecatedRules":[]}`

// eslintIgnored is real output for a file passed on the command line that
// the config ignores.
const eslintIgnored = `{"filePath":%s,"messages":[{"ruleId":null,"fatal":false,"severity":1,"message":"File ignored because of a matching ignore pattern. Use \"--no-ignore\" to disable file ignore settings or use \"--no-warn-ignored\" to suppress this warning.","nodeType":null}],"suppressedMessages":[],"errorCount":0,"fatalErrorCount":0,"warningCount":1,"fixableErrorCount":0,"fixableWarningCount":0,"usedDeprecatedRules":[]}`

// eslintShared is a JavaScript project carrying its own copy of the shared
// config files fitness ships, the way a consumer repo keeps them.
func eslintShared(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	all := map[string]string{"package.json": eslintPackage}
	for _, name := range []string{"eslint.config.mjs", "eslint.base.mjs"} {
		body, err := os.ReadFile(filepath.Join("..", "internal", "sharedconf", "config", name))
		mustDo(t, err)
		all[name] = string(body)
	}
	for name, body := range files {
		all[name] = body
	}
	return all
}

// eslintReport is eslint's JSON report: each template filled with the
// absolute path of the matching file under repo, in order.
func eslintReport(repo string, results ...[2]string) string {
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = fmt.Sprintf(r[0], strconv.Quote(filepath.Join(repo, r[1])))
	}
	return "[" + strings.Join(parts, ",") + "]\n"
}

// eslintRun runs `fitness-install -- eslint` in repo with eslint standing
// in, answering a call with files and the repo's own config with a report.
// It matches the config path without macOS's /private prefix, so either
// spelling of the temporary folder matches.
func eslintRun(t *testing.T, repo string, files string, exit int, results ...[2]string) (string, int, replays) {
	t.Helper()
	real, err := filepath.EvalSymlinks(repo)
	mustDo(t, err)
	rp := standIn(t, map[string][]response{"eslint": {{
		Match:  []string{"--config ", strings.TrimPrefix(real, "/private") + "/eslint.config.mjs " + eslintArgs + " " + files},
		Stdout: eslintReport(real, results...),
		Exit:   exit,
	}}})
	out, code := fitness(t, repo, rp.env, "eslint")
	return out, code, rp
}

// eslintStaged commits a project with the shared config and an unsorted
// app.js, stages files, and returns the repo.
func eslintStaged(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintUnsorted}))
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return repo
}

// eslintPassedWith asserts the eslint row shows a pass that judged files.
func eslintPassedWith(t *testing.T, out, files string) {
	t.Helper()
	if !regexp.MustCompile(`│ passed\s+│ ` + files + `\s+│`).MatchString(row(out, "eslint")) {
		t.Errorf("eslint must pass with %s files, got row %q", files, row(out, "eslint"))
	}
}

// eslintFinds asserts eslint failed and reported want. The table wraps the
// long absolute path at any character, so spaces are ignored when matching.
func eslintFinds(t *testing.T, out string, code int, want string) {
	t.Helper()
	squash := strings.NewReplacer("│", "", " ", "", "\n", "")
	if !strings.Contains(squash.Replace(out), squash.Replace(want)) {
		t.Errorf("output is missing %q:\n%s", want, out)
	}
	sees(t, out, code, 1)
}

// eslintUncalled asserts fitness never ran eslint.
func eslintUncalled(t *testing.T, rp replays) {
	t.Helper()
	if calls := rp.calls("eslint"); len(calls) > 0 {
		t.Errorf("eslint must not run, got calls %q", calls)
	}
}

func Test0024_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"app.js": eslintUnsorted})
	rp := standIn(t, map[string][]response{"eslint": {{Stdout: "[]\n"}}})
	out, code := fitness(t, repo, rp.env, "eslint")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "eslint")
	eslintUncalled(t, rp)
}

func Test0024_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintUnsorted})), nil, "eslint")
	sees(t, out, code, 1, eslintMissing)
}

func Test0024_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintSorted, "src/b.ts": eslintTyped}))
	out, code, _ := eslintRun(t, repo, "app.js eslint.base.mjs eslint.config.mjs src/b.ts", 0,
		[2]string{eslintClean, "app.js"}, [2]string{eslintClean, "eslint.base.mjs"},
		[2]string{eslintClean, "eslint.config.mjs"}, [2]string{eslintClean, "src/b.ts"})
	sees(t, out, code, 0, "All 1 checks passed")
	eslintPassedWith(t, out, "4")
}

func Test0024_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintUnsorted}))
	out, code, _ := eslintRun(t, repo, "app.js eslint.base.mjs eslint.config.mjs", 1,
		[2]string{eslintSortKeys, "app.js"}, [2]string{eslintClean, "eslint.base.mjs"},
		[2]string{eslintClean, "eslint.config.mjs"})
	eslintFinds(t, out, code, "/app.js:1:26 - Expected object keys to be in natural ascending order. 'a' should be before 'z'. (sort-keys)")
}

func Test0024_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"package.json":      eslintPackage,
		"eslint.config.mjs": "export default [{ rules: { 'no-console': 'warn' } }];\n",
		"app.js":            eslintUnsorted + "console.log(o);\n",
	})
	out, code, _ := eslintRun(t, repo, "app.js eslint.config.mjs", 0,
		[2]string{eslintConsole, "app.js"}, [2]string{eslintClean, "eslint.config.mjs"})
	eslintFinds(t, out, code, "/app.js:2:1 - Unexpected console statement. (no-console)")
	if strings.Contains(out, "sort-keys") {
		t.Errorf("the repo's own config must replace the shared one:\n%s", out)
	}
}

func Test0024_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"package.json":      eslintPackage,
		"eslint.config.mjs": "export default [{ ignores: ['skip.js'] }];\n",
		"skip.js":           eslintSorted,
	})
	out, code, _ := eslintRun(t, repo, "eslint.config.mjs skip.js", 0,
		[2]string{eslintClean, "eslint.config.mjs"}, [2]string{eslintIgnored, "skip.js"})
	eslintFinds(t, out, code, "/skip.js - ESLint ignores this tracked file; remove the ignore pattern so it is linted")
}

func Test0024_7(t *testing.T) {
	t.Parallel()
	repo := eslintStaged(t, map[string]string{"src/b.ts": eslintTyped, "README.md": readme + "\nOne more short line.\n"})
	out, code, rp := eslintRun(t, repo, "src/b.ts", 0, [2]string{eslintClean, "src/b.ts"})
	sees(t, out, code, 0, "All 1 checks passed")
	eslintPassedWith(t, out, "1")
	if rp.called("eslint", "app.js") {
		t.Errorf("eslint must lint only the staged file, got calls %q", rp.calls("eslint"))
	}
}

func Test0024_8(t *testing.T) {
	t.Parallel()
	repo := eslintStaged(t, map[string]string{"README.md": readme + "\nOne more short line.\n"})
	rp := standIn(t, map[string][]response{"eslint": {{Stdout: "[]\n"}}})
	out, code := fitness(t, repo, rp.env, "eslint")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "eslint")
	eslintUncalled(t, rp)
}
