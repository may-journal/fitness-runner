package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/spotless"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "spotless", Scope: checkkit.ScopeFiles, Stage: checkkit.StageTests},
		Run:      spotless.Run,
	})
}
