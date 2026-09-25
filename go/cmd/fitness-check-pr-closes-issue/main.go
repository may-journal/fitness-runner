// Command fitness-check-pr-closes-issue validates that a pull request
// description closes at least one issue on merge. GitHub auto-closes an issue
// only for a fixed set of keywords, so a PR that merely references an issue
// (`Implements #12`, `addresses #12`) leaves it open — this check fails that.
//
// Two rules run over the body:
//   - Every PR must carry at least one closing keyword (`close`/`fix`/
//     `resolve` and their tenses) linking an issue. A body with only
//     references, or none, is invalid.
//   - Every issue the PR says it implements (`Implements #NN`, or a `Plan #NN`
//     line) must be among the closed issues.
//
// It reads its target from stdin, a `--body-file` path (`-` meaning stdin), or
// the runner's context-inline `--body` value, and passes inert with no input.
package main

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "pr-closes-issue", ContextInlineArg: "--body"},
		Run:      run,
	})
}

// closingKeywords is GitHub's fixed set of nine issue-closing keywords, as one
// case-insensitive alternation. GitHub auto-closes a linked issue on merge for
// exactly these — nothing else counts as a closure.
const closingKeywords = `close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved`

// closingIssueRe matches a closing keyword linking an issue number, e.g.
// `Closes #12` or `Fixes: #12`.
var closingIssueRe = regexp.MustCompile(`(?i)\b(?:` + closingKeywords + `)[:\s]+#(\d+)`)

// implementedIssueRe matches a claim that the PR implements an issue —
// `Implements #12`, `implemented #12`, or a `Plan #12` line — which must be
// backed by a closing keyword for that same issue.
var implementedIssueRe = regexp.MustCompile(`(?i)\b(?:implement(?:s|ed)?|plan)[:\s]+#(\d+)`)

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.Run(args, validate)
}

// validate reports the PR body's closure violations: a body with no closing
// keyword at all, and any implemented issue left unclosed.
func validate(body string) []string {
	var errors []string
	if !closingIssueRe.MatchString(body) {
		errors = append(errors,
			"PR description has no GitHub closing keyword (close/fix/resolve + #NN) — every PR must close at least one issue on merge")
	}
	closed := closedIssues(body)
	for _, n := range implementedIssues(body) {
		if !closed[n] {
			errors = append(errors, fmt.Sprintf(
				"PR says it implements #%d but no closing keyword (close/fix/resolve + #%d) closes it", n, n))
		}
	}
	return errors
}

// closedIssues is the set of issue numbers the body closes with a keyword.
func closedIssues(body string) map[int]bool {
	set := map[int]bool{}
	for _, m := range closingIssueRe.FindAllStringSubmatch(body, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			set[n] = true
		}
	}
	return set
}

// implementedIssues lists, in first-seen order without duplicates, the issue
// numbers the body claims to implement.
func implementedIssues(body string) []int {
	var nums []int
	seen := map[int]bool{}
	for _, m := range implementedIssueRe.FindAllStringSubmatch(body, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		nums = append(nums, n)
	}
	return nums
}
