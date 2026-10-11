package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
)

// runBranchCheck judges live metadata without invoking any GitHub effects.
func runBranchCheck(args []string) int {
	if len(args) != 0 {
		return ghFailure("branch-check", fmt.Errorf("usage: fitness branch-check"))
	}
	root, err := os.Getwd()
	if err != nil {
		return ghFailure("branch-check", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := checkBranch(ctx, root); err != nil {
		return ghFailure("branch-check", fmt.Errorf("%w; check gh authentication and network access, then retry fitness branch-check", err))
	}
	return 0
}

func checkBranch(ctx context.Context, root string) error {
	pr, repo, err := discoverBranchPR(ctx, root)
	if err != nil || pr == nil {
		return err
	}
	gh := aftereffect.ReadOnlyClient(ctx, repo)
	b := blobChecker{root}
	plans, err := branchPlans(ctx, root, repo, pr)
	if err != nil {
		return err
	}
	nums := b.closedIssues(pr.Body)
	nums = append(nums, plans...)
	errs := judgeBranchPR(pr, gh, b)
	issues, readErrs := readIssues(uniqueIssues(nums), gh)
	errs = append(errs, readErrs...)
	fmt.Fprint(os.Stderr, strings.Join(readErrs, "\n"))
	errs = append(errs, judgeBranchPlans(issues, repo, b)...)
	if len(errs) > 0 {
		return fmt.Errorf("branch metadata validation failed")
	}
	return nil
}

func judgeBranchPR(pr *branchPR, gh aftereffect.Client, b blobChecker) []string {
	fmt.Printf("PR: %s\n", pr.URL)
	rep := newTargetReport("pr-check", "pull")
	errs := ghCommands["pr-check"].judge(pr.Target, gh, b)
	rep.add(pr.Number, errs)
	rep.finish()
	if len(b.closedIssues(pr.Body)) == 0 {
		fmt.Println("branch-check: no closing issue links; PR closing-link rules still apply.")
	}
	return errs
}

// judgeBranchPlans leaves ordinary issues to the PR's checklist rules.
func judgeBranchPlans(issues []linkedIssue, repo string, b blobChecker) []string {
	rep := newTargetReport("plan-check", "issues")
	var errs []string
	for _, issue := range issues {
		fmt.Printf("Issue: https://github.com/%s/issues/%d\n", repo, issue.number)
		if !aftereffect.HasLabel(issue.labels, "Plan") {
			continue
		}
		findings := b.judge(ghCommands["plan-check"].checks, issue.body)
		rep.add(issue.number, findings)
		errs = append(errs, findings...)
	}
	if len(rep.targets) > 0 {
		rep.finish()
	}
	return errs
}

// branchPlans uses the API's base commit, even when origin is a stale fork.
func branchPlans(ctx context.Context, root, repo string, pr *branchPR) ([]int, error) {
	if pr.BaseSHA == "" {
		return nil, fmt.Errorf("GitHub PR response has no base commit; retry fitness branch-check")
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", "--format=%B", pr.BaseSHA+"..HEAD", "--")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read branch Plan trailers from base commit %s; fetch %s from https://github.com/%s.git and retry: %w", pr.BaseSHA, pr.Base, repo, err)
	}
	var nums []int
	for _, text := range planNumbers(string(out)) {
		n, _ := strconv.Atoi(text)
		nums = append(nums, n)
	}
	return nums, nil
}

func uniqueIssues(nums []int) []int {
	seen := map[int]bool{}
	var unique []int
	for _, n := range nums {
		if !seen[n] {
			unique = append(unique, n)
			seen[n] = true
		}
	}
	return unique
}

// branchOutput includes the actionable gh auth/network error without hiding
// the command's failure, and shares the whole run's remote-read deadline.
func branchOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("GitHub reads timed out: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("gh %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
