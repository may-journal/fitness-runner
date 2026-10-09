package requirements

import (
	"os/exec"
	"strings"
	"testing"
)

// readRepoFirst runs `fitness-install -- <args>` on happyRepo; with no args
// it runs read-repo-first alone.
func readRepoFirst(t *testing.T, args ...string) (string, int) {
	t.Helper()
	if len(args) == 0 {
		args = []string{"read-repo-first"}
	}
	return fitness(t, example(t, "happyRepo", nil), nil, args...)
}

// readRepoFirstChosen runs read-repo-first beside the checks a user chose.
func readRepoFirstChosen(t *testing.T, checks string) (string, int) {
	t.Helper()
	return readRepoFirst(t, "--policy=external", "--checks=read-repo-first,"+checks)
}

func Test0053_1(t *testing.T) {
	out, code := readRepoFirst(t)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "read-repo-first")
}

func Test0053_2(t *testing.T) {
	out, code := readRepoFirst(t)
	sees(t, out, code, 0, "Did you familiarize yourself with the decisions logged in the repo, "+
		`specifically all "Fitness Checks" that are enabled via @mayjournal/fitness?`)
}

func Test0053_3(t *testing.T) {
	out, code := readRepoFirst(t)
	sees(t, out, code, 0, "NOTE: Do not under any circumstance use `--no-verify`")
}

func Test0053_4(t *testing.T) {
	out, code := readRepoFirstChosen(t, "jscpd")
	sees(t, out, code, 0, "Check │ Src", "jscpd │ go/cmd/fitness-check-jscpd/README.md")
	if strings.Contains(out, "fitness-check-gofmt") {
		t.Errorf("table lists gofmt, which this run did not enable:\n%s", out)
	}
}

func Test0053_5(t *testing.T) {
	out, code := readRepoFirstChosen(t, "markdown-filename-kebab-case")
	sees(t, out, code, 0, "markdown-filename-kebab-c… │ go/cmd/fitness-check-markdown-file"+"n…")
}

// readRepoFirstRaw runs read-repo-first on happyRepo with env added and
// returns its banner, up to the note, exactly as printed with color codes.
func readRepoFirstRaw(t *testing.T, env ...string) string {
	t.Helper()
	cmd := exec.Command(installer(t), "--", "read-repo-first")
	cmd.Dir, cmd.Env = example(t, "happyRepo", nil), append(userEnv(), env...)
	out, err := cmd.CombinedOutput()
	mustDo(t, err)
	banner, _, _ := strings.Cut(string(out), "NOTE:")
	return banner
}

func Test0053_6(t *testing.T) {
	colored := readRepoFirstRaw(t, "NO_COLOR=")
	plain := readRepoFirstRaw(t, "NO_COLOR=1")
	if !ansi.MatchString(colored) {
		t.Errorf("without NO_COLOR the table should be colored:\n%q", colored)
	}
	if ansi.MatchString(plain) || !strings.Contains(plain, "│ Check") {
		t.Errorf("with NO_COLOR the table should print plain:\n%q", plain)
	}
}
