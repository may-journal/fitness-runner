// Command fitness-check-plan-trailer validates an optional `Plan #NN` commit
// trailer: a commit may reference the Plan Issue it implements with a line
// reading exactly `Plan #<number>`. The trailer is optional — a message
// without one passes — but a plan reference that is present must be well
// formed, so `Plan 63`, `plan #63`, or `Plan: 63` fail with a fix hint.
//
// Like semantic-commit it reads the proposed message from the context-inline
// `--message` value, falling back to the HEAD commit message.
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
		Describe: checkkit.Describe{Name: "plan-trailer", ContextInlineArg: "--message"},
		Run:      run,
	})
}

var (
	// planRefRe matches a line that is entirely a plan reference: `Plan`
	// followed by separators and a number, and nothing else. It will not
	// match body prose like "Plan the rollout in 3 steps".
	planRefRe = regexp.MustCompile(`(?mi)^\s*plan[ :#]+\s*\d+\s*$`)
	// validPlanRe is the one accepted spelling.
	validPlanRe = regexp.MustCompile(`^Plan #\d+$`)
)

func run(root string, _ []string) (checkkit.Result, error) {
	msg, provided := checkkit.CtxMessage()
	if !provided {
		msg = gitx.HeadMessage(root)
	}
	return judge(msg), nil
}

// judge flags every plan-reference line that is not exactly `Plan #<number>`;
// a message with no plan reference passes.
func judge(msg string) checkkit.Result {
	var errs []string
	for _, line := range planRefRe.FindAllString(msg, -1) {
		t := strings.TrimSpace(line)
		if !validPlanRe.MatchString(t) {
			errs = append(errs, fmt.Sprintf("plan trailer \"%s\" must read exactly \"Plan #<number>\"", t))
		}
	}
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...)
	}
	return checkkit.Pass(1)
}
