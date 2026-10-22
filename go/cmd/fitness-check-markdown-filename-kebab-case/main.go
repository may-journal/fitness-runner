// Command fitness-check-markdown-filename-kebab-case requires every markdown
// file name to be kebab-case, standard root docs exempt. The validation core is shared with
// the camelCase flavor in internal/mdfilename.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdfilename"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-filename-kebab-case", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return mdfilename.Run(root, mdfilename.Kebab), nil
}
