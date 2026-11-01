package check

import (
	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "plan-structure", Scope: checkkit.ScopeBlob, Blob: true, ContextInlineArg: "--body"},
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

// run judges the body by the repo's Plan template when it has one, else planSpec.
func run(root string, args []string) (checkkit.Result, error) {
	return bodycheck.RunIf(args, bodycheck.PlanBody, func(body string) []string {
		return mdtemplate.Validate(body, mdtemplate.SpecFor(root, "Plan", planSpec))
	})
}
