package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runnersScript runs the checked-in runner setup script with args.
func runnersScript(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join("..", "..", "scripts", "fitness-runners.sh")}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil && cmd.ProcessState == nil {
		t.Fatal(err)
	}
	return string(out), cmd.ProcessState.ExitCode()
}

func Test0157_1(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	out, code := runnersScript(t, "--dry-run", "--count", "3", "--user", "ci-user", "--out", dir)
	sees(t, out, code, 0, "runner-3")
	for _, n := range []string{"1", "2", "3"} {
		plist, err := os.ReadFile(filepath.Join(dir, "com.vanderbot.fitness-runner."+n+".plist"))
		mustDo(t, err)
		sees(t, string(plist), 0, 0,
			"<key>UserName</key>\n\t<string>ci-user</string>",
			"<key>KeepAlive</key>\n\t<true/>",
			"/Users/ci-user/runner-"+n+"/run.sh")
	}
	entries, err := os.ReadDir(dir)
	mustDo(t, err)
	if len(entries) != 3 {
		t.Errorf("a dry run must write only the three launchd jobs, got %d files", len(entries))
	}
}

func Test0157_2(t *testing.T) {
	t.Parallel()
	out, code := runnersScript(t, "--dry-run", "--count", "0", "--out", t.TempDir())
	sees(t, out, code, 1, "--count must be a whole number of at least 1, got '0'")
}
