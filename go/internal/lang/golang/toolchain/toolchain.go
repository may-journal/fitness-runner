// Package toolchain finds a repo's Go modules and runs the go toolchain in them
// for the go-vet, go-test, and gofmt checks. Each check self-gates: a repo
// with no go.mod passes clean, and a missing toolchain reports info with an
// install hint.
package toolchain

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// NotInstalled is the info a check reports when Go is missing from PATH.
const NotInstalled = "Go not installed: see https://go.dev/dl"

// Missing is the internal error for a Go toolchain missing from PATH: the
// check could not judge, so it reports info with the install hint.
func Missing() error { return checkkit.Internal(errors.New(NotInstalled)) }

// Modules returns the slash-separated directories, relative to root, that
// hold a go.mod — "." for the root itself. Modules under testdata or vendor
// are fixtures, not the repo's own code, and are left out.
func Modules(root string) []string {
	var mods []string
	for _, f := range walkfs.FilesByExt(root, golang.Manifest) {
		if path.Base(f) == golang.Manifest && !Fixture(f) {
			mods = append(mods, path.Dir(f))
		}
	}
	return mods
}

// Applies reports whether the repo at root holds a Go module of its own.
func Applies(root string) bool { return len(Modules(root)) > 0 }

// Unchanged reports whether a scoped run changes no Go file (.go, go.mod, or
// go.sum), so go-test and go-vet, which work on whole packages, can skip it.
// An unscoped run is never unchanged.
func Unchanged() bool {
	changed := checkkit.ChangedFiles()
	if changed == nil {
		return false
	}
	for _, f := range changed {
		if isGoFile(f) {
			return false
		}
	}
	return true
}

// isGoFile reports whether a path is Go source or module metadata.
func isGoFile(p string) bool {
	base := path.Base(p)
	return strings.HasSuffix(base, golang.Ext) || base == golang.Manifest || base == golang.Sum
}

// ScopedModules returns the repo's modules for go-test and go-vet, or none
// when a scoped run changed no Go file.
func ScopedModules(root string) []string {
	if Unchanged() {
		return nil
	}
	return Modules(root)
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

// Exec runs bin with args in root/dir and returns its exit code, combined
// output, and envErr, a checkkit.Internal error when the environment rather
// than the code failed the run: bin could not start, a signal ended it, or a
// failing run printed sandbox text such as `dial tcp`. A run that fails
// without an exit code maps to 1.
func Exec(root, dir, bin string, args ...string) (code int, out string, envErr error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(root, filepath.FromSlash(dir))
	cmd.Env = CleanEnv(os.Environ())
	o := checkkit.ExecCombined(cmd)
	name := "go " + strings.Join(args, " ")
	if err := o.EnvErr(name); err != nil {
		return o.Code, o.Stdout, err
	}
	return o.Code, o.Stdout, SandboxErr(name, o.Code, o.Stdout)
}

// SandboxErr is the internal error for a failing go run whose output shows
// the sandbox stopped it; nil for exit 0 or output without sandbox text.
func SandboxErr(name string, code int, out string) error {
	if code == 0 {
		return nil
	}
	if phrase := checkkit.SandboxText(out); phrase != "" {
		return checkkit.Internal(fmt.Errorf("%s hit the environment (%s)", name, phrase))
	}
	return nil
}

// hookVars are the variables git exports to hooks, pointing at the repo being
// committed. A test that builds throwaway repos with plain `git` commands
// would inherit them and write into that repo instead.
var hookVars = map[string]bool{
	"GIT_DIR": true, "GIT_INDEX_FILE": true, "GIT_WORK_TREE": true,
	"GIT_COMMON_DIR": true, "GIT_OBJECT_DIRECTORY": true, "GIT_PREFIX": true,
}

// CleanEnv returns env without git's hook variables, so a toolchain run from
// a git hook cannot touch the repository through them, and without the
// runner's FITNESS_* context, so a repo's own tests never inherit a scope.
func CleanEnv(env []string) []string {
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !hookVars[name] && !strings.HasPrefix(name, "FITNESS_") {
			out = append(out, kv)
		}
	}
	return out
}

// EachModule judges every module dir in mods with judge and returns the
// check's Result: failed with every module's findings, else passed,
// counting the modules. The first internal error ends the run, since the
// check could not judge.
func EachModule(mods []string, judge func(dir string) ([]string, error)) (checkkit.Result, error) {
	var errs []string
	for _, dir := range mods {
		found, err := judge(dir)
		if err != nil {
			return checkkit.Result{}, err
		}
		errs = append(errs, found...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(mods), errs...), nil
	}
	return checkkit.Pass(len(mods)), nil
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
