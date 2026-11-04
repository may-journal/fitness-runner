package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func Test0144_1(t *testing.T) {
	// Not parallel: it keeps every CPU busy, which would slow the tests beside it.
	defer busyCPUs()()
	for _, prelude := range []string{"", "trap '' TERM\n"} {
		repo := example(t, "happyRepo", hungTool("2000", prelude))
		path := filepath.Join(t.TempDir(), "fitness.json")
		out, code := fitness(t, repo, nil, "--policy=external", "--checks=prettier", "--json="+path)
		sees(t, out, code, 1, "Check timed out after 2s")
		raw, err := os.ReadFile(path)
		mustDo(t, err)
		var report perfReport
		mustDo(t, json.Unmarshal(raw, &report))
		past := time.Duration(reported(t, report, "prettier").Ms)*time.Millisecond - 2*time.Second
		if past > time.Second {
			t.Errorf("with every CPU busy, prettier ended %s past its 2s budget, want within 1s", past)
		}
		if !stopped(t, filepath.Join(repo, "prettier.pid"), max(time.Second-past, 0)) {
			t.Errorf("prettier never started within its 2s budget, so nothing shows its child stops")
		}
	}
}

// busyCPUs spins one goroutine per CPU until the returned func stops them.
func busyCPUs() func() {
	done := make(chan struct{})
	for range runtime.NumCPU() {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}
	return func() { close(done) }
}
