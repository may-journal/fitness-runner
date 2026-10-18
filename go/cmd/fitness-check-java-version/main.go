// Command fitness-check-java-version checks a Java repo pins its JDK and
// that the machine's JDKs satisfy the pin, through the Java language
// module. Toolchain JDKs come from a local lookup of installed JDKs and
// never from Gradle, so the check gets a one-minute time budget.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/version"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-version", TimeoutMs: 60000},
		Run:      version.Run,
	})
}
