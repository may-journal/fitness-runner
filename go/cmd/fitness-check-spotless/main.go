// Command fitness-check-spotless checks Java formatting with the Spotless
// plugin through the Java language module, which owns the build run and its
// judgment.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/spotless"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "spotless", TimeoutMs: 300000},
		Run:      spotless.Run,
	})
}
