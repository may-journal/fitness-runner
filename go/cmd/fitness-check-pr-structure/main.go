// Command fitness-check-pr-structure validates that a pull request description
// follows the PR template: a one-line blockquote summary, a `## Background`
// section, and a `## Changelog` section with at least one bullet, and no other
// `##` sections — the "what changed" lives in the diff, not the prose.
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
		Describe: checkkit.Describe{Name: "pr-structure", ContextInlineArg: "--body"},
		Run:      run,
	})
}

// prSpec is the PR template shape: a summary, Background, and a Changelog with
// at least one bullet.
var prSpec = mdtemplate.Spec{
	Noun:        "PR",
	Placeholder: "REPLACE-ME",
	Sections: []mdtemplate.Section{
		{Heading: "Background", Requires: mdtemplate.Prose},
		{Heading: "Changelog", Requires: mdtemplate.Bullet},
	},
}

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.Run(args, func(body string) []string {
		return mdtemplate.Validate(body, prSpec)
	})
}
