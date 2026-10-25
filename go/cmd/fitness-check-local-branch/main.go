package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", Scope: checkkit.ScopeGitHub, TimeoutMs: 35000},
		Run:      metadata.LocalBranch,
	})
}
