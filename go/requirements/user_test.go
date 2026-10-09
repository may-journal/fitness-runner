// Package requirements proves docs/requirements the way a user meets them:
// each test copies an example repo, runs the documented one-line installer
// command in it, and asserts what that acceptance's Then promises. Tests run
// the candidate installer at FITNESS_INSTALL, which the release build makes.
package requirements

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ansi matches terminal color codes, which a reader does not see as text.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// example copies tests/examples/<name> into a fresh directory, writes the
// Given's files over it, and commits it as a user's repo would be. Each test
// skips first, before any setup, when no candidate installer is set.
func example(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	installer(t)
	repo := t.TempDir()
	if err := os.CopyFS(repo, os.DirFS(filepath.Join("..", "..", "tests", "examples", name))); err != nil {
		t.Fatal(err)
	}
	commit(t, repo, files, "docs(readme): add readme")
	return repo
}

// commit writes files into repo and commits everything with message.
func commit(t *testing.T, repo string, files map[string]string, message string) {
	t.Helper()
	write(t, repo, files)
	git(t, repo, "init", "-q")
	git(t, repo, "add", "-A")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", message)
}

// write puts each file into repo, creating folders as needed. An empty body
// deletes the file.
func write(t *testing.T, repo string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(repo, filepath.FromSlash(name))
		if body == "" {
			_ = os.Remove(path)
			continue
		}
		mustDo(t, os.MkdirAll(filepath.Dir(path), 0o755))
		mustDo(t, os.WriteFile(path, []byte(body), 0o755))
	}
}

func git(t *testing.T, repo string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func mustDo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// fitness runs the documented one-liner, `fitness-install -- <args>`, in
// repo with env added, and returns what it printed and its exit code. With
// no args it runs the full suite.
func fitness(t *testing.T, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	if len(args) == 0 {
		args = []string{"--policy=external", "--all"}
	}
	cmd := exec.Command(installer(t), append([]string{"--"}, args...)...)
	cmd.Dir, cmd.Env = repo, append(userEnv(), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("running fitness: %v", err)
	}
	return ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// userEnv is this process's environment without the CI job's GITHUB_*
// variables, which a user's own terminal does not have.
func userEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GITHUB_") {
			env = append(env, kv)
		}
	}
	return env
}

// installer returns the candidate fitness-install the environment provides.
func installer(t *testing.T) string {
	t.Helper()
	path := os.Getenv("FITNESS_INSTALL")
	if path == "" {
		t.Skip("set FITNESS_INSTALL to the candidate fitness-install to run requirement tests")
	}
	return path
}

// read returns text as a person reads it: the table's borders dropped and its
// wrapped lines joined, so a message reads as one line.
func read(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "│", " ")), " ")
}

// sees asserts the user saw every want and got exit code code.
func sees(t *testing.T, out string, gotCode, code int, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(read(out), read(w)) {
			t.Errorf("output is missing %q:\n%s", w, out)
		}
	}
	if gotCode != code {
		t.Errorf("exit code = %d, want %d:\n%s", gotCode, code, out)
	}
}

// resultRow matches a results-table line: name, status, files, time.
var resultRow = regexp.MustCompile(`^│ (\S+)\s+│ (passed|failed)\s+│`)

// row returns the results-table line for check, or "". The table shortens
// long names with "…", as a reader sees them.
func row(out, check string) string {
	for _, line := range strings.Split(out, "\n") {
		if m := resultRow.FindStringSubmatch(line); m != nil && names(m[1], check) {
			return line
		}
	}
	return ""
}

// names reports whether a table cell shows check, whole or shortened.
func names(cell, check string) bool {
	short, cut := strings.CutSuffix(cell, "…")
	return cell == check || cut && strings.HasPrefix(check, short)
}

// passedWithNoFiles asserts check's row shows a pass that judged 0 files.
func passedWithNoFiles(t *testing.T, out, check string) {
	t.Helper()
	if !regexp.MustCompile(`│ passed\s+│ 0\s+│`).MatchString(row(out, check)) {
		t.Errorf("%s must pass with 0 files, got row %q", check, row(out, check))
	}
}
