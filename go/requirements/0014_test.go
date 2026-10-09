package requirements

import (
	"fmt"
	"strconv"
	"testing"
)

// judgePR runs pr-check on a pull_request event with title and body.
func judgePR(t *testing.T, title, body string) (string, int) {
	t.Helper()
	payload := map[string]any{"action": "edited", "pull_request": map[string]any{"number": 1, "title": title, "body": body}}
	return action(t, "pr-check", "pull_request", payload)
}

func itoa(n int) string { return strconv.Itoa(n) }

func Test0014_1(t *testing.T) {
	plan := newIssue(t, planBody("- [x] Write the tests."), "Plan")
	out, code := judgePR(t, "test(app): add one file", prBody(plan))
	sees(t, out, code, 0, "Passed.")
}

func Test0014_2(t *testing.T) {
	plan := newIssue(t, planBody("- [x] Write the tests."), "Plan")
	out, code := judgePR(t, "added one file", prBody(plan))
	sees(t, out, code, 1, `"added one file"`)
}

func Test0014_3(t *testing.T) {
	plan := newIssue(t, planBody("- [ ] Write the tests."), "Plan")
	out, code := judgePR(t, "test(app): add one file", prBody(plan))
	sees(t, out, code, 1, itoa(plan)+`, which has an unchecked item: "Write the tests."`)
}

func Test0014_4(t *testing.T) {
	bug := newIssue(t, "A bug the Plan fixes.")
	plan := newIssue(t, planBody("- [x] Fix the bug.")+fmt.Sprintf("\nCloses #%d\n", bug), "Plan")
	out, code := judgePR(t, "test(app): add one file", prBody(plan))
	sees(t, out, code, 1, "#"+itoa(bug))
}

func Test0014_5(t *testing.T) {
	plan := newIssue(t, planBody("- [x] Write the tests."), "Plan")
	pr := newPR(t, "test(app): add one file", prBody(plan))
	listed(t, "pulls?state=open", pr)
	out, _ := action(t, "pr-check", "workflow_dispatch", map[string]any{})
	sees(t, out, 0, 0, "pr-check #"+itoa(pr))
}
