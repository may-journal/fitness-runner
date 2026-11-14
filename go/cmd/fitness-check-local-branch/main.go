package check

import (
	"errors"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// withoutEngine reports that a build of this check without the engine could
// not judge, since the engine supplies the PR and Plan judgments.
func withoutEngine(string, []string) (checkkit.Result, error) {
	return checkkit.Result{}, errors.New("local-branch runs only inside fitness, which supplies its PR and Plan judgments")
}

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", Scope: checkkit.ScopeGitHub},
		Run:      withoutEngine,
	})
}
