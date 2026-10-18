package requirements

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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
	sees(t, out, code, 1, "fitness init: fatal: not in a git directory")
	if _, err := os.Stat(filepath.Join(dir, ".githooks")); err == nil {
		t.Error("fitness init wrote .githooks outside a git repo")
	}
}

func Test0008_6(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, t.TempDir(), nil, "hook", "pre-push")
	sees(t, out, code, 1, "fitness hook: fatal: not a git repository")
}

func Test0008_7(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "hook", "commit-msg")
	sees(t, out, code, 1, "fitness hook commit-msg: missing message file")
}

func Test0008_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	alone := lone(t, filepath.Join(installOnly(t), "fitness"))
	out, code := runStarted(t, alone, repo)
	sees(t, ansi.ReplaceAllString(out, ""), code, 1, "reinstall fitness so its checks sit beside it")
}

// lone places the installed runner by itself in a temp dir. A hard link
// leaves no open write handle for a parallel test's fork to inherit, which
// would make the exec fail with "text file busy".
func lone(t *testing.T, installed string) string {
	t.Helper()
	alone := filepath.Join(t.TempDir(), "fitness")
	if os.Link(installed, alone) == nil {
		return alone
	}
	raw, err := os.ReadFile(installed)
	mustDo(t, err)
	mustDo(t, os.WriteFile(alone, raw, 0o755))
	return alone
}

// runStarted runs bin in repo and fails the test if the process never
// starts, retrying only while the binary is still busy.
func runStarted(t *testing.T, bin, repo string) (string, int) {
	t.Helper()
	for attempt := 0; ; attempt++ {
		cmd := exec.Command(bin, "--policy=external", "--all")
		cmd.Dir, cmd.Env = repo, userEnv()
		out, err := cmd.CombinedOutput()
		if cmd.ProcessState != nil {
			return string(out), cmd.ProcessState.ExitCode()
		}
		if !errors.Is(err, syscall.ETXTBSY) || attempt == 20 {
			t.Fatalf("start %s: %v", bin, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
