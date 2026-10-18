package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

const zeroSha = "0000000000000000000000000000000000000000"

var approveRe = regexp.MustCompile(`(?i)approve`)

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
	issues, err := json.Marshal(planReads.shared())
	if err != nil {
		fmt.Fprintln(os.Stderr, "pre-push:", err)
		return 1
	}
	_ = os.Setenv("FITNESS_INTERNAL_HOOK", "pre-push")
	_ = os.Setenv("FITNESS_PUSH_BRANCHES", string(data))
	_ = os.Setenv(metadata.PushIssuesEnv, string(issues))
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
	args := append([]string{"log", "--no-walk=unsorted", "--format=%H %s"}, commits...)
	out, _ := exec.Command("git", args...).Output()
	byHash := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		hash, subject, _ := strings.Cut(line, " ")
		byHash[hash] = strings.TrimSpace(subject)
	}
	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = byHash[c]
	}
	return subjects
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
	return gitx.PlanNumbers(string(out))
}

// approvedPlan reports whether issue n carries the exact `Plan` label and an
// approval comment.
func approvedPlan(n string) bool {
	issue, ok := planReads.read(n)
	if !ok || !slices.Contains(issue.Labels, "Plan") {
		return false
	}
	return slices.ContainsFunc(issue.Comments, approveRe.MatchString)
}

// planIssue is one issue as the gate reads it through the gh CLI.
type planIssue struct {
	URL      string   `json:"url"`
	Body     string   `json:"body"`
	Labels   []string `json:"labels"`
	Comments []string `json:"comments"`
}

// planCache holds the gate's issue reads, so each issue is read once per push
// and local-branch can reuse them. A failed read is cached as not ok.
type planCache map[string]*planIssue

var planReads = planCache{}

// read returns issue n from one `gh issue view`.
func (c planCache) read(n string) (planIssue, bool) {
	issue, seen := c[n]
	if !seen {
		issue = readPlanIssue(n)
		c[n] = issue
	}
	if issue == nil {
		return planIssue{}, false
	}
	return *issue, true
}

func readPlanIssue(n string) *planIssue {
	out, err := exec.Command("gh", "issue", "view", n, "--json", "url,body,labels,comments",
		"--jq", "{url, body, labels: [.labels[].name], comments: [.comments[].body]}").Output()
	if err != nil {
		return nil
	}
	var issue planIssue
	if json.Unmarshal(out, &issue) != nil {
		return nil
	}
	return &issue
}

// shared lists the issues read so far in the form local-branch takes.
func (c planCache) shared() []metadata.PushIssue {
	var issues []metadata.PushIssue
	for _, issue := range c {
		if issue == nil {
			continue
		}
		if p, ok := metadata.PushIssueAt(issue.URL, issue.Body, issue.Labels); ok {
			issues = append(issues, p)
		}
	}
	return issues
}
