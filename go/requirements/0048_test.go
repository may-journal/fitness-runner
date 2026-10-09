package requirements

import "testing"

// planStructure runs `fitness-install -- plan-structure --body-file plan.md`
// on happyRepo with body as plan.md.
func planStructure(t *testing.T, body string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"plan.md": body})
	return fitness(t, repo, nil, "plan-structure", "--body-file", "plan.md")
}

// planStructureOf is a plan with pitch, then Background, then the steps.
func planStructureOf(pitch, steps string) string {
	return pitch + "\n\n## Background\n\nThe sync job drops edits made offline.\n\n## What needs to happen\n\n" + steps + "\n"
}

const planStructurePitch = "> Offline edits survive the next sync."

func Test0048_1(t *testing.T) {
	out, code := planStructure(t, planBody("- [ ] Write the tests."))
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0048_2(t *testing.T) {
	out, code := planStructure(t, planStructureOf("Offline edits survive the next sync.", "- [ ] Keep queued edits."))
	sees(t, out, code, 1, "Missing pitch: add a one-line blockquote (> …) before the first ## section")
}

func Test0048_3(t *testing.T) {
	out, code := planStructure(t, planStructureOf("> REPLACE-ME", "- [ ] Keep queued edits."))
	sees(t, out, code, 1, "Pitch is still the template placeholder; write the real one-line value")
}

func Test0048_4(t *testing.T) {
	out, code := planStructure(t, planStructurePitch+"\n\n## What needs to happen\n\n- [ ] Keep queued edits.\n")
	sees(t, out, code, 1, "Missing `## Background` section")
}

func Test0048_5(t *testing.T) {
	out, code := planStructure(t, planStructureOf(planStructurePitch, "```\n- [ ] Keep queued edits.\n```"))
	sees(t, out, code, 1, "`## What needs to happen` has no checklist item (- [ ] …)")
}

func Test0048_6(t *testing.T) {
	out, code := planStructure(t, planStructureOf(planStructurePitch, "- [ ] Keep queued edits.\n\n## Notes\n\nSee the sync design."))
	sees(t, out, code, 1, "Unexpected `## Notes` section — a plan has only Background and What needs to happen")
}

func Test0048_7(t *testing.T) {
	out, code := planStructure(t, planStructureOf(planStructurePitch, "- [ ] Keep queued edits.\n\n## Open questions\n\n- Which sync wins?"))
	sees(t, out, code, 1, "Remove the `## Open questions` section — turn unknowns into checklist steps")
}

func Test0048_8(t *testing.T) {
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "plan-structure")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "plan-structure")
}
