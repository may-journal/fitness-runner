package check

import (
	"github.com/may-journal/fitness-runner/go/pkg/bodycheck"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/mdtemplate"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "pr-structure", Scope: checkkit.ScopeBlob, Blob: true, ContextInlineArg: "--body"},
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
	return bodycheck.RunIf(args, bodycheck.PRBody, func(body string) []string {
		return mdtemplate.Validate(body, prSpec)
	})
}
