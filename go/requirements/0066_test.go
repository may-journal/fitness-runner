package requirements

import (
	"encoding/json"
	"net/url"
	"os/exec"
	"strings"
	"testing"
)

// branchRepo supplies a local branch and fetched base without network writes.
func branchRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", files)
	git(t, repo, "branch", "-M", "work")
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repo0014+".git")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	return repo
}

// branchAnswer is the saved result of the filtered open-PR request.
func branchAnswer(t *testing.T, repo, title, body string) response {
	t.Helper()
	base, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	mustDo(t, err)
	data, err := json.Marshal(map[string]any{
		"number": 1, "title": title, "body": body, "head": "work", "source": repo0014,
		"base": "main", "baseSHA": strings.TrimSpace(string(base)), "url": "https://github.com/" + repo0014 + "/pull/1", "draft": true,
	})
	mustDo(t, err)
	return response{Match: []string{"pulls?state=open&head=" + url.QueryEscape("may-journal:work")}, Stdout: string(data) + "\n"}
}

func branchGitHub(pr response, issues ...response) []response {
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repo0014 + "\n"}, pr}
	return append(saved, issues...)
}

func Test0066_1(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	pr := branchAnswer(t, repo, "added one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))})
	out, code := fitness(t, repo, rp.env)
	sees(t, out, code, 1, "README.md: disallowed **bold**", `"added one file"`, "pr-check #1")
}

func Test0066_2(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"gh": branchGitHub(response{Match: []string{"pulls?state=open"}})})
	out, code := fitness(t, branchRepo(t, nil), rp.env, "branch-check")
	sees(t, out, code, 0, "no open PR", "skipped")
}

func Test0066_3(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	git(t, repo, "checkout", "--detach", "HEAD")
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 0, "detached HEAD", "skipped")
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("detached run queried GitHub: %q", rp.calls("gh"))
	}
}

func Test0066_4(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	pr.Stdout += pr.Stdout
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 1, "multiple open PRs")
}

func Test0066_5(t *testing.T) {
	t.Parallel()
	pr := response{Match: []string{"pulls?state=open"}, Exit: 1, Stderr: "HTTP 401: authentication required"}
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
	out, code := fitness(t, branchRepo(t, nil), rp.env, "branch-check")
	sees(t, out, code, 1, "HTTP 401", "retry fitness branch-check")
}

func Test0014_6(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	issue := response{Match: []string{"issues/11"}, Exit: 1, Stderr: "HTTP 404: issue inaccessible"}
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue)})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 1, "cannot validate linked issue #11", "retry")
	actions, actionsCode := judge0014(t, "test(app): add one file", pr0014(11), issue)
	sees(t, actions, actionsCode, 1, `cannot validate linked issue \#11`)
}

func Test0066_6(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	body := strings.ReplaceAll(pr0014(11), "Closes #11", "")
	rp := standIn(t, map[string][]response{"gh": branchGitHub(branchAnswer(t, repo, "test(app): add one file", body))})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 1, "no closing issue links", "closing")
}

func Test0066_7(t *testing.T) {
	t.Parallel()
	repo, pr := forkBranch(t, "main")
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, "An ordinary issue.", "bug"), issue0014(t, 12, planBody("- [x] Write the tests."), "Plan"))})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 0, "pr-check #1", "plan-check #12", "Issue: https://github.com/"+repo0014+"/issues/11")
	if rp.called("gh", "issues/999") {
		t.Fatalf("validated an upstream Plan outside the PR: %q", rp.calls("gh"))
	}
}

func Test0066_8(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code := fitness(t, branchRepo(t, nil), rp.env, "markdown-no-bold-italic")
	sees(t, out, code, 0, "passed")
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("single check queried GitHub: %q", rp.calls("gh"))
	}
}

// forkBranch has a stale fork main, an upstream Plan, and a branch-only Plan.
func forkBranch(t *testing.T, base string) (string, response) {
	t.Helper()
	repo := branchRepo(t, nil)
	git(t, repo, "remote", "set-url", "origin", "https://github.com/fork-owner/example.git")
	git(t, repo, "remote", "add", "upstream", "https://github.com/"+repo0014+".git")
	commit(t, repo, nil, "feat(app): update upstream\n\nPlan #999")
	git(t, repo, "update-ref", "refs/remotes/upstream/"+base, "HEAD")
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	pr.Match = []string{"head=" + url.QueryEscape("fork-owner:work")}
	pr.Stdout = strings.ReplaceAll(pr.Stdout, `"source":"`+repo0014+`"`, `"source":"fork-owner/example"`)
	pr.Stdout = strings.ReplaceAll(pr.Stdout, `"base":"main"`, `"base":"`+base+`"`)
	commit(t, repo, nil, "feat(app): add one file\n\nPlan #12")
	return repo, pr
}
