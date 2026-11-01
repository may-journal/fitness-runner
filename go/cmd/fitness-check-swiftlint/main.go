package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift/swiftlint"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "swiftlint", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsJudged, TimeoutMs: 120000},
		Run:      swiftlint.Run,
	})
}
