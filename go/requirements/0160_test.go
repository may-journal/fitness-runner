package requirements

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// engineOnly0160 reports whether dep is, or lies under, an engine package no
// check may depend on.
func engineOnly0160(dep string) bool {
	return slices.ContainsFunc([]string{"metadata", "aftereffect", "dupe", "report"}, func(name string) bool {
		return strings.HasSuffix(dep, "/internal/"+name) || strings.Contains(dep, "/internal/"+name+"/")
	})
}

func Test0160_1(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-deps", "-test", "-f", "{{.ImportPath}}", "./cmd/fitness-check-...")
	cmd.Dir = ".."
	out, err := cmd.Output()
	mustDo(t, err)
	deps := strings.Fields(string(out))
	if !slices.ContainsFunc(deps, func(dep string) bool { return strings.HasSuffix(dep, "/cmd/fitness-check-local-branch") }) {
		t.Fatalf("go list found no checks:\n%s", out)
	}
	for _, dep := range slices.DeleteFunc(deps, func(dep string) bool { return !engineOnly0160(dep) }) {
		t.Errorf("a check depends on %s", dep)
	}
}

func Test0160_2(t *testing.T) {
	t.Parallel()
	here, err := filepath.Abs("..")
	mustDo(t, err)
	list := exec.Command("go", "list", "./pkg/...")
	list.Dir = ".."
	out, err := list.Output()
	mustDo(t, err)
	var imports strings.Builder
	for _, p := range strings.Fields(string(out)) {
		fmt.Fprintf(&imports, "\t_ %q\n", p)
	}
	mod := t.TempDir()
	write := func(name, text string) { mustDo(t, os.WriteFile(filepath.Join(mod, name), []byte(text), 0o644)) }
	write("go.mod", "module example.com/outside\n\ngo 1.26\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\nreplace github.com/may-journal/fitness-runner/go => "+here+"\n")
	write("main.go", "package main\n\nimport (\n"+imports.String()+")\n\nfunc main() {}\n")
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir, cmd.Env = mod, append(userEnv(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("a module outside this repo cannot import go/pkg: %v\n%s", err, out)
	}
}
