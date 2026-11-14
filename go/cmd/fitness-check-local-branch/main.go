package check

import (
	"errors"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// Judge runs the PR and Plan judgments. The engine sets it, since they share
// the engine's GitHub reads and the pre-push gate's handoff.
var Judge = withoutEngine

// withoutEngine reports that a build holding this check but not the engine
// could not judge, since it has no judgments to run.
func withoutEngine(string, []string) (checkkit.Result, error) {
	return checkkit.Result{}, errors.New("local-branch runs only inside fitness, which supplies its PR and Plan judgments")
}

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", Scope: checkkit.ScopeGitHub},
		Run:      Judge,
	})
}
