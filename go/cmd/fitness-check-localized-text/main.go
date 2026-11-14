package check

import (
	"github.com/may-journal/fitness-runner/go/internal/lang/swift/localized"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: localized.Name, Scope: checkkit.ScopeFiles},
		Run:      localized.Run,
	})
}
