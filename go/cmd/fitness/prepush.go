package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

const zeroSha = "0000000000000000000000000000000000000000"

// planReads holds the push's Plan reads, so each Plan is read once and
// local-branch judges the same reads.
var planReads = metadata.NewPushPlans()

// hookPrePush reads git's push ref updates on stdin and blocks the push unless
// every non-chore/docs update traces to an approved Plan Issue. Commits link a
// Plan once per branch (issue-link-once), so the Plan may be named by an
// earlier branch commit instead of a pushed commit.
func hookPrePush() int {
	s := bufio.NewScanner(os.Stdin)
	var updates []string
	for s.Scan() {
		updates = append(updates, s.Text())
		if code := checkRefUpdate(s.Text()); code != 0 {
			return code
		}
	}
	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "pre-push: cannot read push updates:", err)
		return 1
	}
	return checkPushedBranches(updates)
}

// checkPushedBranches runs one meta-check over the actual pushed destinations.
func checkPushedBranches(updates []string) int {
	branches := gitx.PushedBranches(updates)
	if len(branches) == 0 {
		return 0
	}
	data, err := json.Marshal(branches)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pre-push:", err)
		return 1
	}
	cleanup, err := planReads.Handoff()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pre-push:", err)
		return 1
	}
	defer cleanup()
	_ = os.Setenv("FITNESS_INTERNAL_HOOK", "pre-push")
	_ = os.Setenv("FITNESS_PUSH_BRANCHES", string(data))
	return run([]string{"local-branch"})
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
	return checkPlanApproval(planCandidates(commits, f[1]))
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
	return allChoreDocsSubjects(commitSubjects(commits))
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

// commitSubjects returns each commit's subject line, in order, from one git
// run. A commit git cannot read gets an empty subject, which is never exempt.
func commitSubjects(commits []string) []string {
	byHash := map[string]string{}
	for _, line := range strings.Split(logCommits(commits, "%H %s"), "\n") {
		hash, subject, _ := strings.Cut(line, " ")
		byHash[hash] = strings.TrimSpace(subject)
	}
	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = byHash[c]
	}
	return subjects
}

// logCommits formats exactly the given commits, in order, in one git run.
// --no-walk reads only those commits, not their ancestors, and --stdin takes
// them however many a push holds. A failed run reads as no output.
func logCommits(commits []string, format string) string {
	cmd := exec.Command("git", "log", "--no-walk=unsorted", "--stdin", "--format="+format)
	cmd.Stdin = strings.NewReader(strings.Join(commits, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// planCandidates returns the Plan numbers a push may trace to: those in the
// pushed commits, else those anywhere on the branch.
func planCandidates(commits []string, local string) []string {
	if plans := planRefs(commits); len(plans) > 0 {
		return plans
	}
	return planRefs(gitx.BranchCommits(".", local, gitx.DefaultBase(".")))
}

// checkPlanApproval passes when any candidate names an approved Plan Issue.
func checkPlanApproval(plans []string) int {
	if len(plans) == 0 {
		fmt.Fprintln(os.Stderr, "pre-push: no 'Plan #NN' trailer on the branch.")
		fmt.Fprintln(os.Stderr, "  Add a 'Plan #NN' trailer to the branch's first commit, or push only chore/docs commits.")
		return 1
	}
	for _, n := range plans {
		if planReads.Approved(n) {
			return 0
		}
	}
	fmt.Fprintln(os.Stderr, "pre-push: no approved Plan Issue among ("+strings.Join(plans, " ")+").")
	fmt.Fprintln(os.Stderr, "  Get a human to comment an approval on the Plan Issue before pushing.")
	return 1
}

// planRefs returns the distinct Plan numbers referenced across the commits,
// so a trailer on main never stands in for the branch's own.
func planRefs(commits []string) []string {
	if len(commits) == 0 {
		return nil
	}
	return gitx.PlanNumbers(logCommits(commits, "%B"))
}
