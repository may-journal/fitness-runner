// Command fitness-check-semantic-commit validates that the commit subject
// follows Conventional Commits `type(scope): description` with the scope
// required — the Go port of the semantic-commit check. One deviation from
// the TypeScript original: it inlines the type list the TS check read from
// the conventional-commit-types package at runtime (same eleven types, same
// order).
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "semantic-commit", ContextInlineArg: "--message"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	msg, provided := checkkit.CtxMessage()
	if !provided {
		msg = gitx.HeadMessage(root)
	}
	return judge(firstLine(msg)), nil
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

// judge validates one subject line, mirroring the TS check: a blank subject
// fails with msgEmpty, a semantic (or Merge) subject passes, and anything
// else fails quoting the subject and the accepted types. filesChecked is
// always 1 — the one commit message.
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
