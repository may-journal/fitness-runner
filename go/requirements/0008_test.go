package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func Test0008_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "no-such-check")
	sees(t, out, code, 1, "Unknown check: no-such-check")
}

func Test0008_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}), nil)
	sees(t, out, code, 1, "fitness setup: .fitnessrc.json:")
}

func Test0008_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
	sees(t, out, code, 0, "fitness — run the configured checks.", "fitness init")
}

func Test0008_4(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "hook", "no-such-hook")
	sees(t, out, code, 1, "fitness hook: unknown hook no-such-hook")
}

func Test0008_5(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	out, code := fitness(t, dir, nil, "init")
	sees(t, out, code, 1, "fitness init: git config: fatal: not in a git directory")
	if _, err := os.Stat(filepath.Join(dir, ".githooks")); err == nil {
		t.Error("fitness init wrote .githooks outside a git repo")
	}
}

func Test0008_6(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, t.TempDir(), nil, "hook", "pre-push")
	sees(t, out, code, 1, "fitness hook: git rev-parse: fatal: not a git repository")
}

func Test0008_7(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "hook", "commit-msg")
	sees(t, out, code, 1, "fitness hook commit-msg: missing message file")
}

// runAlone runs the runner at alone in repo with args, as a user types it.
func runAlone(t *testing.T, alone, repo string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(alone, args...)
		cmd.Dir, cmd.Env = repo, userEnv()
		return cmd
	})
}

func Test0008_9(t *testing.T) {
	t.Parallel()
	alone := filepath.Join(t.TempDir(), "fitness")
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), alone)
	out, code := runAlone(t, alone, example(t, "happyRepo", architectedRepo()), "--policy=external", "--all")
	sees(t, out, code, 0, allPassed())
	bold := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **strong** words.\n"})
	out, code = runAlone(t, alone, bold, "check", "markdown-no-bold-italic", "--root", bold)
	sees(t, out, code, 1, `"ok":false`, `README.md: disallowed **bold**`+noBoldItalicRule)
	if strings.Contains(out, "Running checks") {
		t.Errorf("fitness check ran the full suite, not one check:\n%s", out)
	}
}
