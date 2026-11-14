package check

import (
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// Judge runs the PR and Plan judgments. The engine sets it, since they share
// the engine's GitHub reads and the pre-push gate's handoff.
var Judge = withoutEngine

// withoutEngine fails a build that holds this check but not the engine, which
// has no judgments to run.
func withoutEngine(string, []string) (checkkit.Result, error) {
	return checkkit.Fail(0, "local-branch runs only inside fitness, which supplies its PR and Plan judgments"), nil
}

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", Scope: checkkit.ScopeGitHub},
		Run:      Judge,
	})
}
