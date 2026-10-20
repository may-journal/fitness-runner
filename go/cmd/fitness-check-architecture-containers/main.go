// Command fitness-check-architecture-containers matches the containers view
// against the runtime products the repository builds and deploys.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-containers"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if err != nil || len(model.Files) == 0 {
		return checkkit.Pass(0), err
	}
	c := model.Containers()
	if len(c.Errors) > 0 {
		return checkkit.Fail(c.Files, c.Errors...), nil
	}
	return checkkit.Pass(c.Files), nil
}
