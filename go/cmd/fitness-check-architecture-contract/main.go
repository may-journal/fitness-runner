// Command fitness-check-architecture-contract validates declarations beside
// all five architecture views before more specific checks resolve their proof.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-contract"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if len(model.Errors) > 0 {
		return checkkit.Fail(len(model.Files), model.Errors...), err
	}
	return checkkit.Pass(len(model.Files)), err
}
