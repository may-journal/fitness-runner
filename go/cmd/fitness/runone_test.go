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

func Test0001_1(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	c := fakeCheck(t, "echo 'no verdict here'\nexit 0\n", 0)
	rows, success, failure, files := summarize([]resolved{c}, []outcome{runOne(t.TempDir(), c, nil, nil)})
	if failure != 1 || rows[0].Ok {
		t.Fatalf("a check with no verdict must fail its row, got %+v", rows[0])
	}
	if code := finish(rows, success, failure, files, 0); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func Test0002_1(t *testing.T) {
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

func Test0002_2(t *testing.T) {
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
