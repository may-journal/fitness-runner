// Command fitness-check-issue-checklist fails an issue body whose task list
// still has unchecked items, so closing it would close unfinished work.
//
// It reads its target from a `--body-file` path (`-` meaning stdin), the
// runner's context-inline `--body` value, or stdin, and passes inert with no
// input. `fitness close-check` runs it on a closed issue and `fitness
// pr-check` on each issue a PR closes.
package main

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/checklist"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "issue-checklist", ContextInlineArg: "--body"},
		Run:      run,
	})
}

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.Run(args, func(body string) []string {
		var errs []string
		for _, item := range checklist.Unchecked(body) {
			errs = append(errs, fmt.Sprintf("unchecked item: %q", item))
		}
		return errs
	})
}
