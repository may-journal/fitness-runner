// Command fitness-check-markdown-filename-kebab-case validates that every
// markdown basename under the repo is kebab-case, standard root docs exempt —
// the Go port of the markdown-filename-kebab-case flavor of the TypeScript
// markdown-filename-convention package. The validation core is shared with
// the camelCase flavor in internal/mdfilename.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdfilename"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-filename-kebab-case"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return mdfilename.Run(root, mdfilename.Kebab), nil
}
