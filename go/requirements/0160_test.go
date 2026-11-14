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
// check may depend on, wherever its folder sits.
func engineOnly0160(dep string) bool {
	return slices.ContainsFunc(strings.Split(dep, "/"), func(element string) bool {
		return slices.Contains([]string{"metadata", "aftereffect", "dupe", "report"}, element)
	})
}

// goVersion0160 reads the go line of go/go.mod.
func goVersion0160(t *testing.T) string {
	t.Helper()
	text, err := os.ReadFile("../go.mod")
	mustDo(t, err)
	for _, line := range strings.Split(string(text), "\n") {
		if version, ok := strings.CutPrefix(line, "go "); ok {
			return strings.TrimSpace(version)
		}
	}
	t.Fatal("go/go.mod has no go line")
	return ""
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
	write("go.mod", "module example.com/outside\n\ngo "+goVersion0160(t)+"\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\nreplace github.com/may-journal/fitness-runner/go => "+here+"\n")
	write("main.go", "package main\n\nimport (\n"+imports.String()+")\n\nfunc main() {}\n")
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir, cmd.Env = mod, append(userEnv(), "GOFLAGS=-mod=mod", "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("a module outside this repo cannot import go/pkg: %v\n%s", err, out)
	}
}
