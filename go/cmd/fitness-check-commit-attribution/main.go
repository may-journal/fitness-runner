package check

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// requiredTrailers are the AI-disclosure trailers, in error-report order.
var requiredTrailers = []string{"AI-Models", "AI-Tools"}

// msgEmpty is the failure reported when there is no commit message to validate.
const msgEmpty = `No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers to disclose AI usage`

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{
			Name:             "commit-attribution",
			Scope:            checkkit.ScopeGit,
			ContextInlineArg: "--message",
		},
		Run: run,
	})
}

// run judges the inline message, or HEAD's, once the repo uses the
// AI-disclosure convention. An inline message with a trailer reads no history.
func run(root string, _ []string) (checkkit.Result, error) {
	message, inline := checkkit.CtxMessage()
	if inline && anyTrailer(message) {
		return judge(message), nil
	}
	message, ok := adopted(root, message, inline)
	if !ok {
		return checkkit.Pass(0), nil
	}
	return judge(message), nil
}

// adopted reads recent history once and returns the message to judge, HEAD's
// unless inline, and whether it or that history carries a trailer.
func adopted(root, message string, inline bool) (string, bool) {
	history := gitx.RecentMessages(root, historyDepth)
	if !inline {
		message = head(history)
	}
	return message, anyTrailer(message) || slices.ContainsFunc(history, anyTrailer)
}

// historyDepth is how many recent commits run inspects for a trailer.
const historyDepth = 100

// anyTrailer reports whether text carries any required trailer.
func anyTrailer(text string) bool {
	for _, key := range requiredTrailers {
		if hasTrailer(text, key) {
			return true
		}
	}
	return false
}

// head is HEAD's message, the first of history, or empty when git read
// none.
func head(history []string) string {
	if len(history) == 0 {
		return ""
	}
	return history[0]
}

// judge validates one commit message: a blank subject fails with msgEmpty,
// merge and revert subjects pass, and each missing trailer errors.
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
