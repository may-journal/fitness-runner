package check

import (
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-vet", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsJudged, TimeoutMs: 300000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	mods := toolchain.ScopedModules(root)
	if len(mods) == 0 {
		return checkkit.Pass(0), nil
	}
	bin, err := exec.LookPath("go")
	if err != nil {
		return checkkit.Result{}, toolchain.Missing()
	}
	return toolchain.EachModule(mods, func(dir string) ([]string, error) {
		return vet(root, dir, bin)
	})
}

// vet runs go vet in one module and returns its findings; package header
// lines ("# pkg") are dropped since each finding names its file. A run that
// could not start, was killed, or failed without naming a file is an
// internal error, since go vet judged nothing.
func vet(root, dir, bin string) ([]string, error) {
	code, out, envErr := toolchain.Exec(root, dir, bin, "vet", "./...")
	if envErr != nil || code == 0 {
		return nil, envErr
	}
	errs, named := findings(dir, out)
	if !named {
		return nil, checkkit.NoVerdict(toolchain.Label(dir, "go vet ./..."), code, out)
	}
	return errs, nil
}

// findings returns go vet's output lines but the package headers, each
// labeled with its module, and whether any names a .go file.
func findings(dir, out string) (errs []string, named bool) {
	for _, l := range toolchain.Lines(out) {
		if !strings.HasPrefix(l, "#") {
			errs = append(errs, toolchain.Label(dir, l))
			named = named || strings.Contains(l, ".go:")
		}
	}
	return errs, named
}
