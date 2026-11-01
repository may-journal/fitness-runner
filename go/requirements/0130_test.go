package requirements

import (
	"strings"
	"testing"
)

// rcStaged writes files over a fresh copy of given, stages them, and runs
// requirements-changed on the staged commit.
func rcStaged(t *testing.T, given, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", given)
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return fitness(t, repo, nil, "requirements-changed")
}

// rcPull commits files on top of a fresh copy of given, as a pull request
// branch off main, and runs requirements-changed on the pull request.
func rcPull(t *testing.T, given, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", given)
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	commit(t, repo, files, "feat(adder): double numbers")
	return fitness(t, repo, prEnv, "requirements-changed")
}

// rcAdder is goRepo's adder.go with one more function, a product change.
func rcAdder() string {
	return goRepo()["adder.go"] + "\n// Double returns twice a.\nfunc Double(a int) int {\n\treturn a + a\n}\n"
}

// rcPassed is what a passing single-check run prints.
const rcPassed = "All 1 checks passed"

func Test0130_1(t *testing.T) {
	t.Parallel()
	adder := map[string]string{"adder.go": rcAdder()}
	out, code := rcStaged(t, goRepo(), adder)
	sees(t, out, code, 1, "adder.go changed without a test and a requirement doc; change both with it")
	out, code = rcStaged(t, goRepo(), with(adder, map[string]string{"adder_test.go": edited(goRepo()["adder_test.go"])}))
	sees(t, out, code, 1, "adder.go changed without a requirement doc; change one with it")
	out, code = rcPull(t, goRepo(), adder)
	sees(t, out, code, 1, "adder.go changed without a test and a requirement doc")
	rcCodeWithComments(t)
}

func Test0130_2(t *testing.T) {
	t.Parallel()
	out, code := rcStaged(t, goRepo(), map[string]string{
		"adder.go":      rcAdder(),
		"adder_test.go": edited(goRepo()["adder_test.go"]),
		pairedDoc:       reworded(goRepo()[pairedDoc]),
	})
	sees(t, out, code, 0, rcPassed)
}

func Test0130_3(t *testing.T) {
	t.Parallel()
	adder := goRepo()["adder.go"]
	for _, files := range []map[string]string{
		{"README.md": readme + "\nMore words.\n"},
		{".fitnessrc.json": `{"proseBudget": {}}` + "\n"},
		{"CHANGELOG.md": "# Changelog\n\nNothing yet.\n"},
		{"adder.go": "", "sum.go": adder},
		{"adder.go": strings.Replace(adder, "\treturn a + b\n", "\n\treturn  a + b  \n", 1)},
	} {
		out, code := rcStaged(t, goRepo(), files)
		sees(t, out, code, 0, rcPassed)
	}
	rcCommentOnly(t)
}

func Test0130_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goRepo())
	git(t, repo, "checkout", "-q", "-b", "side")
	commit(t, repo, map[string]string{"adder.go": rcAdder()}, "feat(adder): double numbers")
	git(t, repo, "checkout", "-q", "-")
	commit(t, repo, map[string]string{"README.md": readme + "\nMore words.\n"}, "docs(readme): add words")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-commit", "--no-ff", "side")
	out, code := fitness(t, repo, nil, "requirements-changed")
	sees(t, out, code, 0, rcPassed)
}

func Test0130_5(t *testing.T) {
	t.Parallel()
	given := with(goRepo(), map[string]string{"three.go": "package adder\n\n// Three is three.\nconst Three = 3\n"})
	out, code := rcStaged(t, given, map[string]string{"three.go": ""})
	sees(t, out, code, 0, rcPassed)
}

func Test0130_6(t *testing.T) {
	t.Parallel()
	named := strings.Replace(goRepo()[pairedDoc], "by hand.", "by hand, with `adder.go`.", 1)
	out, code := rcStaged(t, with(goRepo(), map[string]string{pairedDoc: named}), map[string]string{"adder.go": ""})
	sees(t, out, code, 1, "adder.go was deleted, leaving requirement doc "+pairedDoc+" and test adder_test.go unchanged; change or delete them with it")
}

func Test0130_7(t *testing.T) {
	t.Parallel()
	given := with(goRepo(), map[string]string{pairedDoc: ""})
	out, code := rcStaged(t, given, map[string]string{"adder.go": rcAdder()})
	sees(t, out, code, 0, rcPassed)
}

func Test0130_8(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, with(goRepo(), map[string]string{".fitnessrc.json": `{"policy": "external", "checks": ["requirements-changed"]}` + "\n"}))
	write(t, repo, map[string]string{"adder.go": rcAdder()})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "feat(adder): double numbers\n\nRequirements: none, a helper no user calls yet")
	sees(t, out, code, 1, "adder.go changed without a test and a requirement doc; change both with it")
}

// rcCommented is goRepo's adder.go with every comment rewritten and one
// added after code. Its body is indented anew, as a comment edit may leave it.
func rcCommented() string {
	return "// Package adder sums numbers.\npackage adder\n\n// Add returns a plus b,\n// in either order.\nfunc Add(a, b int) int {\n    return a + b // the sum\n}\n"
}

// rcScript is a shell script, product code, with comments to rewrite.
const rcScript = "#!/bin/sh\n# greet says hello.\necho \"# not a comment\" # says it\n"

// rcCommentOnly proves that edits to comments alone, in Go and shell,
// pass staged and on a pull request.
func rcCommentOnly(t *testing.T) {
	t.Helper()
	given := with(goRepo(), map[string]string{"greet.sh": rcScript})
	files := map[string]string{
		"adder.go": rcCommented(),
		"greet.sh": "#!/bin/sh\n# greet prints a greeting.\n\necho \"# not a comment\"    # prints it\n",
	}
	out, code := rcStaged(t, given, files)
	sees(t, out, code, 0, rcPassed)
	out, code = rcPull(t, given, files)
	sees(t, out, code, 0, rcPassed)
}

// rcCodeWithComments proves that a code or directive change beside comment
// edits still fails, staged and on a pull request.
func rcCodeWithComments(t *testing.T) {
	t.Helper()
	given := with(goRepo(), map[string]string{"greet.sh": rcScript, "tag.go": "//go:build darwin\n\npackage adder\n"})
	for file, body := range map[string]string{
		"adder.go": strings.Replace(rcCommented(), "a + b", "b + a", 1),
		"greet.sh": strings.Replace(rcScript, "# not a comment\"", "# still not a comment\"", 1),
		"tag.go":   "//go:build linux\n\n// Package adder builds on Linux.\npackage adder\n",
	} {
		out, code := rcStaged(t, given, map[string]string{file: body})
		sees(t, out, code, 1, file+" changed without a test and a requirement doc; change both with it")
		out, code = rcPull(t, given, map[string]string{file: body})
		sees(t, out, code, 1, file+" changed without a test and a requirement doc")
	}
}
