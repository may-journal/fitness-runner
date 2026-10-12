// Command fitness-check-pr-structure validates that a pull request description
// follows the PR template: a one-line blockquote summary, a `## Background`
// section, and a `## Changelog` section that links the PR's CHANGELOG.md
// entry, and no other `##` sections. What changed lives once, in the
// CHANGELOG entry and the diff; the description links to it.
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

// prSpec is the PR template shape: a summary, Background, and a Changelog
// linking the PR's entry.
var prSpec = mdtemplate.Spec{
	Noun:        "PR",
	Placeholder: "REPLACE-ME",
	Sections: []mdtemplate.Section{
		{Heading: "Background", Requires: mdtemplate.Prose},
		{Heading: "Changelog", Requires: mdtemplate.ChangelogLink},
	},
}

func run(_ string, args []string) (checkkit.Result, error) {
	return bodycheck.Run(args, func(body string) []string {
		return mdtemplate.Validate(body, prSpec)
	})
}
