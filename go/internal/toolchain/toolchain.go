// Package toolchain finds a repo's Go modules and runs the go toolchain in them
// for the go-vet, go-test, and gofmt checks. Each check self-gates: a repo
// with no go.mod passes clean, and a missing toolchain fails with an install
// hint.
package toolchain

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// NotInstalled is the failure a check reports when Go is missing from PATH.
const NotInstalled = "Go not installed: see https://go.dev/dl"

// Modules returns the slash-separated directories, relative to root, that
// hold a go.mod — "." for the root itself. Modules under testdata or vendor
// are fixtures, not the repo's own code, and are left out.
func Modules(root string) []string {
	var mods []string
	for _, f := range walkfs.FilesByExt(root, "go.mod") {
		if path.Base(f) == "go.mod" && !Fixture(f) {
			mods = append(mods, path.Dir(f))
		}
	}
	return mods
}

// Fixture reports whether a slash-separated path sits under a testdata or
// vendor directory, which the go tool itself skips.
func Fixture(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if seg == "testdata" || seg == "vendor" {
			return true
		}
	}
	return false
}

// Run executes bin with args in root/dir and returns its exit code and
// combined output. A run that fails without an exit code maps to 1.
func Run(root, dir, bin string, args ...string) (int, string) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(root, filepath.FromSlash(dir))
	cmd.Env = CleanEnv(os.Environ())
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err == nil {
		return 0, out.String()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
		return exitErr.ExitCode(), out.String()
	}
	return 1, out.String()
}

// hookVars are the variables git exports to hooks, pointing at the repo being
// committed. A test that builds throwaway repos with plain `git` commands
// would inherit them and write into that repo instead.
var hookVars = map[string]bool{
	"GIT_DIR": true, "GIT_INDEX_FILE": true, "GIT_WORK_TREE": true,
	"GIT_COMMON_DIR": true, "GIT_OBJECT_DIRECTORY": true, "GIT_PREFIX": true,
}

// CleanEnv returns env without git's hook variables, so a toolchain run from
// a git hook cannot touch the repository through them.
func CleanEnv(env []string) []string {
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !hookVars[name] {
			out = append(out, kv)
		}
	}
	return out
}

// Label prefixes a message with its module directory, unless the module is
// the repo root.
func Label(dir, msg string) string {
	if dir == "." {
		return msg
	}
	return fmt.Sprintf("%s: %s", dir, msg)
}

// Lines splits tool output into its non-blank lines.
func Lines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, strings.TrimRight(l, "\r"))
		}
	}
	return lines
}
