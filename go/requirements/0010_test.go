package requirements

import (
	"path/filepath"
	"strings"
	"testing"
)

func Test0010_1(t *testing.T) {
	cases{
		{"a `packages` setting whose globs match package folders, a file and a folder twice", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", rootsCheck(t, `{"packages": ["cmd/*", "cmd/b", "lib"]}`)), nil, "check", "roots-rule")
			sees(t, out, code, 1, "package roots: cmd/a, cmd/b, lib")
		}},
		{"a `packages` glob that matches no folder", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", rootsCheck(t, `{"packages": ["cmd/*", "Packages/*"]}`)), nil, "check", "roots-rule")
			sees(t, out, code, 1, `.fitnessrc.json: packages glob "Packages/*" matches no folder of tracked files`)
			if strings.Contains(out, "package roots:") {
				t.Errorf("a glob matching no folder must stop the run before any check:\n%s", out)
			}
		}},
		{"no `packages` setting, with the variable inherited from the shell", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", rootsCheck(t, "{}")), []string{"FITNESS_PACKAGES=stale"}, "check", "roots-rule")
			sees(t, out, code, 1, "package roots: .")
		}},
	}.run(t)
}

// rootsCheck is a repo with config, folders cmd/a, cmd/b and lib, and a
// file cmd/notes.md. Its Go check roots-rule fails naming the package
// roots checkkit gives it.
func rootsCheck(t *testing.T, config string) map[string]string {
	t.Helper()
	sdk, err := filepath.Abs("..")
	mustDo(t, err)
	return map[string]string{
		".fitnessrc.json": config + "\n",
		"cmd/a/main.go":   "package main\n", "cmd/b/main.go": "package main\n", "lib/lib.go": "package lib\n",
		"cmd/notes.md": "# Notes\n\nThe commands.\n",
		"fitness/go.mod": "module example.com/repo/fitness\n\ngo 1.26\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\n" +
			"replace github.com/may-journal/fitness-runner/go => " + sdk + "\n",
		"fitness/roots-rule/README.md": "# Package roots\n\nNames the package roots it gets.\n",
		"fitness/roots-rule/main.go": "// Package main names the package roots it gets.\npackage main\n\n" +
			"import (\n\t\"strings\"\n\n\t\"github.com/may-journal/fitness-runner/go/check\"\n\t\"github.com/may-journal/fitness-runner/go/pkg/checkkit\"\n)\n\n" +
			"var rule = check.Check{Run: func(*check.Context) (check.Result, error) {\n" +
			"\treturn check.Fail(1, \"package roots: \"+strings.Join(checkkit.PackageRoots(), \", \")), nil\n}}\n\n" +
			"func main() { check.Main(rule) }\n",
	}
}
