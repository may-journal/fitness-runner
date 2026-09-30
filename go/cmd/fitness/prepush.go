package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

const zeroSha = "0000000000000000000000000000000000000000"

var (
	planRe    = regexp.MustCompile(`(?i)plan #\s*(\d+)`)
	approveRe = regexp.MustCompile(`(?i)approve`)
	issueRe   = regexp.MustCompile(`(?:^|[^\w&/])#(\d+)\b`)
)

// hookPrePush reads git's push ref updates on stdin and blocks the push unless
// every non-chore/docs update traces to an approved Plan Issue. Commits link a
// Plan once per branch (issue-link-once), so the Plan may be named by an
// earlier branch commit or the branch's open PR instead of a pushed commit.
func hookPrePush() int {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		if code := checkRefUpdate(s.Text()); code != 0 {
			return code
		}
	}
	return 0
}

// checkRefUpdate gates one "local_ref local_sha remote_ref remote_sha" line: a
// deletion or an all-chore/docs push passes; otherwise it needs an approved
// Plan.
func checkRefUpdate(line string) int {
	f := strings.Fields(line)
	if len(f) < 4 || f[1] == zeroSha {
		return 0
	}
	commits := commitsOf(f[1], f[3])
	if len(commits) == 0 || allChoreDocs(commits) {
		return 0
	}
	return checkPlanApproval(planCandidates(commits, f[1], strings.TrimPrefix(f[0], "refs/heads/")))
}

// commitsOf lists the commits a ref update pushes: those not already on origin.
// Excluding every origin ref, not just the remote tip, keeps commits merged in
// from another branch (say main) out of the gate.
func commitsOf(local, remote string) []string {
	args := []string{"rev-list", local, "--not", "--remotes=origin"}
	if remote != zeroSha {
		args = append(args, remote)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// allChoreDocs reports whether every pushed commit is a chore, docs, or merge
// commit. A merge is exempt because the commits it brings in are gated on their
// own.
func allChoreDocs(commits []string) bool {
	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = commitSubject(c)
	}
	return allChoreDocsSubjects(subjects)
}

// allChoreDocsSubjects is the pure classifier over commit subjects.
func allChoreDocsSubjects(subjects []string) bool {
	for _, s := range subjects {
		if !strings.HasPrefix(s, "chore") && !strings.HasPrefix(s, "docs") && !strings.HasPrefix(s, "Merge ") {
			return false
		}
	}
	return true
}

// commitSubject returns one commit's subject line.
func commitSubject(c string) string {
	out, _ := exec.Command("git", "log", "-1", "--format=%s", c).Output()
	return strings.TrimSpace(string(out))
}

// planCandidates returns the Plan numbers a push may trace to: those in the
// pushed commits, else those in the whole branch and its open PR body.
func planCandidates(commits []string, local, branch string) []string {
	if plans := planRefs(commits); len(plans) > 0 {
		return plans
	}
	plans := planRefs(gitx.BranchCommits(".", local, gitx.DefaultBase(".")))
	for _, n := range issueNumbers(openPRBody(branch)) {
		if !slices.Contains(plans, n) {
			plans = append(plans, n)
		}
	}
	return plans
}

// openPRBody returns the body of branch's open PR through the gh CLI; empty
// when there is none or gh cannot answer. A variable so tests can stub it.
var openPRBody = func(branch string) string {
	out, err := exec.Command("gh", "pr", "view", branch, "--json", "state,body", "-q", `select(.state == "OPEN") | .body`).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// issueNumbers returns the distinct `#NN` numbers in text, in first-seen
// order. A PR body names its Plan with a closing keyword or `Part of #NN`.
func issueNumbers(text string) []string {
	var nums []string
	for _, m := range issueRe.FindAllStringSubmatch(text, -1) {
		if !slices.Contains(nums, m[1]) {
			nums = append(nums, m[1])
		}
	}
	return nums
}

// checkPlanApproval passes when any candidate names an approved Plan Issue.
func checkPlanApproval(plans []string) int {
	if len(plans) == 0 {
		fmt.Fprintln(os.Stderr, "pre-push: no 'Plan #NN' trailer on the branch and no Plan in its open PR.")
		fmt.Fprintln(os.Stderr, "  Add a 'Plan #NN' trailer to the branch's first commit, or push only chore/docs commits.")
		return 1
	}
	for _, n := range plans {
		if approvedPlan(n) {
			return 0
		}
	}
	fmt.Fprintln(os.Stderr, "pre-push: no approved Plan Issue among ("+strings.Join(plans, " ")+").")
	fmt.Fprintln(os.Stderr, "  Get a human to comment an approval on the Plan Issue before pushing.")
	return 1
}

// planRefs returns the distinct Plan numbers referenced across the commits.
// --no-walk reads only those commits, not their ancestors, so a trailer on
// main never stands in for the branch's own.
func planRefs(commits []string) []string {
	if len(commits) == 0 {
		return nil
	}
	args := append([]string{"log", "--no-walk=unsorted", "--format=%B"}, commits...)
	out, _ := exec.Command("git", args...).Output()
	return planNumbers(string(out))
}

// planNumbers is the pure extractor: the distinct `Plan #NN` numbers in text,
// in first-seen order.
func planNumbers(text string) []string {
	seen := map[string]bool{}
	var nums []string
	for _, m := range planRe.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			nums = append(nums, m[1])
		}
	}
	return nums
}

// approvedPlan reports whether issue n is Plan-labeled and carries an approval
// comment, resolved through the gh CLI.
func approvedPlan(n string) bool {
	if !hasPlanLabel(n) {
		return false
	}
	out, err := exec.Command("gh", "issue", "view", n, "--json", "comments", "-q", ".comments[].body").Output()
	if err != nil {
		return false
	}
	return approveRe.Match(out)
}

// hasPlanLabel reports whether issue n carries the exact `Plan` label.
func hasPlanLabel(n string) bool {
	out, err := exec.Command("gh", "issue", "view", n, "--json", "labels", "-q", ".labels[].name").Output()
	if err != nil {
		return false
	}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l == "Plan" {
			return true
		}
	}
	return false
}
