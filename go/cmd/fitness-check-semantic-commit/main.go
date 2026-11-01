package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "semantic-commit", Scope: checkkit.ScopeGit, ContextInlineArg: "--message"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	msg, provided := messageArg(args)
	if !provided {
		msg, provided = checkkit.CtxMessage()
	}
	if provided {
		return judge(firstLine(msg)), nil
	}
	if commits, ok := gitx.BranchOwnCommits(root); ok {
		return judgeCommits(root, commits), nil
	}
	return judge(firstLine(gitx.HeadMessage(root))), nil
}

// judgeCommits judges each of the branch's own commits, so history the
// branch inherits from its base never fails it. No own commits pass.
func judgeCommits(root string, commits []string) checkkit.Result {
	result := checkkit.Pass(len(commits))
	for _, sha := range commits {
		result.Errors = append(result.Errors, judge(firstLine(gitx.Message(root, sha))).Errors...)
	}
	result.Ok = len(result.Errors) == 0
	return result
}

// messageArg extracts an explicit --message value (either form) from args.
// The check can then validate a subject passed directly, such as a PR title
// from the pr-check workflow.
// That workflow runs the binary without the runner's context-inline extraction.
func messageArg(args []string) (string, bool) {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--message="); ok {
			return v, true
		}
		if a == "--message" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// semanticTypes is the accepted commit-type vocabulary, inlined from the
// conventional-commit-types package the TS check read, in published order.
var semanticTypes = []string{
	"feat", "fix", "docs", "style", "refactor", "perf",
	"test", "build", "ci", "chore", "revert",
}

var semanticRe = regexp.MustCompile(`^(` + strings.Join(semanticTypes, "|") + `)\([^)]+\): .+`)

// msgEmpty is the failure when there is no subject line to validate.
const msgEmpty = "No commit message to validate; use type(scope): description"

// judge validates one subject line, mirroring the TS check.
// A blank subject fails with msgEmpty, and a semantic (or Merge) subject passes.
// Anything else fails quoting the subject and the accepted types.
// filesChecked is always 1, the one commit message.
func judge(subject string) checkkit.Result {
	if strings.TrimSpace(subject) == "" {
		return checkkit.Fail(1, msgEmpty)
	}
	if isSemanticSubject(subject) {
		return checkkit.Pass(1)
	}
	return checkkit.Fail(1, fmt.Sprintf(
		"Commit message: \"%s\" — use type(scope): description (types: %s)",
		subject, strings.Join(semanticTypes, ", ")))
}

// isSemanticSubject reports whether subject follows type(scope): description
// (or is a Merge commit).
func isSemanticSubject(subject string) bool {
	if strings.HasPrefix(subject, "Merge ") {
		return true
	}
	return semanticRe.MatchString(subject)
}

// firstLine returns the first line of s (the subject line of a commit
// message).
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
