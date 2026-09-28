package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// runPRCheck validates the triggering PR's title and description, or every
// open PR on workflow_dispatch, writes the job summary, and fails on any
// violation. The PR check workflows call this in one step.
func runPRCheck() int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ev, err := readEvent()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness pr-check: "+err.Error())
		return 1
	}
	gh, err := newGHClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness pr-check: "+err.Error())
		return 1
	}
	return prCheck(ev, gh, newBodyChecker(root))
}

// prCheck is runPRCheck past the environment: it validates each target PR and
// reports the result.
func prCheck(ev ghEvent, gh githubAPI, c bodyChecker) int {
	targets, err := prTargets(ev, gh)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness pr-check: "+err.Error())
		return 1
	}
	var summary strings.Builder
	summary.WriteString("## PR checks\n\n")
	failed := false
	for _, t := range targets {
		errs := validatePR(t, gh, c)
		failed = failed || len(errs) > 0
		summary.WriteString(prReport(t.Number, errs))
	}
	writeStepSummary(summary.String())
	fmt.Print(summary.String())
	if failed {
		fmt.Fprintln(os.Stderr, "One or more PR descriptions have violations.")
		return 1
	}
	return 0
}

// prTargets is the triggering PR, or every open PR on workflow_dispatch.
func prTargets(ev ghEvent, gh githubAPI) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.openPRs()
	}
	if ev.PullRequest == nil {
		return nil, fmt.Errorf("no pull_request in the %q event", ev.name)
	}
	return []target{*ev.PullRequest}, nil
}

// validatePR runs the PR checks: the title as a semantic-commit subject, the
// description's structure, its closing keywords, and the prose checks.
func validatePR(t target, gh githubAPI, c bodyChecker) []string {
	bodyFile, err := writeBody(t.Body)
	if err != nil {
		return []string{err.Error()}
	}
	defer os.Remove(bodyFile)
	errs := c.errorsOf("semantic-commit", []string{"--message", t.Title}, "")
	errs = append(errs, c.errorsOf("pr-structure", nil, t.Body)...)
	errs = append(errs, c.errorsOf("pr-closes-issue", closesArgs(bodyFile, requiredClosures(c.closedIssues(bodyFile), gh, c)), "")...)
	return append(errs, c.docErrors(bodyFile)...)
}

// requiredClosures returns the issues the PR must also close: for each Plan
// the PR closes, the issues that Plan itself closes.
func requiredClosures(prCloses []int, gh githubAPI, c bodyChecker) []int {
	var required []int
	seen := map[int]bool{}
	for _, n := range prCloses {
		for _, m := range planCloses(n, gh, c) {
			if !seen[m] {
				seen[m] = true
				required = append(required, m)
			}
		}
	}
	return required
}

// planCloses lists the issues that issue n closes when it is a Plan; nil for
// a non-Plan or unreadable issue.
func planCloses(n int, gh githubAPI, c bodyChecker) []int {
	body, labels, err := gh.issue(n)
	if err != nil || !hasLabel(labels, "Plan") {
		return nil
	}
	planFile, err := writeBody(body)
	if err != nil {
		return nil
	}
	defer os.Remove(planFile)
	return c.closedIssues(planFile)
}

// hasLabel reports whether labels contains name exactly.
func hasLabel(labels []string, name string) bool {
	for _, l := range labels {
		if l == name {
			return true
		}
	}
	return false
}

// closesArgs builds the pr-closes-issue arguments, adding --require-close
// when a closed Plan names issues of its own.
func closesArgs(bodyFile string, required []int) []string {
	args := []string{"--body-file", bodyFile}
	if len(required) == 0 {
		return args
	}
	nums := make([]string, len(required))
	for i, n := range required {
		nums[i] = strconv.Itoa(n)
	}
	return append(args, "--require-close="+strings.Join(nums, ","))
}

// prReport is one PR's job-summary entry.
func prReport(n int, errs []string) string {
	if len(errs) == 0 {
		return fmt.Sprintf("✅ #%d: PR description looks good.\n\n", n)
	}
	return fmt.Sprintf("❌ #%d: PR description needs work:\n\n%s\n", n, bulletList(errs))
}

// bulletList renders errs as a markdown list.
func bulletList(errs []string) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString("- " + e + "\n")
	}
	return b.String()
}
