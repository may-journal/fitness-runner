package requirements

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// iloOrigin is the sandbox's URL, so a bare `#NN` belongs to the sandbox.
const iloOrigin = "https://github.com/" + sandbox + ".git"

// iloBranch is happyRepo on branch, off a main pushed to a local origin.
func iloBranch(t *testing.T, branch string) string {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	remote(t, repo)
	git(t, repo, "push", "-q", "origin", "HEAD:main")
	git(t, repo, "switch", "-q", "-c", branch)
	return repo
}

// iloOnSandbox is iloBranch with origin pointed at the sandbox on GitHub.
func iloOnSandbox(t *testing.T, branch string) string {
	t.Helper()
	repo := iloBranch(t, branch)
	git(t, repo, "remote", "set-url", "origin", iloOrigin)
	return repo
}

// iloCommit commits message in repo, authored at date when it is set.
func iloCommit(t *testing.T, repo, date, message string) {
	t.Helper()
	if out, code := user(t, repo, []string{"GIT_AUTHOR_DATE=" + date}, "commit", "-q", "--allow-empty", "-m", message); code != 0 {
		t.Fatalf("commit %q: %s", message, out)
	}
}

// iloCheck runs issue-link-once on the proposed commit message.
func iloCheck(t *testing.T, repo, message string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, "issue-link-once", "--message", message)
}

// iloHead returns repo's HEAD commit.
func iloHead(t *testing.T, repo string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	mustDo(t, err)
	return strings.TrimSpace(string(out))
}

// iloPullRequest runs issue-link-once as CI does on pull request #9, opened
// at noon with a body closing #12, whose head is repo's HEAD.
func iloPullRequest(t *testing.T, repo string) (string, int) {
	t.Helper()
	payload := map[string]any{"pull_request": map[string]any{
		"number": 9, "body": "Closes #12", "created_at": "2026-09-30T12:00:00Z",
		"head": map[string]any{"sha": iloHead(t, repo)},
	}}
	data, err := json.Marshal(payload)
	mustDo(t, err)
	path := filepath.Join(t.TempDir(), "event.json")
	mustDo(t, os.WriteFile(path, data, 0o644))
	env := []string{"GITHUB_BASE_REF=main", "GITHUB_EVENT_PATH=" + path}
	return fitness(t, repo, env, "issue-link-once")
}

func Test0031_1(t *testing.T) {
	t.Parallel()
	repo := iloOnSandbox(t, "topic")
	iloCommit(t, repo, "", "feat(app): start\n\nPlan #12")
	out, code := iloCheck(t, repo, "fix(app): more\n\nSee https://github.com/"+sandbox+"/issues/12")
	sees(t, out, code, 1, sandbox+`#12 is already linked by commit`, `("feat(app): start")`)
}

func Test0031_2(t *testing.T) {
	t.Parallel()
	repo := iloOnSandbox(t, "topic")
	iloCommit(t, repo, "", "feat(app): start\n\nPlan #12")
	out, code := iloCheck(t, repo, "fix(app): more\n\nSee may-journal/fitness-runner#12")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0031_3(t *testing.T) {
	t.Parallel()
	repo := iloBranch(t, "topic")
	iloCommit(t, repo, "", "feat(app): start\n\nPlan #12")
	message := "fix(app): more\n# Plan #12\n# ------------------------ >8 ------------------------\nPlan #12\n"
	out, code := iloCheck(t, repo, message)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0031_4(t *testing.T) {
	t.Parallel()
	n := newPR(t, "test: link once", "Closes #12")
	repo := iloOnSandbox(t, sandboxBranch(t))
	out, code := iloCheck(t, repo, "fix(app): more\n\nFixes #12")
	sees(t, out, code, 1, sandbox+"#12 is already linked by open PR #"+itoa(n))
}

func Test0031_5(t *testing.T) {
	t.Parallel()
	repo := iloBranch(t, "topic")
	iloCommit(t, repo, "2026-09-30T10:00:00Z", "feat(app): start\n\nPlan #12")
	git(t, repo, "switch", "-q", "main")
	iloCommit(t, repo, "2026-09-30T13:00:00Z", "feat(app): more on main\n\nPlan #12")
	git(t, repo, "push", "-q", "origin", "main")
	git(t, repo, "switch", "-q", "topic")
	user(t, repo, nil, "merge", "-q", "--no-edit", "main")
	out, code := iloPullRequest(t, repo)
	sees(t, out, code, 0, "All 1 checks passed · 2 files scanned")
}

func Test0031_6(t *testing.T) {
	t.Parallel()
	repo := iloBranch(t, "topic")
	iloCommit(t, repo, "2026-09-30T10:00:00Z", "feat(app): start\n\nPlan #12")
	iloCommit(t, repo, "2026-09-30T11:00:00Z", "fix(app): more\n\nPlan #12")
	out, code := iloPullRequest(t, repo)
	sees(t, out, code, 1, "commit "+iloHead(t, repo)[:7]+` ("fix(app): more"): #12 is already linked by commit`, "--force-with-lease")
}

func Test0031_7(t *testing.T) {
	t.Parallel()
	repo := iloBranch(t, "topic")
	iloCommit(t, repo, "2026-09-30T13:00:00Z", "fix(app): more\n\nFixes #12")
	out, code := iloPullRequest(t, repo)
	sees(t, out, code, 1, "#12 is already linked by open PR #9", "--force-with-lease")
}

func Test0031_8(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "issue-link-once")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "issue-link-once")
}
