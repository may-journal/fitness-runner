package requirements

import (
	"strings"
	"testing"
)

// judgePlan runs plan-check on Plan n with body, as an issue edit triggers it.
func judgePlan(t *testing.T, n int, body string) (string, int) {
	t.Helper()
	return action(t, "plan-check", "issues", issueEvent(n, body))
}

// brokenPlan has no Background section.
const brokenPlan = "> One clear pitch for this Plan.\n\n## What needs to happen\n\n- [ ] Write the tests.\n"

func Test0013_1(t *testing.T) {
	body := planBody("- [ ] Write the tests.")
	n := newIssue(t, body, "Plan")
	out, code := judgePlan(t, n, body)
	sees(t, out, code, 0, "Passed.")
	issue := issueNow(t, n)
	if len(issue.Comments) != 1 || !strings.HasPrefix(issue.Comments[0].Body, "✅ Plan looks good.") || !hasLabel(issue.Labels, "fitness-valid") {
		t.Errorf("want one passing verdict and fitness-valid, got %+v", issue)
	}
}

func Test0013_2(t *testing.T) {
	n := newIssue(t, brokenPlan, "Plan")
	out, code := judgePlan(t, n, brokenPlan)
	sees(t, out, code, 1, "Background")
	issue := issueNow(t, n)
	if len(issue.Comments) != 1 || !strings.Contains(issue.Comments[0].Body, "❌ Plan needs work:\n- ") || !hasLabel(issue.Labels, "fitness-invalid") {
		t.Errorf("want one failing verdict listing errors and fitness-invalid, got %+v", issue)
	}
}

func Test0013_3(t *testing.T) {
	body := planBody("- [ ] Write the tests.")
	n := newIssue(t, body, "Plan")
	judgePlan(t, n, body)
	out, code := judgePlan(t, n, body)
	sees(t, out, code, 0, "Passed.")
	if got := len(issueNow(t, n).Comments); got != 1 {
		t.Errorf("comments = %d, want 1", got)
	}
}

func Test0013_4(t *testing.T) {
	n := newIssue(t, brokenPlan, "Plan")
	judgePlan(t, n, brokenPlan)
	fixed := planBody("- [ ] Write the tests.")
	mustDo(t, setIssue(n, "body="+fixed))
	out, code := judgePlan(t, n, fixed)
	sees(t, out, code, 0, "Passed.")
	c := issueNow(t, n).Comments
	if len(c) != 2 || !c[0].Hidden || c[1].Hidden || !strings.HasPrefix(c[1].Body, "✅") {
		t.Errorf("want the failed verdict hidden under a pass, got %+v", c)
	}
}

func Test0013_5(t *testing.T) {
	body := planBody("- [ ] Write the tests.")
	n := newIssue(t, body, "Plan")
	judgePlan(t, n, body)
	edited := planBody("- [ ] Write the tests.\n- [ ] Write the doc.")
	mustDo(t, setIssue(n, "body="+edited))
	out, code := judgePlan(t, n, edited)
	sees(t, out, code, 0, "Passed.")
	c := issueNow(t, n).Comments
	if len(c) != 1 || !strings.HasPrefix(c[0].Body, "✅ Validated (updated ") {
		t.Errorf("want the passing verdict updated in place, got %+v", c)
	}
}

func Test0013_6(t *testing.T) {
	n := newIssue(t, planBody("- [ ] Write the tests."), "Plan")
	listed(t, "issues?labels=Plan&state=open", n)
	out, _ := action(t, "plan-check", "workflow_dispatch", map[string]any{})
	sees(t, out, 0, 0, "plan-check #"+itoa(n))
	if !hasLabel(issueNow(t, n).Labels, "fitness-valid") {
		t.Errorf("open Plan #%d was not judged", n)
	}
}
