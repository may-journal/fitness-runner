package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func Test0001_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goRepo())
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, "All 47 checks passed")
}

func Test0001_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "README.md: disallowed **bold**", "46 of 47 checks passed, 1 failed")
}

func Test0001_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(goRepo(), map[string]string{"broken_test.go": "package adder\n\nfunc Test0001_2(t *testing.T {\n"}))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "check failed (exit 2)")
	if !strings.Contains(row(out, "requirements"), "failed") {
		t.Errorf("requirements must report red, got row %q", row(out, "requirements"))
	}
}

func Test0001_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"timeoutMs": 1}` + "\n"})
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "Check timed out after 0.001s")
}

func Test0001_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	for _, bin := range bundleChecks(t, installOnly(t)) {
		if row(out, strings.TrimPrefix(bin, "fitness-check-")) == "" && bin != "fitness-check-go-test-coverage" {
			t.Errorf("%s is installed but fitness never ran it", bin)
		}
	}
}

func Test0001_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", hungTool("300", ""))
	fitness(t, repo, nil)
	childStopped(t, repo)
}

func Test0001_9(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", hungTool("300", "trap '' TERM\n"))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "Check timed out after 0.3s")
	childStopped(t, repo)
}

// childStopped asserts the hung prettier's child process is gone.
func childStopped(t *testing.T, repo string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repo, "prettier.pid"))
	mustDo(t, err)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if !gone(pid, 3*time.Second) {
		t.Fatalf("prettier's child process %d is still running after fitness stopped it", pid)
	}
}

func Test0001_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", hungTool("300", ""))
	out, _ := fitness(t, repo, nil)
	if !strings.Contains(out, "Check timed out after 0.3s") || strings.Contains(row(out, "go-test"), "failed") {
		t.Errorf("prettier must stop at the configured 0.3s while checks with their own budget keep it:\n%s", out)
	}
}

// readme is happyRepo's README, for Givens that add to it.
const readme = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# App\n\nShort and clean.\n"

// bundleChecks lists the check programs the installer put in dir.
func bundleChecks(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	mustDo(t, err)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "fitness-check-") {
			names = append(names, e.Name())
		}
	}
	return names
}

// installOnly runs `fitness-install --install-only`, which prints the
// directory holding the runner and every check, and returns that directory.
func installOnly(t *testing.T) string {
	t.Helper()
	out, err := exec.Command(installer(t), "--install-only").Output()
	mustDo(t, err)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// gone polls until pid no longer exists or the deadline passes.
func gone(pid int, within time.Duration) bool {
	for deadline := time.Now().Add(within); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
	}
	return false
}
