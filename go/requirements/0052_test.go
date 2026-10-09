package requirements

import (
	"os"
	"path/filepath"
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

// prettierRun runs `fitness-install -- prettier <args>` on happyRepo with
// files written over it and the real Prettier installed.
func prettierRun(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	withJSTools(t, repo)
	return fitness(t, repo, nil, append([]string{"prettier"}, args...)...)
}

func Test0052_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", prettierJS(map[string]string{".prettierrc.json": `{ "singleQuote": true, "semi": false }` + "\n"}))
	withJSTools(t, repo)
	// FORCE_COLOR makes Prettier color its output, as it does in CI.
	out, code := fitness(t, repo, []string{"FORCE_COLOR=1"}, "prettier")
	sees(t, out, code, 1, "✖ app.js")
}

func Test0052_2(t *testing.T) {
	t.Parallel()
	out, code := prettierRun(t, prettierJS(map[string]string{".prettierrc.json": "", "package.json": prettierPackageField}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0052_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", map[string]string{"app.js": prettierMessy}), nil, "prettier")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0052_4(t *testing.T) {
	t.Parallel()
	out, code := prettierRun(t, prettierJS(map[string]string{".prettierrc.json": "", "app.js": prettierShared}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0052_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", prettierJS(nil))
	withJSTools(t, repo)
	write(t, repo, map[string]string{"app.js": prettierMessy, "lib.js": prettierFormatted})
	git(t, repo, "add", "lib.js")
	out, code := fitness(t, repo, nil, "prettier")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0052_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", prettierJS(map[string]string{"app.js": prettierMessy}))
	withJSTools(t, repo)
	out, code := fitness(t, repo, nil, "prettier", "--write", ".")
	sees(t, out, code, 0, "All 1 checks passed")
	if got, _ := os.ReadFile(filepath.Join(repo, "app.js")); string(got) != prettierFormatted {
		t.Errorf("app.js = %q, want it rewritten to %q", got, prettierFormatted)
	}
}

func Test0052_7(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", map[string]string{".prettierignore": "README.md\n"}), nil, "prettier")
	sees(t, out, code, 1, ".prettierignore exists; Prettier checks every tracked file, so delete it and fix the findings instead")
}

func Test0052_8(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", prettierJS(nil)), nil, "prettier")
	sees(t, out, code, 1, "Prettier not installed: npm install --save-dev prettier")
}
