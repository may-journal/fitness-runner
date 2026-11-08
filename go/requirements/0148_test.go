package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adrIssueBody is an ADR issue body that follows the template.
const adrIssueBody = "> The app stores notes in SQLite.\n\n## Background\n\nWe sync offline.\n\n" +
	"## Decision\n\nStore notes in SQLite.\n\n## Currently Known Consequences\n\nOne file holds every note.\n"

// judgeADR runs issue-check on ADR issue 21 titled title with body, in
// happyRepo with files written over it.
func judgeADR(t *testing.T, title, body string, files map[string]string) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 21, "title": title, "body": body}
	return ghRun(t, "issue-check", "issues", bodyEdit(issue), files, kindGitHub("adr", "ADR"))
}

func Test0148_1(t *testing.T) {
	t.Parallel()
	template, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", "adr.md"))
	mustDo(t, err)
	for name, files := range map[string]map[string]string{
		"the built-in shape": nil,
		"the repo template":  {".github/ISSUE_TEMPLATE/adr.md": string(template)},
	} {
		out, code, rp := judgeADR(t, "[ADR] Store notes in SQLite", adrIssueBody, files)
		sees(t, out, code, 0, "Passed.")
		if !commentedOn(rp, kindIssue, "body=✅ ADR looks good.") {
			t.Errorf("want a passing ADR verdict by %s, got gh calls %q", name, rp.calls("gh"))
		}
	}
}

func Test0148_2(t *testing.T) {
	t.Parallel()
	template, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", "adr.md"))
	mustDo(t, err)
	files := map[string]string{".github/ISSUE_TEMPLATE/adr.md": string(template)}
	body := strings.Replace(adrIssueBody, "## Currently Known Consequences\n\nOne file holds every note.\n", "## Status\n\nAccepted\n", 1)
	out, code, rp := judgeADR(t, "Store notes in SQLite", body, files)
	sees(t, out, code, 1, "the title must read", "Unexpected", "Status", "Missing", "Currently Known Consequences")
	if !labeledOn(rp, kindIssue, "fitness-invalid") {
		t.Errorf("want fitness-invalid, got gh calls %q", rp.calls("gh"))
	}
}
