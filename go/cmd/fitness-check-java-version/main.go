package check

import (
	"github.com/may-journal/fitness-runner/go/internal/lang/java/version"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-version", Scope: checkkit.ScopeFiles},
		Run:      version.Run,
	})
}
