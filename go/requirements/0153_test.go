package requirements

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// longBranchCommits is how many commits longBranch adds to the branch.
const longBranchCommits = 1000

// commitChecks are the checks that read a branch's commits.
var commitChecks = []string{"commit-attribution", "semantic-commit", "local-branch"}

// longBranch is branchRepo with longBranchCommits empty commits on its
// branch, each with a conventional subject and AI trailers. One fast-import
// call writes them all.
func longBranch(t *testing.T) string {
	t.Helper()
	repo := branchRepo(t, nil)
	var stream strings.Builder
	for i := range longBranchCommits {
		msg := fmt.Sprintf("test(app): add step %d\n\nAI-Tools: Claude Code\nAI-Models: Claude\n", i)
		fmt.Fprintf(&stream, "commit refs/heads/work\ncommitter t <t@t> %d +0000\ndata %d\n%s\n", 1700000000+i, len(msg), msg)
		if i == 0 {
			stream.WriteString("from refs/remotes/origin/main\n\n")
		}
	}
	gitInput(t, repo, stream.String(), "fast-import", "--quiet")
	return repo
}

func Test0153_1(t *testing.T) {
	t.Parallel()
	repo := longBranch(t)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))})
	out, code := fitness(t, repo, rp.env, "--policy=external", "--checks="+strings.Join(commitChecks, ","))
	t.Log("\n" + out)
	sees(t, out, code, 0, "All 3 checks passed")
	for _, check := range commitChecks {
		took := rowMillis.FindStringSubmatch(row(out, check))
		if took == nil {
			t.Fatalf("%s shows no time in its row %q", check, row(out, check))
		}
		if ms, _ := strconv.Atoi(took[1]); ms >= 5000 {
			t.Errorf("%s took %d ms on a branch of %d commits, want under 5000 ms", check, ms, longBranchCommits)
		}
	}
}
