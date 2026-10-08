package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// universalChecks apply to every repo, so they are exempt from the
// passes-clean-where-it-does-not-apply contract. Each needs a reason.
var universalChecks = map[string]string{
	"semantic-commit": "every repo has a commit message to judge",
	"changelog":       "every repo keeps a CHANGELOG.md",
	"plan-trailer":    "every repo has a commit message; it passes but counts the message",
}

// buildChecks compiles every check binary into a temp dir once per test run.
func buildChecks(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("go", "build", "-o", dir, "github.com/may-journal/fitness-runner/go/cmd/...")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building checks: %v\n%s", err, out)
	}
	return dir
}

// contractEnv is the environment a contract run sees: no runner context and
// no git hook variables, so each check judges only the fixture.
func contractEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "FITNESS_") && !strings.HasPrefix(kv, "GIT_") {
			env = append(env, kv)
		}
	}
	return env
}

// runCheck runs one check binary against root and decodes its Result.
func runCheck(t *testing.T, bin, root string) checkkit.Result {
	t.Helper()
	cmd := exec.Command(bin, "--root", root)
	cmd.Dir, cmd.Env = root, contractEnv()
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if err != nil && !(errors.As(err, &exitErr) && exitErr.ExitCode() == 1) {
		t.Fatalf("%s crashed: %v", filepath.Base(bin), err)
	}
	var res checkkit.Result
	if decodeErr := json.Unmarshal(lastJSONLine(out), &res); decodeErr != nil {
		t.Fatalf("%s printed no Result JSON: %v\n%s", filepath.Base(bin), decodeErr, out)
	}
	return res
}

// Test0003_1 enforces what every check owes the runner: its
// --describe name matches its binary, and in a repo it does not apply to it
// passes clean with zero files. That second rule is what lets every repo run
// every check.
func Test0003_1(t *testing.T) {
	for name := range universalChecks {
		if !slices.Contains(allChecks, name) {
			t.Fatalf("universalChecks names %s, which is not in allChecks", name)
		}
	}
	bins := buildChecks(t)
	empty := t.TempDir()
	for _, name := range allChecks {
		contractCase(t, bins, empty, name)
	}
}

// contractCase holds one check to the contract in its own t.Run.
func contractCase(t *testing.T, bins, empty, name string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		bin := filepath.Join(bins, "fitness-check-"+name)
		if got := describe(bin).Name; got != name {
			t.Errorf("--describe name = %q, want %q", got, name)
		}
		if _, universal := universalChecks[name]; universal {
			return
		}
		if res := runCheck(t, bin, empty); !res.Ok || res.FilesChecked != 0 {
			t.Errorf("in a repo it does not apply to: %+v; want a clean pass with 0 files", res)
		}
	})
}
