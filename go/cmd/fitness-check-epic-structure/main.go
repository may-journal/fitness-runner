// Command fitness-check-epic-structure validates that an epic — a GitHub
// Issue body under the `Epic` label — follows the Epic issue template: a
// one-line blockquote pitch, a `## Background` section, and a `## What needs
// to happen` list of plain issue references in run order, and no other `##`
// sections.
//
// It reads its target from stdin, a `--body-file` path (`-` meaning stdin), or
// the runner's context-inline `--body` value, and passes inert with no input.
// It never calls GitHub: `fitness issue-check` compares the list with the
// epic's sub-issues.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/epic"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "epic-structure", Blob: true, ContextInlineArg: "--body"},
		Run:      run,
	})
}

// epicSpec is the Epic issue template shape: a pitch, Background, and the
// list of issues under What needs to happen.
var epicSpec = mdtemplate.Spec{
	Noun:        "epic",
	Placeholder: "REPLACE-ME",
	Sections: []mdtemplate.Section{
		{Heading: "Background", Requires: mdtemplate.Prose},
		{Heading: epic.Section, Requires: mdtemplate.Prose},
	},
	BanOpenQuestions: true,
}

func run(root string, args []string) (checkkit.Result, error) {
	return bodycheck.RunIf(args, bodycheck.EpicBody, func(body string) []string {
		return append(mdtemplate.Validate(body, epicSpec), epic.Validate(body, conf.MaxListItems(root))...)
	})
}
