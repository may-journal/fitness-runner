// Command fitness-check-localized-text fails SwiftUI text and requirement
// test literals that skip the app's String Catalog, through the Swift
// language module, which owns the reading and its judgment.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift/localized"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: localized.Name, Scope: checkkit.ScopeFiles},
		Run:      localized.Run,
	})
}
