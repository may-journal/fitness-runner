package check

import (
	"github.com/may-journal/fitness-runner/go/internal/lang/swift/swiftlint"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "swiftlint", Scope: checkkit.ScopeFiles},
		Run:      swiftlint.Run,
	})
}
