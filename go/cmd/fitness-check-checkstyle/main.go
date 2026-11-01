package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/checkstyle"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "checkstyle", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsJudged, TimeoutMs: 300000},
		Run:      checkstyle.Run,
	})
}
