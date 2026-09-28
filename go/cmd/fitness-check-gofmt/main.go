// Command fitness-check-gofmt lists Go files that gofmt would change. It
// checks the repo's own .go files (not testdata or vendor, and not paths the
// config ignores), so a repo with no Go passes clean.
package main

import (
	"os/exec"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// batch caps the files passed to one gofmt run, keeping argv well under the
// OS limit.
const batch = 200

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "gofmt", TimeoutMs: 60000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := goFiles(root)
	if len(files) == 0 {
		return checkkit.Pass(0), nil
	}
	bin, err := exec.LookPath("gofmt")
	if err != nil {
		return checkkit.Fail(0, toolchain.NotInstalled), nil
	}
	var errs []string
	for start := 0; start < len(files); start += batch {
		end := min(start+batch, len(files))
		errs = append(errs, unformatted(root, bin, files[start:end])...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// goFiles returns the repo's own .go files, leaving out fixtures.
func goFiles(root string) []string {
	var files []string
	for _, f := range walkfs.FilesByExt(root, ".go") {
		if !toolchain.Fixture(f) {
			files = append(files, f)
		}
	}
	return files
}

// unformatted runs gofmt -l over files and turns each listed path into a
// finding.
func unformatted(root, bin string, files []string) []string {
	_, out := toolchain.Run(root, ".", bin, append([]string{"-l"}, files...)...)
	var errs []string
	for _, l := range toolchain.Lines(out) {
		errs = append(errs, l+": not gofmt-formatted (run: gofmt -w "+l+")")
	}
	return errs
}
