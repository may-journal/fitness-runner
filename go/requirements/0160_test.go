package requirements

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// engineOnly0160 reports whether dep is, or lies under, an engine package no
// check may depend on, wherever its folder sits.
func engineOnly0160(dep string) bool {
	return slices.ContainsFunc(strings.Split(dep, "/"), func(element string) bool {
		return slices.Contains([]string{"metadata", "aftereffect", "dupe", "report"}, element)
	})
}

func Test0160_1(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-test", "-f", `{{.ImportPath}}: {{join .Deps " "}}`, "./cmd/fitness-check-...")
	cmd.Dir = ".."
	out, err := cmd.Output()
	mustDo(t, err)
	var found bool
	for _, line := range strings.Split(string(out), "\n") {
		check, deps, _ := strings.Cut(line, ": ")
		found = found || strings.HasSuffix(check, "/cmd/fitness-check-local-branch")
		for _, dep := range slices.DeleteFunc(strings.Fields(deps), func(dep string) bool { return !engineOnly0160(dep) }) {
			t.Errorf("check package %s depends on engine package %s", check, dep)
		}
	}
	if !found {
		t.Fatalf("go list listed no check package named fitness-check-local-branch, so the test cannot prove the checks' imports:\n%s", out)
	}
}

func Test0160_2(t *testing.T) {
	t.Parallel()
	list := exec.Command("go", "list", "./pkg/...")
	list.Dir = ".."
	out, err := list.Output()
	mustDo(t, err)
	var imports strings.Builder
	for _, p := range strings.Fields(string(out)) {
		fmt.Fprintf(&imports, "\t_ %q\n", p)
	}
	mod := outsideModule(t, "example.com/outside", "package main\n\nimport (\n"+imports.String()+")\n\nfunc main() {}\n")
	if out, err := goOutside(mod, "build", "./..."); err != nil {
		t.Errorf("a module outside this repo cannot import go/pkg: %v\n%s", err, out)
	}
}
