package check

import (
	"github.com/may-journal/fitness-runner/go/internal/mdfilename"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-filename-camel-case", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return mdfilename.Run(root, mdfilename.Camel), nil
}
