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
	"os"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/epic"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
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

// run judges the body's sections by the repo's Epic template when it has
// one, else epicSpec, then its list. A bare #N reads as an issue in the
// judged repo, so both spellings of one issue match.
func run(root string, args []string) (checkkit.Result, error) {
	return bodycheck.RunIf(args, bodycheck.EpicBody, func(body string) []string {
		spec := mdtemplate.SpecFor(root, "Epic", epicSpec)
		return append(mdtemplate.Validate(body, spec), epic.Validate(body, repoOf(root))...)
	})
}

// repoOf is the repo a bare #N names: the Actions repository the epic lives
// in, else the checkout's GitHub origin, else none.
func repoOf(root string) string {
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		return repo
	}
	return gitx.OriginSlug(root)
}
