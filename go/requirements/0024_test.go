package requirements

import (
	"os"
	"path/filepath"
	"regexp"
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

// eslintRun commits files into happyRepo, links the real eslint into it,
// and runs `fitness-install -- eslint`.
func eslintRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	withJSTools(t, repo)
	return fitness(t, repo, nil, "eslint")
}

// eslintStaged commits a project with the shared config and an unsorted
// app.js, links the real eslint, stages files, and runs eslint on them.
func eslintStaged(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintUnsorted}))
	withJSTools(t, repo)
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return fitness(t, repo, nil, "eslint")
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

func Test0024_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", map[string]string{"app.js": eslintUnsorted}), nil, "eslint")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "eslint")
}

func Test0024_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", eslintShared(t, map[string]string{"app.js": eslintUnsorted})), nil, "eslint")
	sees(t, out, code, 1, eslintMissing)
}

func Test0024_3(t *testing.T) {
	t.Parallel()
	out, code := eslintRun(t, eslintShared(t, map[string]string{"app.js": eslintSorted, "src/b.ts": eslintTyped}))
	sees(t, out, code, 0, "All 1 checks passed")
	eslintPassedWith(t, out, "4")
}

func Test0024_4(t *testing.T) {
	t.Parallel()
	out, code := eslintRun(t, eslintShared(t, map[string]string{"app.js": eslintUnsorted}))
	eslintFinds(t, out, code, "/app.js:1:26 - Expected object keys to be in natural ascending order. 'a' should be before 'z'. (sort-keys)")
}

func Test0024_5(t *testing.T) {
	t.Parallel()
	out, code := eslintRun(t, map[string]string{
		"package.json":      eslintPackage,
		"eslint.config.mjs": "export default [{ rules: { 'no-console': 'warn' } }];\n",
		"app.js":            eslintUnsorted + "console.log(o);\n",
	})
	eslintFinds(t, out, code, "/app.js:2:1 - Unexpected console statement. (no-console)")
	if strings.Contains(out, "sort-keys") {
		t.Errorf("the repo's own config must replace the shared one:\n%s", out)
	}
}

func Test0024_6(t *testing.T) {
	t.Parallel()
	out, code := eslintRun(t, map[string]string{
		"package.json":      eslintPackage,
		"eslint.config.mjs": "export default [{ ignores: ['skip.js'] }];\n",
		"skip.js":           eslintSorted,
	})
	eslintFinds(t, out, code, "/skip.js - ESLint ignores this tracked file; remove the ignore pattern so it is linted")
}

func Test0024_7(t *testing.T) {
	t.Parallel()
	out, code := eslintStaged(t, map[string]string{"src/b.ts": eslintTyped, "README.md": readme + "\nOne more short line.\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	eslintPassedWith(t, out, "1")
}

func Test0024_8(t *testing.T) {
	t.Parallel()
	out, code := eslintStaged(t, map[string]string{"README.md": readme + "\nOne more short line.\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "eslint")
}
