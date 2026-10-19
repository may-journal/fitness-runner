package requirements

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// branchReadOnce proves selection does not append a second metadata run.
func branchReadOnce(t *testing.T, rp replays) {
	t.Helper()
	count := 0
	for _, call := range rp.calls("gh") {
		if strings.Contains(call, "pulls?state=open") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("want one branch discovery, got %d: %q", count, rp.calls("gh"))
	}
}

func Test0067_1(t *testing.T) {
	t.Parallel()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "added one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, "An ordinary issue."))})
	out, code := fitness(t, repo, rp.env, "--policy=external", "--checks=markdown-no-bold-italic,local-branch")
	sees(t, out, code, 1, `"added one file"`, "1 of 2 checks passed")
	if !strings.Contains(row(out, "local-branch"), "failed") {
		t.Fatalf("missing metadata result row: %s", out)
	}
	branchReadOnce(t, rp)
}

func Test0067_2(t *testing.T) {
	t.Parallel()
	bin := filepath.Join(installOnly(t), "fitness-check-local-branch")
	rp := standIn(t, map[string][]response{"gh": nil})
	cmd := exec.Command(bin, "--describe")
	cmd.Env = append(userEnv(), rp.env...)
	out, err := cmd.Output()
	mustDo(t, err)
	sees(t, string(out), cmd.ProcessState.ExitCode(), 0, `"name":"local-branch"`, `"timeoutMs":35000`)
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("describe queried GitHub: %q", rp.calls("gh"))
	}
}

// metadataPush runs real git hooks against a bare push destination and saved PR.
func metadataPushRepo(t *testing.T, title string) (string, []string, replays) {
	t.Helper()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, title, pr0014(11))
	saved := branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))
	saved = append(saved, planView(t, 11, planBody("- [x] Write the tests."), []string{"Plan"}, "I approve"))
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, repo, nil, "init")
	sees(t, out, code, 0, "installed hooks")
	env := pushEnv([]string{"PATH=" + installOnly(t) + string(os.PathListSeparator) + os.Getenv("PATH")}, rp)
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add one file\n\nPlan #11")
	sees(t, out, code, 0, "feat(app): add one file")
	return repo, env, rp
}

func metadataPush(t *testing.T, title string) (string, int, replays) {
	t.Helper()
	repo, env, rp := metadataPushRepo(t, title)
	out, code := user(t, repo, env, "push", "origin", "HEAD:work")
	return out, code, rp
}

func Test0067_3(t *testing.T) {
	t.Parallel()
	out, code, rp := metadataPush(t, "added one file")
	sees(t, out, code, 1, `"added one file"`, "local-branch", "failed")
	branchReadOnce(t, rp)
	pushAsked(t, rp, "11", "comments")
}

func Test0067_9(t *testing.T) {
	t.Parallel()
	out, code, rp := metadataPush(t, "test(app): add one file")
	sees(t, out, code, 0, "local-branch", "All 1 checks passed", "HEAD -> work")
	branchReadOnce(t, rp)
	pushAsked(t, rp, "11", "comments")
	planReadOnce(t, rp, "11")
}

func Test0067_5(t *testing.T) {
	t.Parallel()
	repo, hookEnv := hooked(t, nil)
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repo0014+".git")
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code := user(t, repo, pushEnv(hookEnv, rp), "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 0, "docs(readme): touch nothing")
	if row(out, "local-branch") != "" || len(rp.calls("gh")) != 0 {
		t.Fatalf("pre-commit ran metadata: %s\n%q", out, rp.calls("gh"))
	}
}

func Test0067_6(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code := fitness(t, branchRepo(t, nil), append(rp.env, "GITHUB_ACTIONS=true"))
	// Every installed check runs and passes, local-branch among them.
	sees(t, out, code, 0, allPassed())
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("Actions ran local discovery: %q", rp.calls("gh"))
	}
}

func Test0067_7(t *testing.T) {
	t.Parallel()
	rp := standIn(t, map[string][]response{"gh": nil})
	out, code := fitness(t, example(t, "happyRepo", nil), rp.env, "local-branch")
	sees(t, out, code, 0, "skipped")
	passedWithNoFiles(t, out, "local-branch")
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("non-GitHub repo queried GitHub: %q", rp.calls("gh"))
	}
}

func Test0067_8(t *testing.T) {
	t.Parallel()
	repo, pr := forkBranch(t, "release")
	var data map[string]any
	mustDo(t, json.Unmarshal([]byte(pr.Stdout), &data))
	data["baseSHA"] = strings.Repeat("1", 40)
	raw, err := json.Marshal(data)
	mustDo(t, err)
	pr.Stdout = string(raw)
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
	out, code := fitness(t, repo, rp.env, "local-branch")
	sees(t, out, code, 1, "fetch release from https://github.com/"+repo0014+".git", "retry fitness branch-check")
}
