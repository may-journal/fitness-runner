// Command fitness-check-java-version checks a Java repo pins its JDK and
// that the running JDK satisfies the pin, through the Java language module.
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
