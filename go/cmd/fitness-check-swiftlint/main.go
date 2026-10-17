// Command fitness-check-swiftlint runs SwiftLint over the repo through the
// Swift language module, which owns the tool run and its judgment.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift/swiftlint"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "swiftlint", TimeoutMs: 120000},
		Run:      swiftlint.Run,
	})
}
