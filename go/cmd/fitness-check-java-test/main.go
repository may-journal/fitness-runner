package check

import (
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/lang/java/tests"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-test", Scope: checkkit.ScopeFiles, Stage: checkkit.StageTests},
		Run:      tests.Run,
	})
}
