package requirements

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func Test0131_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", shellCheck("house-rule", "", passing))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, allPassedWith(1))
	passedWith(t, out, "house-rule", "1")
}

func Test0131_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goCheck(t, "go-rule", t.TempDir()))
	out, code := fitness(t, repo, nil, "check", "go-rule")
	sees(t, out, code, 0, "fitness: building fitness/go-rule with go", "All 1 checks passed")
	passedWith(t, out, "go-rule", "3")
	out, code = fitness(t, repo, nil, "check", "go-rule")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "go-rule", "3")
	if strings.Contains(out, "building fitness/go-rule") {
		t.Errorf("a second run of an unchanged Go check built it again:\n%s", out)
	}
}

func Test0131_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goCheck(t, "go-rule", t.TempDir()))
	out, code := fitness(t, repo, []string{pathWithout(t, "go")}, "check", "go-rule")
	sees(t, out, code, 1, "local check fitness/go-rule is Go source, and go is not on PATH; install Go from https://go.dev/dl")
}

func Test0131_4(t *testing.T) {
	t.Parallel()
	goRule := goCheck(t, "go-rule", t.TempDir())
	for want, files := range map[string]map[string]string{
		"local check fitness/house-rule has no README.md":                                              {"fitness/house-rule/check": verdictScript(passing)},
		"local check fitness/house-rule has no program; add an executable check or a main.go":          {"fitness/house-rule/README.md": "# House rule\n\nKeeps the house.\n"},
		"local check fitness/go-rule is Go source, so fitness/go.mod must exist":                       with(goRule, map[string]string{"fitness/go.mod": ""}),
		"local check fitness/go-rule does not build; fix it so `go -C fitness build ./go-rule` passes": with(goRule, map[string]string{"fitness/go-rule/main.go": "package main\n\nfunc main() { undefined() }\n"}),
	} {
		out, code := fitness(t, example(t, "happyRepo", files), nil)
		sees(t, out, code, 1, want)
	}
}

func Test0131_5(t *testing.T) {
	t.Parallel()
	files := with(shellCheck("house-rule", "", `{"ok": false, "errors": ["the house rule is broken"], "filesChecked": 1}`),
		with(goCheck(t, "go-rule", t.TempDir()), map[string]string{
			"notes.md": "# Notes\n\nTODO: write these.\n",
			"fitness/go-rule/main_test.go": "package main\n\nimport (\n\t\"testing\"\n\n\t\"github.com/may-journal/fitness-runner/go/check\"\n)\n\nfunc TestRule(t *testing.T) { check.Test(t, rule) }\n\n" +
				"func TestFixture(t *testing.T) {\n\tif check.Judge(t, rule, check.Repo(t, map[string]string{\"a.md\": \"TODO\"})).Ok {\n\t\tt.Error(\"a TODO must fail\")\n\t}\n}\n",
		}))
	repo := example(t, "happyRepo", files)
	out, code := fitness(t, repo, nil, "check", "house-rule")
	sees(t, out, code, 1, "the house rule is broken")
	if row(out, "go-rule") != "" || row(out, "markdown-links") != "" {
		t.Errorf("fitness check house-rule ran other checks too:\n%s", out)
	}
	unitTested(t, repo)
}

// unitTested asserts that go test in repo's fitness folder passes the check's
// fixture test and fails its repo test on the finding.
func unitTested(t *testing.T, repo string) {
	t.Helper()
	out, failed := goTestFitness(t, repo)
	if !failed || !strings.Contains(out, "notes.md: holds a TODO") || strings.Contains(out, "FAIL: TestFixture") {
		t.Errorf("go test in fitness/ must pass the fixture test and fail on the repo's finding:\n%s", out)
	}
}

// goTestFitness runs `go -C fitness test ./...` in repo, as its developer does, and
// returns what it printed and whether it failed.
func goTestFitness(t *testing.T, repo string) (string, bool) {
	t.Helper()
	cmd := exec.Command("go", "-C", "fitness", "test", "./...")
	cmd.Dir, cmd.Env = repo, userEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err != nil
}

func Test0131_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", shellCheck("house-rule", "", passing))
	got := listedCheck{}
	for _, c := range listJSON(t, repo) {
		if c.Name == "house-rule" {
			got = c
		}
	}
	if got.Scope != "local" || got.Summary != "Keeps the house in order." {
		t.Errorf("fitness list shows house-rule with scope %q and summary %q, want local and the README's first sentence", got.Scope, got.Summary)
	}
	out, code := fitness(t, repo, nil, "explain", "house-rule")
	sees(t, out, code, 0, "# house-rule", "Scope: local", "Keeps the house in order. Each room stays tidy.", "## Fix", "Tidy the room the error names.")
}

func Test0131_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", shellCheck("slow-rule", `{"timeoutMs": 300}`, ""))
	for _, c := range listJSON(t, repo) {
		if c.Name == "slow-rule" && c.BudgetMs != 300 {
			t.Errorf("fitness list shows slow-rule with budget %dms, want the 300ms it declares", c.BudgetMs)
		}
	}
	out, code := fitness(t, repo, nil, "check", "slow-rule")
	sees(t, out, code, 1, "Check timed out after 0.3s")
	if !strings.Contains(row(out, "slow-rule"), "failed") {
		t.Errorf("a check past the budget it declares must report red, got row %q", row(out, "slow-rule"))
	}
}

func Test0131_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", shellCheck("new-rule", `{"minFitness": "999.0.0"}`, passing))
	out, code := fitness(t, repo, nil, "check", "new-rule")
	sees(t, out, code, 1, "check new-rule needs fitness 999.0.0 or later", "--version=999.0.0")
}

// houseReadme is a folder check's README: a title, the rule, and the fix.
const houseReadme = "# House rule\n\nKeeps the house in order. Each room stays tidy.\n\n## Fix\n\nTidy the room the error names.\n"

// shellCheck is a repo whose fitness folder holds one shell check, name,
// with a README. describe, when set, is the JSON it answers --describe
// with; otherwise it ignores the handshake. verdict is what it prints when
// it runs, and an empty verdict makes it hang.
func shellCheck(name, describe, verdict string) map[string]string {
	script := verdictScript(verdict)
	if describe != "" {
		script = strings.Replace(script, "#!/bin/sh\n", "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '"+describe+"'; exit 0; }\n", 1)
	}
	return map[string]string{"fitness/" + name + "/README.md": houseReadme, "fitness/" + name + "/check": script}
}

// verdictScript is a check program that prints verdict, or hangs when it
// is empty.
func verdictScript(verdict string) string {
	if verdict == "" {
		return "#!/bin/sh\nexec /bin/sleep 30\n"
	}
	return "#!/bin/sh\necho '" + verdict + "'\n"
}

// goCheck is a repo whose fitness folder holds one Go check, name, built
// on the check SDK from this checkout. It fails each Markdown file that
// holds a TODO. marker goes into its source, so a test that needs a cold
// build gets source no other test built.
func goCheck(t *testing.T, name, marker string) map[string]string {
	t.Helper()
	sdk, err := filepath.Abs("..")
	mustDo(t, err)
	return map[string]string{
		"fitness/go.mod": "module example.com/repo/fitness\n\ngo 1.26\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\n" +
			"replace github.com/may-journal/fitness-runner/go => " + sdk + "\n",
		"fitness/" + name + "/README.md": "# No to-do markers\n\nFails a Markdown file that holds a to-do marker.\n",
		"fitness/" + name + "/main.go":   goRuleSource(marker),
	}
}

// goRuleSource is the Go check goCheck writes, with marker in a comment.
func goRuleSource(marker string) string {
	return "// Package main is a check built from " + marker + ".\npackage main\n\n" +
		"import (\n\t\"strings\"\n\n\t\"github.com/may-journal/fitness-runner/go/check\"\n)\n\n" +
		"var rule = check.Check{Run: func(ctx *check.Context) (check.Result, error) {\n" +
		"\tfiles := ctx.InScope(ctx.Files(\".md\"))\n\tvar found []check.Finding\n" +
		"\tfor _, f := range files {\n\t\ttext, err := ctx.Read(f)\n\t\tif err != nil {\n\t\t\treturn check.Result{}, err\n\t\t}\n" +
		"\t\tif strings.Contains(text, \"TODO\") {\n\t\t\tfound = append(found, check.Finding{File: f, Rule: \"no-todo\", Message: \"holds a TODO\", Fix: \"do it\"})\n\t\t}\n\t}\n" +
		"\treturn check.Found(len(files), found...), nil\n}}\n\n" +
		"func main() { check.Main(rule) }\n"
}
