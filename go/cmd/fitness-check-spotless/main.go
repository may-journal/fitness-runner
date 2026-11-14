package check

import (
	"github.com/may-journal/fitness-runner/go/internal/lang/java/spotless"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "spotless", Scope: checkkit.ScopeFiles, Stage: checkkit.StageTests},
		Run:      spotless.Run,
	})
}
