// Command fitness-check-java-version checks a Java repo pins its JDK and
// that the machine's JDKs satisfy the pin, through the Java language
// module. Toolchain JDKs come from a local lookup of installed JDKs, never
// from Gradle.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/version"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "java-version"},
		Run:      version.Run,
	})
}
