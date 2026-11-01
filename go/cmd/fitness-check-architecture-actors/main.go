package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-actors", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// run reads the whole model and every requirement doc unless the run is
// architecture.Unmoved.
// So a removed actor fails even when no requirement changed.
func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if err != nil || len(model.Files) == 0 {
		return checkkit.Pass(0), err
	}
	docs := len(model.Requirements)
	if errs := model.CheckGivens(); len(errs) > 0 {
		return checkkit.Fail(docs, errs...), nil
	}
	return checkkit.Pass(docs), nil
}
