package metadata

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// LocalBranch judges live metadata through the normal check result protocol.
func LocalBranch(root string, args []string) (checkkit.Result, error) {
	if len(args) != 0 {
		return checkkit.Fail(0, "usage: fitness branch-check"), nil
	}
	if !localBranchApplicable(root) {
		fmt.Fprintln(os.Stderr, "local-branch: no local GitHub branch context; skipped.")
		return checkkit.Pass(0), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	branches, err := branchContexts(ctx, root)
	if err != nil {
		return branchReadResult(err), nil
	}
	result := checkkit.Pass(0)
	for _, branch := range branches {
		next := checkBranch(ctx, root, branch)
		result.Errors = append(result.Errors, next.Errors...)
		result.FilesChecked += next.FilesChecked
	}
	result.Ok = len(result.Errors) == 0
	return result, nil
}

func localBranchApplicable(root string) bool {
	return !report.Enabled() && os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-commit" && gitx.OriginSlug(root) != ""
}

func checkBranch(ctx context.Context, root string, branch gitx.BranchContext) checkkit.Result {
	pr, repo, err := discoverBranchPR(ctx, root, branch.Name)
	if err != nil || pr == nil {
		return branchReadResult(err)
	}
	pr.BaseSHA = currentBase(ctx, root, repo, pr)
	gh := aftereffect.ReadOnlyClient(ctx, repo)
	b := blobChecker{root}
	plans, err := branchPlans(ctx, root, repo, branch.Tip, pr)
	if err != nil {
		return branchReadResult(err)
	}
	if err := readBranchChangelog(ctx, root, branch.Tip, pr); err != nil {
		return branchReadResult(err)
	}
	nums := b.closedIssues(pr.Body)
	nums = append(nums, plans...)
	errs := judgeBranchPR(pr, gh, b)
	issues, readErrs := readIssues(uniqueIssues(nums), gh)
	errs = append(errs, readErrs...)
	errs = append(errs, judgeBranchPlans(issues, repo, b)...)
	return checkkit.Result{Ok: len(errs) == 0, Errors: errs, FilesChecked: 1 + len(issues)}
}

// readBranchChangelog gives pr the local tip's entry files, so pr-check
// judges them before the push.
func readBranchChangelog(ctx context.Context, root, tip string, pr *branchPR) error {
	files, err := branchEntryFiles(ctx, root, tip, pr)
	pr.EntryFiles = &files
	return err
}

func branchReadResult(err error) checkkit.Result {
	if err == nil {
		return checkkit.Pass(0)
	}
	return checkkit.Fail(0, fmt.Sprintf("%v; check gh authentication and network access, then retry fitness branch-check", err))
}

func judgeBranchPR(pr *branchPR, gh aftereffect.Client, b blobChecker) []string {
	fmt.Fprintf(os.Stderr, "PR: %s\n", pr.URL)
	rep := newTargetReport("pr-check", "pull")
	rep.output = os.Stderr
	errs := ghCommands["pr-check"].judge(pr.Target, gh, b)
	rep.add(pr.Number, errs)
	rep.finish()
	if len(b.closedIssues(pr.Body)) == 0 {
		fmt.Fprintln(os.Stderr, "branch-check: no closing issue links; PR closing-link rules still apply.")
	}
	return errs
}

// judgeBranchPlans leaves ordinary issues to the PR's checklist rules.
func judgeBranchPlans(issues []linkedIssue, repo string, b blobChecker) []string {
	rep := newTargetReport("plan-check", "issues")
	rep.output = os.Stderr
	var errs []string
	for _, issue := range issues {
		fmt.Fprintf(os.Stderr, "Issue: https://github.com/%s/issues/%d\n", repo, issue.number)
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

// currentBase is the base commit GitHub will diff from once the tip is
// pushed. The API's base commit only moves when the PR's head is pushed, so
// after the base branch advances and the branch rebases, it would count other
// PRs' commits as this branch's. The fetched base branch replaces it when it
// already contains that commit. A fork's origin is not the base repository,
// so a fork PR keeps the API's commit.
func currentBase(ctx context.Context, root, repo string, pr *branchPR) string {
	if pr.BaseSHA == "" || !strings.EqualFold(pr.Source, repo) {
		return pr.BaseSHA
	}
	if fetched := gitx.FetchedBase(ctx, root, pr.Base); fetched != "" && contains(ctx, root, fetched, pr.BaseSHA) {
		return fetched
	}
	return pr.BaseSHA
}

// contains reports whether commit is in the history of tip.
func contains(ctx context.Context, root, tip, commit string) bool {
	return exec.CommandContext(ctx, "git", "-C", root, "merge-base", "--is-ancestor", commit, tip).Run() == nil
}

// branchPlans uses the base commit from currentBase, even when origin is a
// stale fork.
func branchPlans(ctx context.Context, root, repo, tip string, pr *branchPR) ([]int, error) {
	if pr.BaseSHA == "" {
		return nil, fmt.Errorf("GitHub PR response has no base commit; retry fitness branch-check")
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", "--format=%B", pr.BaseSHA+".."+tip, "--")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read branch Plan trailers from base commit %s; fetch %s from https://github.com/%s.git and retry: %w", pr.BaseSHA, pr.Base, repo, err)
	}
	var nums []int
	for _, text := range gitx.PlanNumbers(string(out)) {
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
// A timeout, a missing gh, or an auth or network failure is a
// checkkit.Internal error, so it reads as the environment, not a finding.
func branchOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = root
	o := checkkit.ExecCombined(cmd)
	if ctx.Err() != nil {
		return nil, checkkit.Internal(fmt.Errorf("GitHub reads timed out: %w", ctx.Err()))
	}
	if o.Err == nil {
		return []byte(o.Stdout), nil
	}
	err := fmt.Errorf("gh %s: %w: %s", args[0], o.Err, strings.TrimSpace(o.Stdout))
	if aftereffect.GHEnvironment(o) {
		return nil, checkkit.Internal(err)
	}
	return nil, err
}
