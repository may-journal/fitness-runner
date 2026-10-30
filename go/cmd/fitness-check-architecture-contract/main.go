package check

import (
	"github.com/may-journal/fitness-runner/go/internal/architecture"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "architecture-contract", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	model, err := architecture.Load(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(model.Files) == 0 {
		return noViews(root), nil
	}
	if len(model.Errors) > 0 {
		return checkkit.Fail(len(model.Files), model.Errors...), nil
	}
	return checkkit.Pass(len(model.Files)), nil
}

// noViews passes a repo without product code and fails one with it, since
// its code then has no architecture. A scoped run whose changes cannot move
// the views passes with zero files, as it judges none.
func noViews(root string) checkkit.Result {
	if architecture.Unmoved(root) {
		return checkkit.Pass(0)
	}
	code := architecture.ProductCode(root)
	if len(code) == 0 {
		return checkkit.Pass(0)
	}
	return checkkit.Fail(len(code), "docs/architecture/ has no views, but product code such as "+code[0]+" needs them; add the five views to docs/architecture/ or architecture/: "+
		"actors (01-system-context.md), containers (02-containers.md), components (03-components.md), "+
		"code (04-code.md), and journey (05-user-journey.md)")
}
