// Command fitness-check-java-version checks a Java repo pins its JDK and
// that the machine's JDKs satisfy the pin, through the Java language
// module. Toolchain JDKs come from a local lookup of installed JDKs, never
// from Gradle.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/version"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-version", Scope: checkkit.ScopeFiles},
		Run:      version.Run,
	})
}
