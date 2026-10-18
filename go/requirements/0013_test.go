package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	return ghRun(t, "plan-check", event, payload, nil, saved)
}

// ghRun runs a GitHub command the way its Actions step does, in happyRepo
// with files written over it, on the event payload GitHub would deliver,
// with gh answering from saved responses.
func ghRun(t *testing.T, command, event string, payload map[string]any, files map[string]string, saved []response) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	rp := standIn(t, map[string][]response{"gh": saved})
	env := append(rp.env, "GITHUB_EVENT_NAME="+event, "GITHUB_EVENT_PATH="+path, "GITHUB_REPOSITORY=may-journal/example")
	out, code := fitness(t, repo, env, command)
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
	return issueGitHub(planIssue, "fitness:plan-structure:", verdicts, append([]string{"Plan"}, labels...)...)
}

// issueGitHub is the saved GitHub that holds issue n's verdict comments,
// found by a query naming marker, as the verdicts query prints them, and
// its labels. Every write succeeds.
func issueGitHub(n, marker, verdicts string, labels ...string) []response {
	issue, _ := json.Marshal(map[string]any{"body": "", "labels": labels})
	return []response{
		{Match: []string{marker}, Stdout: verdicts},
		{Match: []string{"minimizeComment"}},
		{Match: []string{"labels: [.labels[].name]", "issues/" + n}, Stdout: string(issue) + "\n"},
		{Match: []string{"-X POST"}},
		{Match: []string{"-X PATCH", "issues/comments/"}},
		{Match: []string{"-X DELETE", "issues/" + n + "/labels/"}},
		{Match: []string{"--silent", "/labels/fitness"}},
	}
}

// planVerdict is one verdict comment the fitness bot wrote, as the verdicts
// query prints it: id 11, starting with text and marked as judging body.
func planVerdict(text, body string) string {
	return verdictBy("github-actions", "Bot", 11, text, body)
}

// verdictBy is planVerdict with comment id, written by login, an account of
// kind, as GraphQL names them: an app's login has no [bot] suffix.
func verdictBy(login, kind string, id int, text, body string) string {
	sum := sha256.Sum256([]byte(body))
	mark := "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
	line, _ := json.Marshal(map[string]any{"id": id, "nodeId": fmt.Sprintf("IC_%d", id), "body": text + "\n\n<sub>Checked by fitness-runner</sub>\n\n" + mark, "isHidden": false,
		"author": map[string]any{"login": login, "__typename": kind}})
	return string(line) + "\n"
}

// commented reports whether plan-check posted a comment on the Plan
// holding every want.
func commented(rp replays, want ...string) bool {
	return commentedOn(rp, planIssue, want...)
}

// commentedOn reports whether a GitHub command posted a comment on issue n
// holding every want.
func commentedOn(rp replays, n string, want ...string) bool {
	return rp.called("gh", append([]string{"-X POST", "body=", "issues/" + n + "/comments"}, want...)...)
}

// labeled reports whether plan-check added label to the Plan.
func labeled(rp replays, label string) bool {
	return labeledOn(rp, planIssue, label)
}

// labeledOn reports whether a GitHub command added label to issue n.
func labeledOn(rp replays, n, label string) bool {
	return rp.called("gh", "-X POST", "labels[]="+label, "issues/"+n+"/labels")
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
	own := planVerdict("✅ Plan looks good.", body)
	pasted := verdictBy("dev", "User", 12, "✅ Plan looks good.", body)
	for _, c := range []struct {
		verdicts string
		posts    bool
	}{{own, false}, {pasted, true}, {own + pasted, false}} {
		out, code, rp := judgePlan(t, body, planGitHub(c.verdicts, "fitness", "fitness-valid"))
		sees(t, out, code, 0, "Passed.")
		if commented(rp, "body=✅ Plan looks good.") != c.posts || touched(rp, "12") {
			t.Errorf("want a new verdict %v and the pasted one untouched, got gh calls %q", c.posts, rp.calls("gh"))
		}
	}
}

// touched reports whether plan-check edited or hid comment id.
func touched(rp replays, id string) bool {
	return rp.called("gh", "IC_"+id) || rp.called("gh", "issues/comments/"+id)
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

// readOnce fails unless branch-check fetched issue n exactly once.
func readOnce(t *testing.T, rp replays, n string) {
	t.Helper()
	count := 0
	for _, call := range rp.calls("gh") {
		if strings.Contains(call, "issues/"+n) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("linked Plan #%s fetched %d times: %q", n, count, rp.calls("gh"))
	}
}

func Test0013_9(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	commit(t, repo, nil, "feat(app): add one file\n\nPlan #11")
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, brokenPlan, "Plan"))})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 1, "plan-check #11", "Background")
	readOnce(t, rp, "11")

	fork, forkPR := forkBranch(t, "release")
	git(t, fork, "update-ref", "-d", "refs/remotes/origin/main")
	rp = standIn(t, map[string][]response{"gh": branchGitHub(forkPR, issue0014(t, 11, "An ordinary issue."), issue0014(t, 12, brokenPlan, "Plan"))})
	out, code = fitness(t, fork, rp.env, "branch-check")
	sees(t, out, code, 1, "plan-check #12", "Background")
	readOnce(t, rp, "12")
}

func Test0013_10(t *testing.T) {
	t.Parallel()
	gone := response{Match: []string{"fitness:plan-structure:"}, Stderr: "gh: Could not resolve to an Issue with the number of " + planIssue + ".\n", Exit: 1}
	out, code, rp := judgePlan(t, planBody("- [ ] Write the tests."), append([]response{gone}, planGitHub("")...))
	sees(t, out, code, 0, "#"+planIssue+" no longer exists")
	if rp.called("gh", "-X POST") {
		t.Errorf("want nothing written to a moved Plan, got gh calls %q", rp.calls("gh"))
	}
}
