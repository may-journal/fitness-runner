package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// docChecks are the prose and markdown checks every Plan and PR body gets.
// They read the body through --body-file, with --root at the checkout so
// their config resolves.
var docChecks = []string{
	"prose-budget",
	"text-readability",
	"markdown-no-bold-italic",
	"mermaid-callouts",
	"mermaid-callout-why",
	"mermaid-diagram-prose",
	"mermaid-diagram-table-gap",
	"mermaid-legend",
	"cspell",
}

// target is one issue or PR a GitHub command judges.
type target = aftereffect.Target

// blobChecker runs installed check binaries on a GitHub blob, an issue or PR
// body, saved to a file the checks read with --body-file.
type blobChecker struct{ root string }

// judge runs each check on body and returns their findings.
func (b blobChecker) judge(checks []string, body string) []string {
	file, err := writeBody(body)
	if err != nil {
		return []string{err.Error()}
	}
	defer os.Remove(file)
	var errs []string
	for _, name := range checks {
		errs = append(errs, b.errorsOf(name, "--root", b.root, "--body-file", file)...)
	}
	return errs
}

// errorsOf runs one check and returns its errors; a check that prints no
// result fails closed with an error naming it.
func (b blobChecker) errorsOf(name string, args ...string) []string {
	var res checkkit.Result
	if err := json.Unmarshal(lastJSONLine(execCheck(name, args)), &res); err != nil {
		return []string{fmt.Sprintf("%s: no result from the check", name)}
	}
	return res.Errors
}

// closedIssues lists the issues body closes, parsed by pr-closes-issue so the
// Plan side and PR side share one keyword set.
func (b blobChecker) closedIssues(body string) []int {
	file, err := writeBody(body)
	if err != nil {
		return nil
	}
	defer os.Remove(file)
	var nums []int
	_ = json.Unmarshal(lastJSONLine(execCheck("pr-closes-issue", []string{"--emit-closed", "--body-file", file})), &nums)
	return nums
}

// execCheck runs fitness-check-<name> and returns its stdout. A failing check
// still prints its JSON result, so a non-zero exit is not an error here.
func execCheck(name string, args []string) []byte {
	bin, err := findCheckBinary(name)
	if err != nil {
		return nil
	}
	cmd := exec.Command(bin, args...)
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, os.Stderr
	_ = cmd.Run()
	return stdout.Bytes()
}

// writeBody saves body to a temp file for the --body-file checks; the caller
// removes it.
func writeBody(body string) (string, error) {
	f, err := os.CreateTemp("", "fitness-body-*.md")
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(body)
	return f.Name(), err
}

// linkedIssue is an issue a PR closes, as read from GitHub.
type linkedIssue struct {
	number int
	body   string
	labels []string
}

// prLinks judges what a PR points at: its title as a semantic-commit
// subject, its closing keywords, and the checklist of each issue it closes.
func prLinks(t target, gh aftereffect.Client, b blobChecker) []string {
	errs := b.errorsOf("semantic-commit", "--message", t.Title)
	closed := readIssues(b.closedIssues(t.Body), gh)
	errs = append(errs, closesErrors(t.Body, requiredClosures(closed, b), b)...)
	for _, i := range closed {
		for _, e := range b.judge([]string{"issue-checklist"}, i.body) {
			errs = append(errs, fmt.Sprintf("PR closes #%d, which has an %s; tick it once it is done, or stop closing #%d", i.number, e, i.number))
		}
	}
	return errs
}

// readIssues reads each issue once; an unreadable issue is skipped.
func readIssues(nums []int, gh aftereffect.Client) []linkedIssue {
	var issues []linkedIssue
	for _, n := range nums {
		body, labels, err := gh.Issue(n)
		if err == nil {
			issues = append(issues, linkedIssue{n, body, labels})
		}
	}
	return issues
}

// requiredClosures returns the issues a PR must also close: for each Plan it
// closes, the issues that Plan itself closes.
func requiredClosures(closed []linkedIssue, b blobChecker) []int {
	var required []int
	seen := map[int]bool{}
	for _, i := range closed {
		if !aftereffect.HasLabel(i.labels, "Plan") {
			continue
		}
		for _, m := range b.closedIssues(i.body) {
			if !seen[m] {
				seen[m] = true
				required = append(required, m)
			}
		}
	}
	return required
}

// closesErrors runs pr-closes-issue on body, requiring each of required to
// be closed too.
func closesErrors(body string, required []int, b blobChecker) []string {
	file, err := writeBody(body)
	if err != nil {
		return []string{err.Error()}
	}
	defer os.Remove(file)
	args := []string{"--body-file", file}
	if len(required) > 0 {
		nums := make([]string, len(required))
		for i, n := range required {
			nums[i] = strconv.Itoa(n)
		}
		args = append(args, "--require-close="+strings.Join(nums, ","))
	}
	return b.errorsOf("pr-closes-issue", args...)
}
