package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-code", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// run reads the whole model unless architecture.Unmoved, so a change to only
// code or view still meets the other side.
func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if err != nil || len(model.Files) == 0 {
		return checkkit.Pass(0), err
	}
	c := model.Code()
	if len(c.Errors) > 0 {
		return checkkit.Fail(c.Files, c.Errors...), nil
	}
	return checkkit.Pass(c.Files), nil
}
