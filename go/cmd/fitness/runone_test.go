package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/render"
)

// fakeCheck writes a shell script standing in for a check binary.
func fakeCheck(t *testing.T, script string, timeoutMs int) resolved {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fitness-check-fake")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return resolved{name: "fake", bin: bin, desc: checkkit.Describe{Name: "fake", TimeoutMs: timeoutMs}}
}

// runFake runs one fake check through the runner and returns its row and
// the run's exit code.
func runFake(t *testing.T, script string) (render.Row, int) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	c := fakeCheck(t, script, 0)
	rows, success, failure, files := summarize([]resolved{c}, []outcome{runOne(t.TempDir(), c, nil, nil)})
	return rows[0], finish(rows, success, failure, files, 0)
}

func Test0001_1(t *testing.T) {
	row, code := runFake(t, `echo '{"ok":true,"errors":[],"filesChecked":2}'`+"\n")
	if !row.Ok || row.FilesChecked != 2 || code != 0 {
		t.Fatalf("a passing check must report green and exit 0, got %+v, exit %d", row, code)
	}
}

func Test0001_2(t *testing.T) {
	row, code := runFake(t, `echo '{"ok":false,"errors":["a.go:3:1: bad"],"filesChecked":1}'`+"\n")
	if row.Ok || len(row.Errors) != 1 || row.Errors[0] != "a.go:3:1: bad" || code != 1 {
		t.Fatalf("a failing check must report red with its errors and exit 1, got %+v, exit %d", row, code)
	}
}

func Test0001_3(t *testing.T) {
	row, code := runFake(t, "echo 'no verdict here'\nexit 0\n")
	if row.Ok || code != 1 {
		t.Fatalf("a check with no verdict must report red and exit 1, got %+v, exit %d", row, code)
	}
}

func Test0001_4(t *testing.T) {
	c := fakeCheck(t, "exec /bin/sleep 30\n", 200)
	start := time.Now()
	o := runOne(t.TempDir(), c, nil, nil)
	if !o.timedOut || time.Since(start) > 5*time.Second {
		t.Fatalf("want a prompt timeout, got %+v after %s", o, time.Since(start))
	}
	if row := rowFor(c, o); row.Ok || !strings.Contains(row.Errors[0], "timed out") {
		t.Fatalf("want a timed-out failure row, got %+v", row)
	}
}

func Test0002_1(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	c := fakeCheck(t, "/bin/sleep 30 >/dev/null 2>&1 &\necho $! > \"$CHILD_PID\"\nwait\n", 200)
	runOne(t.TempDir(), c, nil, []string{"CHILD_PID=" + pidFile})
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if !waitGone(pid, 3*time.Second) {
		t.Fatalf("child process %d outlived its hung check", pid)
	}
}

// waitGone polls until pid no longer exists or the deadline passes.
func waitGone(pid int, within time.Duration) bool {
	for deadline := time.Now().Add(within); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
	}
	return false
}

func Test0002_2(t *testing.T) {
	own := resolved{name: "registry", desc: checkkit.Describe{TimeoutMs: 9000}}
	plain := resolved{name: "plain"}
	got := withBudgets(&conf.Config{TimeoutMs: 300}, []resolved{own, plain})
	if timeoutFor(got[0]) != 9*time.Second || timeoutFor(got[1]) != 300*time.Millisecond {
		t.Fatalf("configured budget must replace only the default, got %v and %v", timeoutFor(got[0]), timeoutFor(got[1]))
	}
	if timeoutFor(withBudgets(nil, []resolved{plain})[0]) != defaultTimeout {
		t.Fatal("without config a check keeps the default budget")
	}
}
