// Command fitness-check-go-vet runs `go vet ./...` in every Go module of the
// repo. A repo with no go.mod passes clean, so the check is safe in the
// default list for every repo.
package main

import (
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-vet", TimeoutMs: 300000},
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
		return checkkit.Fail(0, toolchain.NotInstalled), nil
	}
	var errs []string
	for _, dir := range mods {
		errs = append(errs, vet(root, dir, bin)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(mods), errs...), nil
	}
	return checkkit.Pass(len(mods)), nil
}

// vet runs go vet in one module and returns its findings; package header
// lines ("# pkg") are dropped since each finding names its file.
func vet(root, dir, bin string) []string {
	code, out := toolchain.Run(root, dir, bin, "vet", "./...")
	if code == 0 {
		return nil
	}
	var errs []string
	for _, l := range toolchain.Lines(out) {
		if !strings.HasPrefix(l, "#") {
			errs = append(errs, toolchain.Label(dir, l))
		}
	}
	if len(errs) == 0 {
		errs = []string{toolchain.Label(dir, "go vet failed (run: go vet ./...)")}
	}
	return errs
}
