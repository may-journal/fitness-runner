package check

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/checklist"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "issue-checklist", Scope: checkkit.ScopeBlob, Comments: checkkit.CommentsNotApplicable, Blob: true, ContextInlineArg: "--body"},
		Run:      run,
	})
}

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.RunIf(args, bodycheck.Closing, func(body string) []string {
		var errs []string
		for _, item := range checklist.Unchecked(body) {
			errs = append(errs, fmt.Sprintf("unchecked item: %q", item))
		}
		return errs
	})
}
