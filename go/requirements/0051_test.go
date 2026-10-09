package requirements

import (
	"strings"
	"testing"
)

// prStructureSummary is a real one-line blockquote summary.
const prStructureSummary = "> Ship the thing so a reader gets the value.\n\n"

// prStructureBackground is a filled `## Background` section.
const prStructureBackground = "## Background\n\nWhy this exists.\n\n"

// prStructureChangelog is a `## Changelog` section with one bullet.
const prStructureChangelog = "## Changelog\n\n- Feat: the new behavior\n"

// prStructure runs `fitness-install -- pr-structure --body-file pr.md` on
// happyRepo with body as the PR description.
func prStructure(t *testing.T, body string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"pr.md": body})
	return fitness(t, repo, nil, "pr-structure", "--body-file", "pr.md")
}

func Test0051_1(t *testing.T) {
	out, code := prStructure(t, prStructureSummary+prStructureBackground+prStructureChangelog)
	sees(t, out, code, 0, "pr-structure")
}

func Test0051_2(t *testing.T) {
	out, code := prStructure(t, prStructureBackground+prStructureChangelog)
	sees(t, out, code, 1, "Missing pitch: add a one-line blockquote (> …) before the first ## section")
}

func Test0051_3(t *testing.T) {
	out, code := prStructure(t, "> REPLACE-ME\n\n"+prStructureBackground+prStructureChangelog)
	sees(t, out, code, 1, "Pitch is still the template placeholder; write the real one-line value")
}

func Test0051_4(t *testing.T) {
	out, code := prStructure(t, prStructureSummary+prStructureChangelog)
	sees(t, out, code, 1, "Missing `## Background` section")
}

func Test0051_5(t *testing.T) {
	out, code := prStructure(t, prStructureSummary+prStructureBackground+"## Changelog\n\nNothing yet.\n")
	sees(t, out, code, 1, "`## Changelog` has no list item (- …)")
}

func Test0051_6(t *testing.T) {
	out, code := prStructure(t, prStructureSummary+prStructureBackground+prStructureChangelog+"\n## Testing\n\nRan it.\n")
	sees(t, out, code, 1, "Unexpected `## Testing` section — a PR has only Background and Changelog")
}

func Test0051_7(t *testing.T) {
	fence := strings.Repeat("`", 3)
	sample := "\n" + fence + "markdown\n## Testing\n" + fence + "\n"
	out, code := prStructure(t, prStructureSummary+prStructureBackground+prStructureChangelog+sample)
	sees(t, out, code, 0, "pr-structure")
}

func Test0051_8(t *testing.T) {
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "pr-structure")
	sees(t, out, code, 0, "pr-structure")
	passedWithNoFiles(t, out, "pr-structure")
}
