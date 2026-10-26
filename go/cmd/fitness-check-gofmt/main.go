package check

import (
	"os/exec"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// batch caps the files passed to one gofmt run, keeping argv well under the
// OS limit.
const batch = 200

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "gofmt", Scope: checkkit.ScopeFiles, TimeoutMs: 60000},
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
	var found []checkkit.Finding
	for start := 0; start < len(files); start += batch {
		end := min(start+batch, len(files))
		found = append(found, unformatted(root, bin, files[start:end])...)
	}
	return checkkit.Found(len(files), errorLine, found...), nil
}

// rule names what gofmt enforces: Go files stay as gofmt writes them.
const rule = "gofmt-formatted"

// message says what is wrong with a file gofmt would change.
const message = "not gofmt-formatted"

// fixCommand is the command that formats a file, given its path.
const fixCommand = "gofmt -w "

// errorLine renders a finding as `path: not gofmt-formatted (run: gofmt -w
// path)`.
func errorLine(f checkkit.Finding) string {
	return f.File + ": " + f.Message + " (run: " + f.Fix + ")"
}

// goFiles returns the repo's own in-scope .go files, leaving out fixtures.
func goFiles(root string) []string {
	var files []string
	for _, f := range walkfs.InScope(walkfs.FilesByExt(root, golang.Ext)) {
		if !toolchain.Fixture(f) {
			files = append(files, f)
		}
	}
	return files
}

// unformatted runs gofmt -l over files and turns each listed path into a
// finding.
func unformatted(root, bin string, files []string) []checkkit.Finding {
	_, out := toolchain.Run(root, ".", bin, append([]string{"-l"}, files...)...)
	var found []checkkit.Finding
	for _, l := range toolchain.Lines(out) {
		found = append(found, checkkit.Finding{File: l, Rule: rule, Message: message, Fix: fixCommand + l})
	}
	return found
}
