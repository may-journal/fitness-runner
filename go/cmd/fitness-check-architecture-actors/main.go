// Command fitness-check-architecture-actors requires each requirement's
// Given to start with an actor the system context declares, so no role
// slips into a requirement that the architecture does not know.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-actors"},
		Run:      run,
	})
}

// run reads the whole model and every requirement doc on each run, scoped
// or not, so a removed actor fails even when no requirement changed. Until
// the repo opts in, a failure only the staged rule finds is a warning.
func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if err != nil || len(model.Files) == 0 {
		return checkkit.Pass(0), err
	}
	docs := len(model.Requirements)
	result := checkkit.Pass(docs)
	if errs := model.CheckGivens(); len(errs) > 0 {
		result = checkkit.Fail(docs, errs...)
	}
	staged := model.StagedGivens()
	result.Warn(staged, conf.StagedNote(len(staged)))
	return result, nil
}
