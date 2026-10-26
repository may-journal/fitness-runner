package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
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
	gated := !disabledCheck("plan-trailer")
	s := bufio.NewScanner(os.Stdin)
	var updates []string
	for s.Scan() {
		updates = append(updates, s.Text())
		if code := gateRef(gated, s.Text()); code != 0 {
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
	if len(branches) == 0 || disabledCheck("local-branch") {
		return 0
	}
	data, err := json.Marshal(branches)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pre-push:", err)
		return 1
	}
	// The handoff only saves reads, so local-branch reads from GitHub without it.
	if cleanup, err := planReads.Handoff(); err == nil {
		defer cleanup()
	}
	_ = os.Setenv("FITNESS_INTERNAL_HOOK", "pre-push")
	_ = os.Setenv("FITNESS_PUSH_BRANCHES", string(data))
	return run([]string{"local-branch"})
}

// gateRef applies the Plan gate to one ref update, unless external
// disabledChecks turned plan-trailer off: the gate enforces that trailer.
func gateRef(gated bool, line string) int {
	if !gated {
		return 0
	}
	return checkRefUpdate(line)
}

// disabledCheck reports whether the repo's external disabledChecks names
// name, so a hook skips it.
func disabledCheck(name string) bool {
	cfg, err := loadConfig(".")
	return err == nil && turnedOff(cfg, name)
}

// checkRefUpdate gates one "local_ref local_sha remote_ref remote_sha" line: a
// deletion or an all-chore/docs push passes; otherwise it needs an approved
// Plan.
func checkRefUpdate(line string) int {
	f := strings.Fields(line)
	if len(f) < 4 || f[1] == zeroSha {
		return 0
	}
	subjects, messages, err := pushedCommits(f[1], f[3])
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	if allChoreDocsSubjects(subjects) {
		return 0
	}
	return gatePlans(messages, f[1])
}

// gatePlans passes when the push traces to an approved Plan Issue.
func gatePlans(messages, local string) int {
	plans, err := planCandidates(messages, local)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	return checkPlanApproval(plans)
}

// pushedCommits reads the subjects and messages of the commits a ref update
// pushes; none means nothing to gate. A stalled git call returns its error.
func pushedCommits(local, remote string) (subjects []string, messages string, err error) {
	commits, err := commitsOf(local, remote)
	if len(commits) == 0 {
		return nil, "", err
	}
	return readCommits(commits)
}

// commitsOf lists the commits a ref update pushes: those not already on origin.
// Excluding every origin ref, not just the remote tip, keeps commits merged in
// from another branch (say main) out of the gate. A failed run reads as no
// commits, and a stalled one returns its error.
func commitsOf(local, remote string) ([]string, error) {
	if remote == zeroSha {
		remote = ""
	}
	return stalledOnly(gitx.Repo(".").ToPush(local, remote))
}

// allChoreDocsSubjects reports whether every pushed commit is a chore, docs,
// or merge commit. A merge is exempt because the commits it brings in are
// gated on their own.
func allChoreDocsSubjects(subjects []string) bool {
	for _, s := range subjects {
		if !strings.HasPrefix(s, "chore") && !strings.HasPrefix(s, "docs") && !strings.HasPrefix(s, "Merge ") {
			return false
		}
	}
	return true
}

// readCommits returns each commit's subject line, in order, and all their
// messages, from one git run. A commit git cannot read gets an empty
// subject, which is never exempt.
func readCommits(commits []string) ([]string, string, error) {
	log, err := logCommits(commits, "%H%x1f%s%x1f%B%x1e")
	if err != nil {
		return nil, "", err
	}
	byHash := map[string]string{}
	var messages strings.Builder
	for _, record := range strings.Split(log, "\x1e") {
		f := strings.SplitN(strings.TrimLeft(record, "\n"), "\x1f", 3)
		if len(f) == 3 {
			byHash[f[0]] = strings.TrimSpace(f[1])
			messages.WriteString(f[2] + "\n")
		}
	}
	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = byHash[c]
	}
	return subjects, messages.String(), nil
}

// logCommits formats exactly the given commits, in order, in one git run.
// --no-walk reads only those commits, not their ancestors, and --stdin takes
// them however many a push holds. A failed run reads as no output, and a
// stalled one returns its error.
func logCommits(commits []string, format string) (string, error) {
	return stalledOnly(gitx.Repo(".").Log(commits, format))
}

// planCandidates returns the Plan numbers a push may trace to: those in the
// pushed commits' messages, else those anywhere on the branch.
func planCandidates(messages, local string) ([]string, error) {
	if plans := gitx.PlanNumbers(messages); len(plans) > 0 {
		return plans, nil
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
func planRefs(commits []string) ([]string, error) {
	if len(commits) == 0 {
		return nil, nil
	}
	log, err := logCommits(commits, "%B")
	return gitx.PlanNumbers(log), err
}
