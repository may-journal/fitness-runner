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

// planReads holds the push's PR and Plan reads, so each is read once and
// local-branch judges the same reads. hookPrePush opens it with the push's
// deadline.
var planReads *metadata.PushPlans

// hookPrePush blocks each push update, chore and docs aside, whose branch has
// an open PR closing no approved Plan. Reads share one deadline, and an
// unanswered probe reports info and allows the push.
func hookPrePush() int {
	ctx, cancel := metadata.PushContext()
	defer cancel()
	planReads = metadata.NewPushPlans(ctx)
	gated := gateOnline(!disabledCheck("plan-trailer"))
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
// disabledChecks turned plan-trailer off, which turns the gate off too.
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
// deletion or an all-chore/docs push passes, and any other goes to gatePlans.
func checkRefUpdate(line string) int {
	f := strings.Fields(line)
	if len(f) < 4 || f[1] == zeroSha {
		return 0
	}
	subjects, err := pushedSubjects(f[1], f[3])
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	if allChoreDocsSubjects(subjects) {
		return 0
	}
	return gatePlans(f[2])
}

// gatePlans passes when the pushed branch's open PR closes an approved Plan
// Issue. A branch with no open PR yet passes, as does a tag, since the PR
// check gates the Plan later.
func gatePlans(remoteRef string) int {
	branch, ok := strings.CutPrefix(remoteRef, "refs/heads/")
	if !ok {
		return 0
	}
	plans, pr, err := planReads.PRPlans(".", branch)
	if err != nil {
		return unreadPR(branch, err)
	}
	if pr == "" {
		fmt.Fprintln(os.Stderr, "pre-push: info: "+branch+" has no open PR yet, so its PR check gates the Plan.")
		return 0
	}
	return checkPlanApproval(plans, pr)
}

// unreadPR refuses a push whose PR read failed. Past the deadline it prints
// one line naming the host, else the read's error.
func unreadPR(branch string, err error) int {
	if late := planReads.TimedOut(); late != "" {
		fmt.Fprintln(os.Stderr, "pre-push: "+late+"; check the network or proxy, then push again.")
		return 1
	}
	fmt.Fprintf(os.Stderr, "pre-push: cannot read the open PR for %s: %v\n", branch, err)
	return 1
}

// pushedSubjects reads the subjects of the commits a ref update pushes;
// none means nothing to gate. A stalled git call returns its error.
func pushedSubjects(local, remote string) ([]string, error) {
	commits, err := commitsOf(local, remote)
	if len(commits) == 0 {
		return nil, err
	}
	return readSubjects(commits)
}

// commitsOf lists the commits a ref update pushes that no origin ref holds,
// keeping commits merged in from main out of the gate. A failed run reads
// as none, and a stalled one returns its error.
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

// readSubjects returns each commit's subject line, in order, from one git
// run. A commit git cannot read gets an empty subject, which is never exempt.
func readSubjects(commits []string) ([]string, error) {
	log, err := logCommits(commits, "%H%x1f%s%x1e")
	if err != nil {
		return nil, err
	}
	byHash := map[string]string{}
	for _, record := range strings.Split(log, "\x1e") {
		if hash, subject, ok := strings.Cut(strings.TrimLeft(record, "\n"), "\x1f"); ok {
			byHash[hash] = strings.TrimSpace(subject)
		}
	}
	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = byHash[c]
	}
	return subjects, nil
}

// logCommits formats exactly the given commits, in order, in one git run using
// --no-walk and --stdin; a failed run reads as no output.
func logCommits(commits []string, format string) (string, error) {
	return stalledOnly(gitx.Repo(".").Log(commits, format))
}

// checkPlanApproval passes when the PR at pr closes an approved Plan Issue of
// the repo's owner. It refuses the push once the deadline passes.
func checkPlanApproval(plans []gitx.PlanRef, pr string) int {
	if len(plans) == 0 {
		fmt.Fprintln(os.Stderr, "pre-push: "+pr+" closes no Plan.")
		fmt.Fprintln(os.Stderr, "  The PR body must close the approved Plan: 'Closes #NN', or 'Closes <owner>/<repo>#NN' for a Plan in another repo of this owner.")
		fmt.Fprintln(os.Stderr, "  Or push only chore/docs commits.")
		return 1
	}
	owner, _, _ := strings.Cut(gitx.OriginSlug("."), "/")
	approvers := planApprovers()
	approved, late := approvedAmong(plans, owner, approvers)
	switch {
	case approved:
		return 0
	case late != "":
		fmt.Fprintln(os.Stderr, "pre-push: "+late+"; check the network or proxy, then push again.")
	default:
		refuseUnapproved(plans, owner, approvers)
	}
	return 1
}

// approvedAmong reads the candidates in turn and reports whether one of
// owner's Plans is approved by approvers. It stops at the first read past
// the push's deadline and returns the line naming the host and the deadline.
func approvedAmong(plans []gitx.PlanRef, owner string, approvers []string) (bool, string) {
	for _, p := range plans {
		if ownPlan(p, owner) && planReads.Approved(p, approvers) {
			return true, ""
		}
		if late := planReads.TimedOut(); late != "" {
			return false, late
		}
	}
	return false, ""
}

// planApprovers is the repo's planApprovers setting, empty when unset or
// unreadable.
func planApprovers() []string {
	if cfg, err := loadConfig("."); err == nil && cfg != nil {
		return cfg.PlanApprovers
	}
	return nil
}

// approverNames says who may approve: the listed logins, or with no list
// the repo's owner and its org's members.
func approverNames(approvers []string) string {
	if len(approvers) == 0 {
		return "the repo's owner or an org member (planApprovers is unset)"
	}
	return "one of planApprovers: " + strings.Join(approvers, ", ")
}

// refuseUnapproved explains a refusal: it lists the candidates, names each
// Plan of another owner, which never counts, and says who may approve.
func refuseUnapproved(plans []gitx.PlanRef, owner string, approvers []string) {
	names := make([]string, len(plans))
	for i, p := range plans {
		names[i] = planName(p)
	}
	fmt.Fprintln(os.Stderr, "pre-push: no approved Plan Issue among ("+strings.Join(names, " ")+").")
	for _, p := range plans {
		if !ownPlan(p, owner) {
			fmt.Fprintf(os.Stderr, "  Plan %s is refused: only a Plan owned by %q counts here.\n", p, owner)
		}
	}
	fmt.Fprintln(os.Stderr, "  A Plan needs a comment starting \"I approve\" from "+approverNames(approvers)+".")
}

// planName names p in a refusal: the number alone for a Plan of this repo,
// else owner/repo#number.
func planName(p gitx.PlanRef) string {
	if p.Qualified() {
		return p.String()
	}
	return p.Number
}

// ownPlan reports whether p may approve a push to a repo of owner. A bare Plan
// always may, and a qualified one only from the same owner.
func ownPlan(p gitx.PlanRef, owner string) bool {
	return !p.Qualified() || (owner != "" && strings.EqualFold(p.Owner, owner))
}
