package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adrIssueBody is an ADR issue body that follows the template, with status.
func adrIssueBody(status string) string {
	return "> The app stores notes in SQLite.\n\n## Status\n\n" + status +
		"\n\n## Context\n\nWe sync offline.\n\n## Decision\n\nStore notes in SQLite.\n\n## Consequences\n\nOne file holds every note.\n"
}

// judgeADR runs issue-check on ADR issue 21 titled title with body, in
// happyRepo with files written over it.
func judgeADR(t *testing.T, title, body string, files map[string]string) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 21, "title": title, "body": body}
	return ghRun(t, "issue-check", "issues", bodyEdit(issue), files, kindGitHub("adr", "ADR"))
}

func Test0148_1(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"Accepted", "Superseded by https://github.com/may-journal/example/issues/42"} {
		out, code, rp := judgeADR(t, "[ADR] Store notes in SQLite", adrIssueBody(status), nil)
		sees(t, out, code, 0, "Passed.")
		if !commentedOn(rp, kindIssue, "body=✅ ADR looks good.") {
			t.Errorf("want a passing ADR verdict for Status %q, got gh calls %q", status, rp.calls("gh"))
		}
	}
}

func Test0148_2(t *testing.T) {
	t.Parallel()
	template, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", "adr.md"))
	mustDo(t, err)
	files := map[string]string{".github/ISSUE_TEMPLATE/adr.md": string(template)}
	body := strings.Replace(adrIssueBody("Accepted after a long talk"), "## Consequences\n\nOne file holds every note.\n", "", 1)
	out, code, rp := judgeADR(t, "Store notes in SQLite", body, files)
	sees(t, out, code, 1, "the title must read", "must be one word: Proposed, Accepted or Superseded", "Consequences")
	if !labeledOn(rp, kindIssue, "fitness-invalid") {
		t.Errorf("want fitness-invalid, got gh calls %q", rp.calls("gh"))
	}
}
