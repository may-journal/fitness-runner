package requirements

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

// longBranchCommits is how many commits longBranch adds to the branch.
const longBranchCommits = 1000

// commitChecks are the checks that read a branch's commits.
var commitChecks = []string{"commit-attribution", "semantic-commit", "local-branch"}

// longBranch is branchRepo with longBranchCommits commits that one
// fast-import call writes, each with a conventional subject and AI trailers.
// Its open PR is based before them and names its base repo, as GitHub does.
func longBranch(t *testing.T) (string, response) {
	t.Helper()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	var answer map[string]any
	mustDo(t, json.Unmarshal([]byte(pr.Stdout), &answer))
	answer["repo"] = repo0014
	data, err := json.Marshal(answer)
	mustDo(t, err)
	pr.Stdout = string(data) + "\n"
	var stream strings.Builder
	for i := range longBranchCommits {
		msg := fmt.Sprintf("test(app): add step %d\n\nAI-Tools: Claude Code\nAI-Models: Claude\n", i)
		fmt.Fprintf(&stream, "commit refs/heads/work\ncommitter t <t@t> %d +0000\ndata %d\n%s\n", 1700000000+i, len(msg), msg)
		if i == 0 {
			stream.WriteString("from refs/remotes/origin/main\n\n")
		}
	}
	gitInput(t, repo, stream.String(), "fast-import", "--quiet")
	return repo, pr
}

func Test0153_1(t *testing.T) {
	t.Parallel()
	repo, pr := longBranch(t)
	rp := standIn(t, map[string][]response{"gh": {pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan")}})
	defer func() {
		if rp.called("gh", "repo view") {
			t.Errorf("local-branch asked gh for the repo, which the PR answer names: %q", rp.calls("gh"))
		}
	}()
	gitPath, calls := loggedGit(t)
	gitDir, _, _ := strings.Cut(strings.TrimPrefix(gitPath, "PATH="), string(os.PathListSeparator))
	env := append([]string{"PATH=" + gitDir + string(os.PathListSeparator) + strings.TrimPrefix(rp.env[0], "PATH=")}, rp.env[1:]...)
	out, code := fitness(t, repo, env, "--policy=external", "--checks="+strings.Join(commitChecks, ","))
	sees(t, out, code, 0, "All 3 checks passed")
	// The fast total bounds the time. A long branch costs git log calls, one per
	// check at any length.
	if logs := gitLogCalls(t, calls); logs > len(commitChecks) {
		t.Errorf("the commit checks ran git log %d times on a branch of %d commits, want at most one each", logs, longBranchCommits)
	}
}

// gitLogCalls counts the git log calls loggedGit kept. None fails the test,
// since then nothing read the branch.
func gitLogCalls(t *testing.T, calls string) int {
	t.Helper()
	raw, _ := os.ReadFile(calls)
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if slices.Contains(strings.Fields(line), "log") {
			n++
		}
	}
	if n == 0 {
		t.Fatalf("no check ran git log through the logged git:\n%s", raw)
	}
	return n
}
