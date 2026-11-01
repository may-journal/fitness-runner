package check

import (
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-test", Scope: checkkit.ScopeFiles, TimeoutMs: 900000},
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
		return test(root, dir, bin)
	})
}

// test runs go test in one module and returns the lines that explain a
// failure. A run that could not start, was killed, or explained no failure is
// an internal error, since no test judged the code.
func test(root, dir, bin string) ([]string, error) {
	code, out, envErr := toolchain.Exec(root, dir, bin, "test", "./...")
	if envErr != nil || code == 0 {
		return nil, envErr
	}
	errs := failureLines(dir, out)
	if len(errs) == 0 {
		return nil, checkkit.NoVerdict(toolchain.Label(dir, "go test ./..."), code, out)
	}
	return errs, nil
}

// failureLines returns the go test output lines that explain a failure,
// each labeled with its module.
func failureLines(dir, out string) []string {
	var errs []string
	for _, l := range toolchain.Lines(out) {
		if failureLine(l) {
			errs = append(errs, toolchain.Label(dir, strings.TrimSpace(l)))
		}
	}
	return errs
}

// failureLine reports whether a go test output line explains a failure, such
// as a failing test, panic, or build error.
func failureLine(l string) bool {
	t := strings.TrimSpace(l)
	for _, p := range []string{"--- FAIL", "FAIL", "panic:"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return strings.Contains(t, ".go:")
}
