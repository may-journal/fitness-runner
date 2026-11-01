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
		Describe: checkkit.Describe{Name: "plan-trailer", Scope: checkkit.ScopeGit, ContextInlineArg: "--message"},
		Run:      run,
	})
}

var (
	// planRefRe matches a line that is entirely a plan reference: `Plan`
	// followed by separators, an optional repo or issue URL, and a number.
	// Nothing else may be on the line. It will not match body prose like "Plan the rollout
	// in 3 steps".
	planRefRe = regexp.MustCompile(`(?mi)^\s*plan[ :#]+\s*(?:https?://\S+/|[\w.-]+/[\w.-]+\s*#?\s*)?\d+\s*$`)
	// validPlanRe is the three accepted spellings: a Plan of this repo, a
	// Plan of another repo, and that Plan's URL.
	validPlanRe = regexp.MustCompile(`^Plan (?:#\d+|[\w.-]+/[\w.-]+#\d+|https://github\.com/[\w.-]+/[\w.-]+/issues/\d+)$`)
)

// planTrailerFix names the accepted spellings in every failure.
const planTrailerFix = `must read exactly "Plan #<number>", "Plan <owner>/<repo>#<number>" or "Plan https://github.com/<owner>/<repo>/issues/<number>"`

func run(root string, _ []string) (checkkit.Result, error) {
	msg, provided := checkkit.CtxMessage()
	if !provided {
		msg = gitx.HeadMessage(root)
	}
	return judge(msg), nil
}

// judge flags every plan-reference line in none of the accepted spellings; a
// message with no plan reference passes.
func judge(msg string) checkkit.Result {
	var errs []string
	for _, line := range planRefRe.FindAllString(msg, -1) {
		t := strings.TrimSpace(line)
		if !validPlanRe.MatchString(t) {
			errs = append(errs, fmt.Sprintf("plan trailer \"%s\" %s", t, planTrailerFix))
		}
	}
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}
