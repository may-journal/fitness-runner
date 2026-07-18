// Command fitness-check-commit-attribution validates that the HEAD or
// proposed commit message discloses AI usage via "AI-Tools:" and
// "AI-Models:" git trailers — the Go port of the commit-attribution check.
// Merge and revert commits are exempt. The runner forwards a proposed
// message via --message (surfaced as FITNESS_CTX_MESSAGE); a message that is
// provided but empty fails rather than falling back to git log.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// requiredTrailers are the AI-disclosure trailers, in error-report order.
var requiredTrailers = []string{"AI-Models", "AI-Tools"}

// msgEmpty is the failure reported when there is no commit message to validate.
const msgEmpty = `No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers to disclose AI usage`

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{
			Name:             "commit-attribution",
			ContextInlineArg: "--message",
		},
		Run: run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return judge(resolveMessage(root)), nil
}

// resolveMessage returns the commit message under validation: the runner's
// context-inline message when provided — even empty, which deliberately does
// not fall back — otherwise the HEAD commit message ("" when git fails).
func resolveMessage(root string) string {
	if message, ok := checkkit.CtxMessage(); ok {
		return message
	}
	return gitx.HeadMessage(root)
}

// judge validates one commit message, mirroring the TS check: a blank
// subject fails with msgEmpty, merge/revert subjects are exempt, and each
// missing required trailer reports its own error.
func judge(message string) checkkit.Result {
	subject := firstLine(message)
	if strings.TrimSpace(subject) == "" {
		return checkkit.Fail(1, msgEmpty)
	}
	if isExemptSubject(subject) {
		return checkkit.Pass(1)
	}
	if errors := missingTrailers(message); len(errors) > 0 {
		return checkkit.Fail(1, errors...)
	}
	return checkkit.Pass(1)
}

// firstLine returns the first line (subject) of a commit message.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// isExemptSubject reports whether the subject is an exempt commit (merge or
// revert) — literal prefixes on the untrimmed subject, exactly like the TS.
func isExemptSubject(subject string) bool {
	return strings.HasPrefix(subject, "Merge ") || strings.HasPrefix(subject, "Revert ")
}

// hasTrailer reports whether the message has the trailer key with a
// non-empty value on its own line.
func hasTrailer(message, key string) bool {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:[ \t]*\S.*$`)
	return re.MatchString(message)
}

// missingTrailers returns an error for each required AI-disclosure trailer
// absent from the message, in requiredTrailers order.
func missingTrailers(message string) []string {
	var errors []string
	for _, key := range requiredTrailers {
		if !hasTrailer(message, key) {
			errors = append(errors, fmt.Sprintf("commit message missing %q trailer", key+":"))
		}
	}
	return errors
}
