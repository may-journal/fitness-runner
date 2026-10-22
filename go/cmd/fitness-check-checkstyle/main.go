// Command fitness-check-checkstyle lints Java with the Checkstyle plugin
// through the Java language module, which owns the build run, the
// complexity ceiling rule and the judgment.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/checkstyle"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "checkstyle", Scope: checkkit.ScopeFiles, TimeoutMs: 300000},
		Run:      checkstyle.Run,
	})
}
