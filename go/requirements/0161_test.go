package requirements

import (
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// module0161 starts the import path of every package in this repo's module.
const module0161 = "github.com/may-journal/fitness-runner/go/"

// checkPackage0161 reports whether path is a check's package.
func checkPackage0161(path string) bool {
	return strings.Contains(path, "/cmd/fitness-check-")
}

// goList0161 runs go list with args in go/ and returns its output.
func goList0161(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = ".."
	out, err := cmd.Output()
	mustDo(t, err)
	return string(out)
}

// fileImports0161 returns the import paths of one Go file.
func fileImports0161(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	mustDo(t, err)
	var paths []string
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		mustDo(t, err)
		paths = append(paths, path)
	}
	return paths
}

// engineImports0161 returns the imports of the engine's files apart from
// checks.go, failing each check among them. It also counts the checks that
// checks.go imports.
func engineImports0161(t *testing.T) (imports []string, builtIns int) {
	t.Helper()
	for _, file := range strings.Fields(goList0161(t, "-f", `{{range .GoFiles}}{{$.Dir}}/{{.}}{{"\n"}}{{end}}{{range .IgnoredGoFiles}}{{$.Dir}}/{{.}}{{"\n"}}{{end}}`, "./cmd/fitness")) {
		paths := fileImports0161(t, file)
		if filepath.Base(file) == "checks.go" {
			builtIns = len(slices.DeleteFunc(paths, func(p string) bool { return !checkPackage0161(p) }))
			continue
		}
		for _, path := range slices.DeleteFunc(slices.Clone(paths), func(p string) bool { return !checkPackage0161(p) }) {
			t.Errorf("engine file %s imports check package %s", filepath.Base(file), path)
		}
		// Only this module's packages can reach a check, and go list resolves
		// them on any platform, unlike a package for another OS.
		imports = append(imports, slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(p, module0161) })...)
	}
	return slices.Compact(slices.Sorted(slices.Values(imports))), builtIns
}

func Test0161_1(t *testing.T) {
	t.Parallel()
	imports, builtIns := engineImports0161(t)
	if builtIns == 0 {
		t.Fatal("checks.go links no check package, so the test cannot tell the built-ins map from the engine")
	}
	if len(imports) == 0 {
		return
	}
	out := goList0161(t, append([]string{"-deps", "-f", `{{.ImportPath}}: {{join .Imports " "}}`}, imports...)...)
	for _, line := range strings.Split(out, "\n") {
		pkg, imported, _ := strings.Cut(line, ": ")
		for _, dep := range slices.DeleteFunc(strings.Fields(imported), func(p string) bool { return !checkPackage0161(p) }) {
			t.Errorf("engine package %s imports check package %s", pkg, dep)
		}
	}
}
