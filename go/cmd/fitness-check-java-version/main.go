package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/version"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-version", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsJudged},
		Run:      version.Run,
	})
}
