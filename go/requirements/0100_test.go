package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// taught asserts the skill folder's SKILL.md quotes every line, so a reader
// of the skill sees the failure the check prints.
func taught(t *testing.T, skill string, quotes ...string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "skills", skill, "SKILL.md"))
	mustDo(t, err)
	for _, quote := range quotes {
		if !strings.Contains(string(raw), quote) {
			t.Errorf("skill %s does not quote %q", skill, quote)
		}
	}
}

func Test0100_1(t *testing.T) {
	t.Parallel()
	const quote = "Missing pitch: add a one-line blockquote (> …) before the first ## section"
	taught(t, "plan-structure", quote)
	out, code := planStructure(t, planStructureOf("Offline edits survive the next sync.", "- [ ] Keep queued edits."))
	sees(t, out, code, 1, quote)
	out, code = planStructure(t, planStructureOf(planStructurePitch, "- [ ] Keep queued edits."))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0100_2(t *testing.T) {
	t.Parallel()
	quotes := []string{
		"item 1 uses a checkbox; list the issue as a plain reference",
		"item 2 has an emoji; GitHub shows each issue's state, so drop it",
		"item 3 (#24) is listed before #25, which it runs after",
	}
	taught(t, "epic-structure", quotes...)
	repo := example(t, "happyRepo", map[string]string{
		"bad.md":  epicBody("1. [ ] #22\n2. #23 ✅\n3. #24, after #25\n4. #25"),
		"good.md": epicBody("1. #22\n2. #23, after #22"),
	})
	out, code := fitness(t, repo, nil, "epic-structure", "--body-file", "bad.md")
	sees(t, out, code, 1, quotes...)
	out, code = fitness(t, repo, nil, "epic-structure", "--body-file", "good.md")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0100_3(t *testing.T) {
	t.Parallel()
	const quote = "Missing `## Background` section"
	taught(t, "pr-structure", quote)
	out, code := prStructure(t, prStructureSummary+prStructureChangelog)
	sees(t, out, code, 1, quote)
	out, code = prStructure(t, prStructureSummary+prStructureBackground+prStructureChangelog)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0100_4(t *testing.T) {
	t.Parallel()
	taught(t, "pr-closes-issue", prClosesNoKeyword)
	out, code := prCloses(t, "This addresses #12.\n")
	sees(t, out, code, 1, prClosesNoKeyword)
	out, code = prCloses(t, "Closes #12.\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0100_5(t *testing.T) {
	t.Parallel()
	const quote = `unchecked item: "Write the tests."`
	taught(t, "issue-checklist", quote)
	out, code := issueChecklist(t, "- [x] Write the doc.\n- [ ] Write the tests.\n")
	sees(t, out, code, 1, quote)
	out, code = issueChecklist(t, "- [x] Write the doc.\n- [x] Write the tests.\n")
	sees(t, out, code, 0, "issue-checklist")
}
