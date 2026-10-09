package requirements

import (
	"strings"
	"testing"
)

// closeIssue closes sandbox issue n for reason and returns the `issues:
// closed` event GitHub sends, sent by the test's own user.
func closeIssue(t *testing.T, n int, body, reason string) map[string]any {
	t.Helper()
	mustDo(t, setIssue(n, "state=closed", "state_reason="+reason))
	closedAt := gh(t, "api", "repos/"+sandbox+"/issues/"+itoa(n), "-q", ".closed_at")
	return map[string]any{
		"action": "closed",
		"issue":  map[string]any{"number": n, "body": body, "state_reason": reason, "closed_at": closedAt},
		"sender": map[string]any{"login": gh(t, "api", "user", "-q", ".login")},
	}
}

// unfinished is an issue body with one unchecked item.
const unfinished = "- [x] Write the doc.\n- [ ] Write the tests.\n"

func Test0015_1(t *testing.T) {
	n := newIssue(t, unfinished)
	ev := closeIssue(t, n, unfinished, "completed")
	out, code := action(t, "close-check", "issues", ev)
	sees(t, out, code, 0, `unchecked item: "Write the tests."`)
	issue := issueNow(t, n)
	login := ev["sender"].(map[string]any)["login"].(string)
	if issue.State != "OPEN" || len(issue.Comments) != 1 || !strings.Contains(issue.Comments[0].Body, `"Write the tests."`) || !strings.Contains(issue.Comments[0].Body, "@"+login) {
		t.Errorf("want the issue reopened with one comment naming the item and @%s, got %+v", login, issue)
	}
}

func Test0015_2(t *testing.T) {
	n := newIssue(t, unfinished)
	ev := closeIssue(t, n, unfinished, "completed")
	action(t, "close-check", "issues", ev)
	out, code := action(t, "close-check", "issues", ev)
	sees(t, out, code, 0)
	if got := len(issueNow(t, n).Comments); got != 1 {
		t.Errorf("comments = %d, want 1", got)
	}
}

func Test0015_3(t *testing.T) {
	n := newIssue(t, unfinished)
	out, code := action(t, "close-check", "issues", closeIssue(t, n, unfinished, "not_planned"))
	sees(t, out, code, 0, "closed as not_planned")
	if issue := issueNow(t, n); issue.State != "CLOSED" || len(issue.Comments) != 0 {
		t.Errorf("want it left closed and quiet, got %+v", issue)
	}
}

func Test0015_4(t *testing.T) {
	done := "- [x] Write the doc.\n- [x] Write the tests.\n"
	n := newIssue(t, done)
	out, code := action(t, "close-check", "issues", closeIssue(t, n, done, "completed"))
	sees(t, out, code, 0, "Passed.")
	if issue := issueNow(t, n); issue.State != "CLOSED" || len(issue.Comments) != 0 {
		t.Errorf("want it left closed and quiet, got %+v", issue)
	}
}
