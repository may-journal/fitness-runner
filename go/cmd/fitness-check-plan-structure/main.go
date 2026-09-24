// Command fitness-check-plan-structure validates that a plan — a GitHub Issue
// body under the `Plan` label — follows the Plan issue template: a one-line
// blockquote pitch, a `## Background` section, and a `## What needs to happen`
// section with at least one checklist item, and no other `##` sections
// (double-especially no "Open questions").
//
// It reads its target from stdin, a `--body-file` path (`-` meaning stdin), or
// the runner's context-inline `--body` value, and passes inert with no input.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "plan-structure", ContextInlineArg: "--body"},
		Run:      run,
	})
}

// planSpec is the Plan issue template shape: a pitch, Background, and a
// checklist under What needs to happen, with Open questions called out.
var planSpec = mdtemplate.Spec{
	Noun:        "plan",
	Placeholder: "REPLACE-ME",
	Sections: []mdtemplate.Section{
		{Heading: "Background", Requires: mdtemplate.Prose},
		{Heading: "What needs to happen", Requires: mdtemplate.Checklist},
	},
	BanOpenQuestions: true,
}

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.Run(args, func(body string) []string {
		return mdtemplate.Validate(body, planSpec)
	})
}
