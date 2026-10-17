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
	repo := example(t, "happyRepo", architectedRepo())
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, "All 58 checks passed")
}

func Test0001_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "README.md: disallowed **bold**", "57 of 58 checks passed, 1 failed")
}

func Test0001_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(goRepo(), map[string]string{"broken_test.go": "package adder\n\nfunc Test0001_2(t *testing.T {\n"}))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "[requirements] Info:", "missing ',' in parameter list", "go-vet")
	if !strings.Contains(row(out, "requirements"), "info") || !strings.Contains(row(out, "go-vet"), "failed") {
		t.Errorf("requirements must show info while go-vet sets the exit code, got rows %q and %q", row(out, "requirements"), row(out, "go-vet"))
	}
	out, code = describeFails(t, example(t, "happyRepo", nil), "markdown-no-bold-italic")
	sees(t, out, code, 0, "--describe gave no metadata, so the check ran with defaults", "0 of 1 checks passed, 1 info")
	if !strings.Contains(row(out, "markdown-no-bold-italic"), "info") {
		t.Errorf("a check whose describe fails must still run and show info, got row %q", row(out, "markdown-no-bold-italic"))
	}
}

// describeFails runs check in repo with a copy of the installed runner, after
// swapping that check for one whose --describe fails and whose run is the
// real one, as a user who replaced one check would. The runner is copied, not
// linked, since Linux resolves a linked runner back to the install folder.
// It returns the output and exit code.
func describeFails(t *testing.T, repo, check string) (string, int) {
	t.Helper()
	bin := "fitness-check-" + check
	installed := installOnly(t)
	bundle := t.TempDir()
	linkTools(bundle, installed, []string{bin, "fitness"})
	runner, err := os.ReadFile(filepath.Join(installed, "fitness"))
	mustDo(t, err)
	write(t, bundle, map[string]string{"fitness": string(runner), bin: "#!/bin/sh\n[ \"$1\" = --describe ] && exit 1\nexec \"" + filepath.Join(installed, bin) + "\" \"$@\"\n"})
	cmd := exec.Command(filepath.Join(bundle, "fitness"), check)
	cmd.Dir, cmd.Env = repo, userEnv()
	raw, _ := cmd.CombinedOutput()
	return ansi.ReplaceAllString(string(raw), ""), cmd.ProcessState.ExitCode()
}

func Test0001_4(t *testing.T) {
	t.Parallel()
	for _, prelude := range []string{"", "trap '' TERM\n"} {
		repo := example(t, "happyRepo", hungTool("300", prelude))
		out, code := fitness(t, repo, nil)
		sees(t, out, code, 1, "Check timed out after 0.3s")
		if !strings.Contains(row(out, "prettier"), "failed") {
			t.Errorf("a timed-out check must report red, got row %q", row(out, "prettier"))
		}
		childStopped(t, repo)
	}
}

func Test0001_11(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(architectedRepo(), map[string]string{"README.md": readme + "\nSome **bold** text.\n"}))
	summary := filepath.Join(t.TempDir(), "summary.md")
	out, code := fitness(t, repo, []string{killedVet(t), "GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary})
	sees(t, out, code, 1, "56 of 58 checks passed, 1 failed, 1 info", "::error file=README.md",
		"::notice title=go-vet::check ended without a result (signal: killed)")
	written, _ := os.ReadFile(summary)
	sees(t, string(written), code, 1, "| go-vet | ℹ️ info |", "### ℹ️ Info")
}

func Test0001_12(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", architectedRepo())
	out, code := fitness(t, repo, []string{killedVet(t)})
	sees(t, out, code, 0, "check ended without a result (signal: killed)", "57 of 58 checks passed, 1 info")
	if !strings.Contains(row(out, "go-vet"), "info") {
		t.Errorf("the check that cannot run must show info, got row %q", row(out, "go-vet"))
	}
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
