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
	for want, files := range map[string]map[string]string{
		adrPath + ": ADR 0001 has no check; add fitness/adr-0001/ with a README.md that links this ADR": with(shellCheck("house-rule", "", passing), map[string]string{adrPath: adrText("")}),
		"fitness/adr-0002: no ADR 0002 exists; delete the check or give it its ADR's number":            with(adrRepo(), adrCheck("0002", passing)),
	} {
		out, code := fitness(t, example(t, "happyRepo", files), nil)
		sees(t, out, code, 1, want)
	}
}

func Test0132_3(t *testing.T) {
	t.Parallel()
	unlinked := with(adrRepo(), map[string]string{
		adrPath:                      adrText(""),
		"fitness/adr-0001/README.md": "# Small files\n\nKeeps every Markdown file small.\n",
	})
	out, code := fitness(t, example(t, "happyRepo", unlinked), nil)
	sees(t, out, code, 1, adrPath+": link its check fitness/adr-0001/README.md", "fitness/adr-0001/README.md: link "+adrPath)
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
	repo := example(t, "happyRepo", adrRepo())
	write(t, repo, map[string]string{adrPath: strings.Replace(adrRepo()[adrPath], "stays small", "stays short", 1)})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, "adr-checks")
	sees(t, out, code, 1, adrPath+" changed without its check fitness/adr-0001; change both together")
	repo = example(t, "happyRepo", adrRepo())
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	commit(t, repo, map[string]string{"fitness/adr-0001/check": verdictScript(`{"ok": true, "errors": [], "filesChecked": 2}`)}, "fix(adr-0001): count both files")
	out, code = fitness(t, repo, prEnv, "adr-checks")
	sees(t, out, code, 1, "fitness/adr-0001/check changed without its ADR "+adrPath+"; change both together")
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

// adrPath is the one ADR adrRepo holds.
const adrPath = "docs/architecture/adr/0001-small-files.md"

// adrText is ADR 0001 with status, a line or more that opens it, and a
// link to its check when it is paired.
func adrText(status string) string {
	return "# 0001 — Small files\n\n" + status + "## Decision\n\n1. Every Markdown file stays small.\n"
}

// adrRepo is a repo with ADR 0001 and its check, each linking the other.
func adrRepo() map[string]string {
	return with(adrCheck("0001", passing), map[string]string{
		adrPath: adrText("Check: [fitness/adr-0001](../../../fitness/adr-0001/README.md)\n\n"),
	})
}

// adrCheck is the check folder fitness/adr-<number> linking ADR 0001 and
// printing verdict.
func adrCheck(number, verdict string) map[string]string {
	folder := "fitness/adr-" + number + "/"
	return map[string]string{
		folder + "README.md": "# Small files\n\nKeeps [ADR 0001](../../" + adrPath + ").\n",
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
