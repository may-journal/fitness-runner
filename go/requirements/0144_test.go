package requirements

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func Test0144_1(t *testing.T) {
	// Not parallel: it keeps every CPU busy, which would slow the tests beside it.
	busyCPUs(t)
	t.Run("tool with a child", func(t *testing.T) { stopsUnderLoad(t, "") })
	t.Run("tool ignoring TERM", func(t *testing.T) { stopsUnderLoad(t, "trap '' TERM\n") })
}

// stopsUnderLoad asserts a hung prettier, after prelude, ends within 1s past
// its 2s budget, and no tool runs once fitness exits.
func stopsUnderLoad(t *testing.T, prelude string) {
	t.Helper()
	repo := example(t, "happyRepo", hungTool("2000", prelude))
	path := filepath.Join(t.TempDir(), "fitness.json")
	out, code := fitness(t, repo, nil, "--policy=external", "--checks=prettier", "--json="+path)
	sees(t, out, code, 1, "Check timed out after 2s")
	raw, err := os.ReadFile(path)
	mustDo(t, err)
	var report perfReport
	mustDo(t, json.Unmarshal(raw, &report))
	if past := time.Duration(reported(t, report, "prettier").Ms)*time.Millisecond - 2*time.Second; past > time.Second {
		t.Errorf("with every CPU busy, prettier ended %s past its 2s budget, want within 1s", past)
	}
	if !stopped(t, filepath.Join(repo, "prettier.pid"), 200*time.Millisecond) {
		t.Errorf("prettier never started within its 2s budget, so nothing shows its child stops")
	}
}

// busyCPUs keeps every CPU busy with one spinning shell per CPU until the
// test ends.
func busyCPUs(t *testing.T) {
	t.Helper()
	for range runtime.NumCPU() {
		cmd := exec.Command("/bin/sh", "-c", "while :; do :; done")
		mustDo(t, cmd.Start())
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
	}
}
