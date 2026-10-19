// Command fitness-check-architecture-journey keeps the user journey diagram
// and the requirement docs covering each other: every requirement doc ID
// appears in a journey diagram label, and every journey node that promises
// behavior names one.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-journey"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if errs := model.JourneyErrors(); len(errs) > 0 {
		return checkkit.Fail(model.JourneyFiles(), errs...), err
	}
	return checkkit.Pass(model.JourneyFiles()), err
}
