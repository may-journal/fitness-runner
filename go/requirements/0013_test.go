package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// planIssue is the Plan number every 0013 event and saved response names.
const planIssue = "7"

// brokenPlan has no Background section.
const brokenPlan = "> One clear pitch for this Plan.\n\n## What needs to happen\n\n- [ ] Write the tests.\n"

// planCheck runs plan-check the way its Actions step does, on the event
// payload GitHub would deliver, with gh answering from saved responses.
func planCheck(t *testing.T, event string, payload map[string]any, saved []response) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	rp := standIn(t, map[string][]response{"gh": saved})
	env := append(rp.env, "GITHUB_EVENT_NAME="+event, "GITHUB_EVENT_PATH="+path, "GITHUB_REPOSITORY=may-journal/example")
	out, code := fitness(t, repo, env, "plan-check")
	return out, code, rp
}

// judgePlan runs plan-check on the Plan with body, as an issue edit triggers
// it.
func judgePlan(t *testing.T, body string, saved []response) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 7, "title": "t", "body": body}
	return planCheck(t, "issues", map[string]any{"action": "edited", "issue": issue}, saved)
}

// planGitHub is the saved GitHub that holds the Plan's verdict comments, as
// the verdicts query prints them, and its labels. Every write succeeds.
func planGitHub(verdicts string, labels ...string) []response {
	issue, _ := json.Marshal(map[string]any{"body": "", "labels": append([]string{"Plan"}, labels...)})
	return []response{
		{Match: []string{"fitness:plan-structure:"}, Stdout: verdicts},
		{Match: []string{"minimizeComment"}},
		{Match: []string{"labels: [.labels[].name]", "issues/" + planIssue}, Stdout: string(issue) + "\n"},
		{Match: []string{"-X POST"}},
		{Match: []string{"-X PATCH", "issues/comments/"}},
		{Match: []string{"-X DELETE", "issues/" + planIssue + "/labels/"}},
		{Match: []string{"--silent", "/labels/fitness"}},
	}
}

// planVerdict is one verdict comment as the verdicts query prints it: id 11,
// starting with text and marked as judging body.
func planVerdict(text, body string) string {
	sum := sha256.Sum256([]byte(body))
	mark := "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
	line, _ := json.Marshal(map[string]any{"id": 11, "nodeId": "IC_11", "body": text + "\n\n<sub>Checked by fitness-runner</sub>\n\n" + mark, "isHidden": false})
	return string(line) + "\n"
}

// commented reports whether plan-check posted a comment on the Plan
// holding every want.
func commented(rp replays, want ...string) bool {
	return rp.called("gh", append([]string{"-X POST", "body=", "issues/" + planIssue + "/comments"}, want...)...)
}

// labeled reports whether plan-check added label to the Plan.
func labeled(rp replays, label string) bool {
	return rp.called("gh", "-X POST", "labels[]="+label, "issues/"+planIssue+"/labels")
}

func Test0013_1(t *testing.T) {
	t.Parallel()
	out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), planGitHub(""))
	sees(t, out, code, 0, "Passed.")
	if !commented(rp, "body=✅ Plan looks good.") || !labeled(rp, "fitness-valid") {
		t.Errorf("want a passing verdict and fitness-valid, got gh calls %q", rp.calls("gh"))
	}
}

func Test0013_2(t *testing.T) {
	t.Parallel()
	out, code, rp := judgePlan(t, brokenPlan, planGitHub(""))
	sees(t, out, code, 1, "Background")
	if !commented(rp, `body=❌ Plan needs work:\n- `) || !labeled(rp, "fitness-invalid") {
		t.Errorf("want a failing verdict listing errors and fitness-invalid, got gh calls %q", rp.calls("gh"))
	}
}

func Test0013_3(t *testing.T) {
	t.Parallel()
	body := planBody("- [ ] Write the tests.")
	out, code, rp := judgePlan(t, body, planGitHub(planVerdict("✅ Plan looks good.", body), "fitness", "fitness-valid"))
	sees(t, out, code, 0, "Passed.")
	if rp.called("gh", "body=") {
		t.Errorf("want no comment written, got gh calls %q", rp.calls("gh"))
	}
}

func Test0013_4(t *testing.T) {
	t.Parallel()
	failed := planVerdict("❌ Plan needs work:\n- Missing section: Background", brokenPlan)
	out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), planGitHub(failed, "fitness", "fitness-invalid"))
	sees(t, out, code, 0, "Passed.")
	if !commented(rp, "body=✅ Plan looks good.") || !rp.called("gh", "minimizeComment", "id=IC_11") {
		t.Errorf("want a passing verdict and the failed one hidden, got gh calls %q", rp.calls("gh"))
	}
}

func Test0013_5(t *testing.T) {
	t.Parallel()
	passed := planVerdict("✅ Plan looks good.", planBody("- [ ] Write the tests."))
	edited := planBody("- [ ] Write the tests.\n- [ ] Write the doc.")
	out, code, rp := judgePlan(t, edited, planGitHub(passed, "fitness", "fitness-valid"))
	sees(t, out, code, 0, "Passed.")
	edit := rp.called("gh", "-X PATCH", "body=✅ Validated (updated ", "issues/comments/11")
	if !edit || commented(rp) {
		t.Errorf("want the passing verdict updated in place, got gh calls %q", rp.calls("gh"))
	}
}

func Test0013_6(t *testing.T) {
	t.Parallel()
	plan, _ := json.Marshal(map[string]any{"number": 7, "body": planBody("- [ ] Write the tests.")})
	saved := append([]response{{Match: []string{"issues?labels=Plan&state=open"}, Stdout: string(plan) + "\n"}}, planGitHub("")...)
	out, code, rp := planCheck(t, "workflow_dispatch", map[string]any{}, saved)
	sees(t, out, code, 0, "plan-check #"+planIssue)
	if !labeled(rp, "fitness-valid") {
		t.Errorf("open Plan #%s was not judged, got gh calls %q", planIssue, rp.calls("gh"))
	}
}
