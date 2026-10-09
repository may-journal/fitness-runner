package requirements

import "testing"

// issueChecklist runs `fitness-install -- issue-checklist` on an issue body.
func issueChecklist(t *testing.T, body string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"issue.md": body})
	return fitness(t, repo, nil, "issue-checklist", "--body-file", "issue.md")
}

func Test0012_1(t *testing.T) {
	t.Parallel()
	out, code := issueChecklist(t, "- [x] Write the doc.\n- [ ] Write the tests.\n")
	sees(t, out, code, 1, `unchecked item: "Write the tests."`)
}

func Test0012_2(t *testing.T) {
	t.Parallel()
	out, code := issueChecklist(t, "- [x] Write the doc.\n- [x] Write the tests.\n")
	sees(t, out, code, 0, "issue-checklist")
}
