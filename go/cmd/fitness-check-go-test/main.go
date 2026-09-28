// Command fitness-check-go-test runs `go test ./...` in every Go module of
// the repo and reports the failing tests. A repo with no go.mod passes clean,
// so the check is safe in the default list for every repo.
package main

import (
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-test", TimeoutMs: 900000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	mods := toolchain.Modules(root)
	if len(mods) == 0 {
		return checkkit.Pass(0), nil
	}
	bin, err := exec.LookPath("go")
	if err != nil {
		return checkkit.Fail(0, toolchain.NotInstalled), nil
	}
	var errs []string
	for _, dir := range mods {
		errs = append(errs, test(root, dir, bin)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(mods), errs...), nil
	}
	return checkkit.Pass(len(mods)), nil
}

// test runs go test in one module and returns the lines that explain a
// failure, falling back to a run hint when none can be picked out.
func test(root, dir, bin string) []string {
	code, out := toolchain.Run(root, dir, bin, "test", "./...")
	if code == 0 {
		return nil
	}
	var errs []string
	for _, l := range toolchain.Lines(out) {
		if failureLine(l) {
			errs = append(errs, toolchain.Label(dir, strings.TrimSpace(l)))
		}
	}
	if len(errs) == 0 {
		errs = []string{toolchain.Label(dir, "go test failed (run: go test ./...)")}
	}
	return errs
}

// failureLine reports whether a go test output line explains a failure: a
// failing test or package, a panic, a build error, or a test's own message.
func failureLine(l string) bool {
	t := strings.TrimSpace(l)
	for _, p := range []string{"--- FAIL", "FAIL", "panic:"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return strings.Contains(t, ".go:")
}
