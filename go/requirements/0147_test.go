package requirements

import (
	"strings"
	"testing"
)

// adrAsIssue runs `fitness-install -- adr-as-issue` on happyRepo with files
// written over it.
func adrAsIssue(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "adr-as-issue")
}

// toADRIssue is the fix every ADR file's error names.
const toADRIssue = "open an `ADR` issue instead; `fitness adr-migrate` opens one per file and points each link at it"

// adrFile is an ADR as a file in an `adr` folder holds it.
const adrFile = "# ADR 0001 — Use SQLite\n\nStatus: accepted\n\n## Context\n\nWe sync offline.\n"

// inADRFolder is the error line for an ADR file in an `adr` folder.
func inADRFolder(path string) string {
	return path + ": a Markdown file in an `adr` folder is an ADR; " + toADRIssue
}

// heldDecision is a check README that still holds its decision.
const heldDecision = "# 0001 — Small files\n\nSmall files read fast.\n\n## Context\n\nLong files hide their point.\n\n" +
	"## Decision\n\n1. Every Markdown file stays small.\n\n## Consequences\n\n1. Readers find the point fast.\n\n" +
	"## Check\n\nFails a long Markdown file.\n"

// heldCheck is ADR 0001's check folder, its README holding the decision.
func heldCheck() map[string]string {
	return map[string]string{"fitness/adr-0001/README.md": heldDecision, "fitness/adr-0001/check": verdictScript(passing)}
}

func Test0147_1(t *testing.T) {
	t.Parallel()
	out, code := adrAsIssue(t, map[string]string{
		"docs/architecture/adr/0001-use-sqlite.md": adrFile,
		"docs/decisions/ADR/0002-sync.md":          adrFile,
		"docs/architecture/adr/README.md":          "# ADRs\n\n- [Use SQLite](0001-use-sqlite.md)\n",
	})
	sees(t, out, code, 1, inADRFolder("docs/decisions/ADR/0002-sync.md"), inADRFolder("docs/architecture/adr/0001-use-sqlite.md"))
	if strings.Contains(out, "adr/README.md:") {
		t.Errorf("the adr folder's README lists ADRs, yet it was reported:\n%s", out)
	}
}

func Test0147_2(t *testing.T) {
	t.Parallel()
	out, code := adrAsIssue(t, map[string]string{
		"docs/storage.md":    "# ADR 12: Use SQLite\n\nWe sync offline.\n",
		"docs/adr-checks.md": "# adr-checks\n\nPairs each decision with a check.\n",
		"docs/guide.md":      "# Guide\n\n## ADR 3\n\nA later heading.\n",
	})
	sees(t, out, code, 1, "docs/storage.md: a title starting with ADR makes the file an ADR; "+toADRIssue)
	for _, kept := range []string{"docs/adr-checks.md", "docs/guide.md"} {
		if strings.Contains(out, kept+":") {
			t.Errorf("%s has no title starting with the word ADR, yet it was reported:\n%s", kept, out)
		}
	}
}

func Test0147_3(t *testing.T) {
	t.Parallel()
	out, code := adrAsIssue(t, with(heldCheck(), map[string]string{
		"docs/writing-checks.md": "# Writing checks\n\n```markdown\n" + heldDecision + "```\n",
	}))
	sees(t, out, code, 1, "fitness/adr-0001/README.md: `## Context`, `## Decision` and `## Consequences` sections make the file an ADR; "+toADRIssue)
	if strings.Contains(out, "docs/writing-checks.md:") {
		t.Errorf("an ADR in a code block is an example, yet it was reported:\n%s", out)
	}
}

func Test0147_4(t *testing.T) {
	t.Parallel()
	out, code := adrAsIssue(t, map[string]string{
		"docs/architecture/adr/0001-use-sqlite.md": "https://github.com/may-journal/example/issues/41\n",
		"fitness/adr-0001/README.md":               "# 0001 — Small files\n\nSmall files read fast.\n\nhttps://github.com/may-journal/example/issues/5\n\n## Check\n\nFails a long Markdown file.\n",
		"fitness/adr-0001/check":                   verdictScript(passing),
		".github/ISSUE_TEMPLATE/adr.md":            "---\nname: ADR\nlabels: ADR\n---\n\n## Status\n\n## Context\n\n## Decision\n\n## Consequences\n",
		"docs/notes.md":                            "# Notes\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

// allowADRs is a config whose adrAsIssue.allow holds entries.
func allowADRs(entries string) string {
	return `{"adrAsIssue": {"allow": [` + entries + `]}}` + "\n"
}

func Test0147_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(heldCheck(), map[string]string{
		".fitnessrc.json": allowADRs(`{"path": "docs/architecture/adr/*.md", "reason": "the migration waits on repo access"}, ` +
			`{"path": "fitness/adr-*/README.md", "reason": "the checks move with them"}`),
		"docs/architecture/adr/0001-use-sqlite.md": adrFile,
		"docs/architecture/adr/0002-sync.md":       adrFile,
		"docs/decisions/adr/0003-theme.md":         adrFile,
	}))
	out, code := fitness(t, repo, nil, "adr-as-issue")
	sees(t, out, code, 1, inADRFolder("docs/decisions/adr/0003-theme.md"))
	for _, kept := range []string{"0001-use-sqlite.md", "0002-sync.md", "fitness/adr-0001/README.md"} {
		if strings.Contains(out, kept+":") {
			t.Errorf("%s is allowed, yet it was reported:\n%s", kept, out)
		}
	}
	out, code = fitness(t, repo, nil, "adr-checks")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0147_6(t *testing.T) {
	t.Parallel()
	out, code := adrAsIssue(t, map[string]string{
		".fitnessrc.json":                          allowADRs(`{"path": "docs/architecture/adr/*.md"}, {"path": "docs/old/adr/*.md", "reason": "moved"}`),
		"docs/architecture/adr/0001-use-sqlite.md": adrFile,
	})
	sees(t, out, code, 1,
		`adrAsIssue.allow path "docs/architecture/adr/*.md" needs a reason`,
		`adrAsIssue.allow path "docs/old/adr/*.md" matches no ADR file; remove it`,
		inADRFolder("docs/architecture/adr/0001-use-sqlite.md"),
	)
}
