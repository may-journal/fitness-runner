package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// goVersion reads the go line of go/go.mod.
func goVersion(t *testing.T) string {
	t.Helper()
	text, err := os.ReadFile("../go.mod")
	mustDo(t, err)
	for _, line := range strings.Split(string(text), "\n") {
		if version, ok := strings.CutPrefix(line, "go "); ok && len(strings.Fields(version)) > 0 {
			return strings.Fields(version)[0]
		}
	}
	t.Fatal("go/go.mod has no go line")
	return ""
}

// outsideModule writes a Go module outside any workspace and returns its
// folder. Its main.go holds main, and its go.mod takes this repo's go module
// from the working tree.
func outsideModule(t *testing.T, module, main string) string {
	t.Helper()
	here, err := filepath.Abs("..")
	mustDo(t, err)
	mod := t.TempDir()
	write(t, mod, map[string]string{
		"go.mod":  "module " + module + "\n\ngo " + goVersion(t) + "\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\nreplace github.com/may-journal/fitness-runner/go => " + strconv.Quote(here) + "\n",
		"main.go": main,
	})
	return mod
}

// goOutside runs go with args in the outside module mod, as a user's terminal
// would with no workspace. A failure stops the test, saying what failed and
// quoting go's output.
func goOutside(t *testing.T, mod, what string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir, cmd.Env = mod, append(userEnv(), "GOFLAGS=-mod=mod", "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", what, err, out)
	}
}
