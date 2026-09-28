package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const zeroSha = "0000000000000000000000000000000000000000"

var (
	planRe    = regexp.MustCompile(`(?i)plan #\s*(\d+)`)
	approveRe = regexp.MustCompile(`(?i)approve`)
)

// hookPrePush reads git's push ref updates on stdin and blocks the push unless
// every non-chore/docs update traces to an approved Plan Issue.
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
	return checkPlanApproval(commits)
}

// commitsOf lists the commits a ref update pushes: everything not yet on origin
// for a new branch, else the remote..local range.
func commitsOf(local, remote string) []string {
	args := []string{"rev-list", remote + ".." + local}
	if remote == zeroSha {
		args = []string{"rev-list", local, "--not", "--remotes=origin"}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// allChoreDocs reports whether every pushed commit is a chore or docs commit.
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
		if !strings.HasPrefix(s, "chore") && !strings.HasPrefix(s, "docs") {
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

// checkPlanApproval passes when any pushed commit names an approved Plan Issue.
func checkPlanApproval(commits []string) int {
	plans := planRefs(commits)
	if len(plans) == 0 {
		fmt.Fprintln(os.Stderr, "pre-push: no 'Plan #NN' trailer in the pushed commits.")
		fmt.Fprintln(os.Stderr, "  Add a 'Plan #NN' trailer, or push only chore/docs commits.")
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
func planRefs(commits []string) []string {
	args := append([]string{"log", "--format=%B"}, commits...)
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
