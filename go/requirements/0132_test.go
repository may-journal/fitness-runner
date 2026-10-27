package requirements

import (
	"strings"
	"testing"
)

func Test0132_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", adrRepo()), nil)
	sees(t, out, code, 0, allPassedWith(1))
	passedWith(t, out, "adr-checks", "1")
	passedWith(t, out, "adr-0001", "1")
}

func Test0132_2(t *testing.T) {
	t.Parallel()
	apart := "docs/architecture/adr/0001-small-files.md"
	out, code := fitness(t, example(t, "happyRepo", with(adrRepo(), map[string]string{apart: adrText("")})), nil)
	sees(t, out, code, 1, apart+": ADR 0001 lives apart from its check; move it into fitness/adr-0001/README.md, beside the check that keeps it, and delete this file")
}

func Test0132_3(t *testing.T) {
	t.Parallel()
	bare := with(adrRepo(), map[string]string{adrPath: "# Small files\n\nKeeps every Markdown file small.\n\n## Decision\n\n1. Every Markdown file stays small.\n"})
	out, code := fitness(t, example(t, "happyRepo", bare), nil)
	sees(t, out, code, 1, adrPath+": is no ADR, as it lacks ## Context and ## Consequences; write the decision this check keeps under ## Context, ## Decision and ## Consequences")
}

func Test0132_4(t *testing.T) {
	t.Parallel()
	retired := with(shellCheck("house-rule", "", passing), map[string]string{adrPath: adrText("Status: Superseded by a later decision.\n\n")})
	out, code := fitness(t, example(t, "happyRepo", retired), nil)
	sees(t, out, code, 0, allPassedWith(1))
	passedWith(t, out, "adr-checks", "1")
}

func Test0132_5(t *testing.T) {
	t.Parallel()
	want := adrPath + " changed its decision without its check fitness/adr-0001; change both together"
	repo := example(t, "happyRepo", adrRepo())
	write(t, repo, map[string]string{adrPath: strings.Replace(adrRepo()[adrPath], "stays small", "stays short", 1)})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, "adr-checks")
	sees(t, out, code, 1, want)
	repo = example(t, "happyRepo", adrRepo())
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	commit(t, repo, map[string]string{adrPath: strings.Replace(adrRepo()[adrPath], "stays small", "stays brief", 1)}, "docs(adr): reword 0001")
	out, code = fitness(t, repo, prEnv, "adr-checks")
	sees(t, out, code, 1, want)
}

func Test0132_6(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", provenRepo()), nil)
	sees(t, out, code, 0, "All 2 checks passed")
	passedWith(t, out, "req-0002", "1")
}

func Test0132_7(t *testing.T) {
	t.Parallel()
	orphan := with(provenRepo(), map[string]string{"fitness/req-0003/README.md": houseReadme, "fitness/req-0003/check": verdictScript(passing)})
	out, code := fitness(t, example(t, "happyRepo", orphan), nil, "requirements")
	sees(t, out, code, 1, "fitness/req-0003 proves requirement 0003, which no doc defines; write the doc or delete the check")
	out, code = stagedRun(t, provenRepo(), map[string]string{"fitness/req-0002/check": verdictScript(`{"ok": true, "errors": [], "filesChecked": 2}`)})
	sees(t, out, code, 1, "fitness/req-0002/check changed without its requirement doc docs/requirements/0002-adds-again.md; change both together")
}

func Test0132_8(t *testing.T) {
	t.Parallel()
	empty := `{"ok": true, "errors": [], "filesChecked": 0}`
	for check, files := range map[string]map[string]string{
		"adr-0001": with(adrRepo(), adrCheck("0001", empty)),
		"req-0002": with(provenRepo(), map[string]string{"fitness/req-0002/check": verdictScript(empty)}),
	} {
		out, code := fitness(t, example(t, "happyRepo", files), nil, "check", check)
		sees(t, out, code, 1, check+" judged no files, so its scan is broken")
	}
}

// adrPath is the one ADR adrRepo holds: the README of its check.
const adrPath = "fitness/adr-0001/README.md"

// adrText is ADR 0001 with status, a line or more that opens it, and the
// sections that state its decision.
func adrText(status string) string {
	return "# 0001 — Small files\n\nSmall files read fast.\n\n" + status +
		"## Context\n\nLong files hide their point.\n\n## Decision\n\n1. Every Markdown file stays small.\n\n" +
		"## Consequences\n\n1. Readers find the point fast.\n"
}

// adrRepo is a repo with ADR 0001 as the README of its check.
func adrRepo() map[string]string {
	return adrCheck("0001", passing)
}

// adrCheck is the check folder fitness/adr-<number>, whose README is ADR
// 0001 and whose program prints verdict.
func adrCheck(number, verdict string) map[string]string {
	folder := "fitness/adr-" + number + "/"
	return map[string]string{
		folder + "README.md": adrText(""),
		folder + "check":     verdictScript(verdict),
	}
}

// provenRepo is twoRepo whose requirement 0002 a check in fitness/req-0002
// proves, with no test of its own.
func provenRepo() map[string]string {
	return with(twoRepo(), map[string]string{
		".fitnessrc.json":            `{"policy": "external", "checks": ["requirements", "req-0002"]}` + "\n",
		"reqs/0001_0002_test.go":     "",
		"reqs/0001_test.go":          testFile("reqs", "Test0001_1"),
		"fitness/req-0002/README.md": "# Doubles\n\nProves [requirement 0002](../../docs/requirements/0002-adds-again.md).\n",
		"fitness/req-0002/check":     verdictScript(passing),
	})
}
