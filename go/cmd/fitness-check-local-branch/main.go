package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Judge runs the PR and Plan judgments. The engine sets it, since they share
// the engine's GitHub reads and the pre-push gate's handoff.
var Judge func(root string, args []string) (checkkit.Result, error)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", Scope: checkkit.ScopeGitHub},
		Run:      Judge,
	})
}
