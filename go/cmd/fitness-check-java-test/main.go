// Command fitness-check-java-test runs Java tests once per build root and
// judges their JUnit XML results and JaCoCo coverage through the Java
// language module, which owns the build run and the judgment.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/tests"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-test", Scope: checkkit.ScopeFiles, TimeoutMs: 900000},
		Run:      tests.Run,
	})
}
