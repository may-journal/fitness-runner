package requirements

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func checkoutMain(t *testing.T, repo string) {
	t.Helper()
	git(t, repo, "checkout", "-b", "main", "origin/main")
}

func pushedAnswer(t *testing.T, repo, branch, title, body string, number int) response {
	t.Helper()
	pr := branchAnswer(t, repo, title, body)
	var data map[string]any
	mustDo(t, json.Unmarshal([]byte(pr.Stdout), &data))
	data["head"], data["number"] = branch, number
	raw, err := json.Marshal(data)
	mustDo(t, err)
	pr.Stdout = string(raw)
	pr.Match = []string{"head=" + url.QueryEscape("may-journal:"+branch)}
	return pr
}

// pushedGitHub supplies approval and issue reads alongside the actual pushed
// PRs. Plan #11 has body plan in both the gate's read and the issue read.
func pushedGitHub(t *testing.T, plan string, prs ...response) []response {
	t.Helper()
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repo0014 + "\n"}}
	saved = append(saved, prs...)
	return append(saved, planView(t, 11, plan, []string{"Plan"}, "I approve"), issue0014(t, 11, plan, "Plan"))
}

func Test0068_1(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "added one file")
	checkoutMain(t, repo)
	pr := pushedAnswer(t, repo, "work", "added one file", pr0014(11), 1)
	empty := response{Match: []string{"head=" + url.QueryEscape("may-journal:main")}}
	saved := pushedGitHub(t, planBody("- [x] Write the tests."), pr, empty)
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work")
	sees(t, out, code, 1, `"added one file"`)
	branchReadOnce(t, rp)
	if rp.called("gh", empty.Match...) {
		t.Fatalf("checked the checkout instead of the pushed branch: %q", rp.calls("gh"))
	}
}

func Test0068_2(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "test(app): add one file")
	checkoutMain(t, repo)
	valid := pushedAnswer(t, repo, "work", "test(app): add one file", pr0014(11), 1)
	invalid := pushedAnswer(t, repo, "main", "changed checkout file", pr0014(11), 2)
	saved := pushedGitHub(t, planBody("- [x] Write the tests."), valid, invalid)
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work")
	sees(t, out, code, 0, "work -> work")
	branchReadOnce(t, rp)
}

func Test0068_3(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "added one file")
	checkoutMain(t, repo)
	git(t, repo, "branch", "other")
	invalid := pushedAnswer(t, repo, "work", "added one file", pr0014(11), 1)
	valid := pushedAnswer(t, repo, "other", "test(app): add another file", pr0014(11), 2)
	saved := pushedGitHub(t, planBody("- [x] Write the tests."), invalid, valid)
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work", "other")
	sees(t, out, code, 1, "pr-check #1", "pr-check #2", `"added one file"`)
	if strings.Count(out, "→ local-branch") != 1 || len(rp.calls("gh")) == 0 {
		t.Fatalf("want one meta-check invocation: %s", out)
	}
	for _, pr := range []response{invalid, valid} {
		if count := strings.Count(strings.Join(rp.calls("gh"), "\n"), pr.Match[0]); count != 1 {
			t.Errorf("want one discovery for %s, got %d", pr.Match[0], count)
		}
	}
}

func Test0068_6(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "test(app): add one file")
	checkoutMain(t, repo)
	git(t, repo, "branch", "other")
	work := pushedAnswer(t, repo, "work", "test(app): add one file", pr0014(11), 1)
	other := pushedAnswer(t, repo, "other", "test(app): add another file", pr0014(11), 2)
	rp := standIn(t, map[string][]response{"gh": pushedGitHub(t, planBody("- [x] Write the tests."), work, other)})
	out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work", "other")
	sees(t, out, code, 0, "work -> work", "other -> other")
	planReadOnce(t, rp, "11")
}

func Test0068_4(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "test(app): add one file")
	out, code := user(t, repo, env, "push", "origin", "work")
	sees(t, out, code, 0, "work -> work")
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code = user(t, repo, pushEnv(env, rp), "push", "origin", "--delete", "work")
	sees(t, out, code, 0, "deleted")
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("deletion queried metadata: %q", rp.calls("gh"))
	}
}

func Test0068_5(t *testing.T) {
	t.Parallel()
	repo, env, _ := metadataPushRepo(t, "test(app): add one file")
	checkoutMain(t, repo)
	pr := pushedAnswer(t, repo, "renamed", "test(app): add one file", pr0014(12), 1)
	saved := append(pushedGitHub(t, brokenPlan, pr), issue0014(t, 12, "An ordinary issue."))
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work:renamed")
	sees(t, out, code, 1, "pr-check #1", "plan-check #11", "Background")
	branchReadOnce(t, rp)
}
