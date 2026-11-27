// Package acceptance runs requirement tests as a user meets them: each test
// runs the installer at FITNESS_INSTALL in a copied example repo. The repo
// root's suite and each language package's suite share it.
package acceptance

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Ansi matches terminal color codes, which a reader does not see as text.
var Ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Cases is an acceptance's table: each case is one situation the acceptance
// covers, named by its Given.
type Cases []struct {
	Name string
	Run  func(*testing.T)
}

// Run runs each case in parallel as its own t.Run of the acceptance's test.
func (cs Cases) Run(t *testing.T) {
	t.Helper()
	t.Parallel()
	cs.RunAlone(t)
}

// RunAlone runs the cases like Run, but before the parallel acceptances start.
// A case that times the product skips t.Parallel, so it runs on an idle machine.
func (cs Cases) RunAlone(t *testing.T) {
	t.Helper()
	for _, c := range cs {
		t.Run(c.Name, c.Run)
	}
}

// repoGitConfig is the git config every test repo gets, whatever the host's,
// since a signed tag needs a message no test gives. Background maintenance after
// a commit can still be writing a pack when the test removes the repo.
var repoGitConfig = [][2]string{{"tag.gpgSign", "false"}, {"maintenance.auto", "false"}, {"gc.auto", "0"}}

// KeepHostGitOut puts repoGitConfig in the environment, so every git the
// tests or fitness start reads it.
func KeepHostGitOut() {
	for i, kv := range repoGitConfig {
		_ = os.Setenv(fmt.Sprintf("GIT_CONFIG_KEY_%d", i), kv[0])
		_ = os.Setenv(fmt.Sprintf("GIT_CONFIG_VALUE_%d", i), kv[1])
	}
	_ = os.Setenv("GIT_CONFIG_COUNT", fmt.Sprint(len(repoGitConfig)))
}

// Main is a suite's TestMain: it keeps the host's git config out, then runs
// the tests.
func Main(m *testing.M) {
	KeepHostGitOut()
	os.Exit(m.Run())
}

// Example copies tests/examples/<name> into a fresh directory, writes the
// Given's files over it, and commits it as a user's repo would be. Each test
// skips first, before any setup, when no candidate installer is set.
func Example(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	Installer(t)
	repo := t.TempDir()
	if err := os.CopyFS(repo, os.DirFS(filepath.Join(examples(t), name))); err != nil {
		t.Fatal(err)
	}
	Commit(t, repo, files, "docs(readme): add readme")
	return repo
}

// examples is the repo's tests/examples, beside the Go module that holds the
// test's folder, so suites at any depth share the example repos.
func examples(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	MustDo(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(filepath.Dir(dir), "tests", "examples")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's folder")
		}
		dir = parent
	}
}

// Commit writes files into repo and commits everything with message.
func Commit(t *testing.T, repo string, files map[string]string, message string) {
	t.Helper()
	Write(t, repo, files)
	Git(t, repo, "init", "-q")
	Git(t, repo, "add", "-A")
	Git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", message)
}

// Write puts each file into repo, creating folders as needed. An empty body
// deletes the file.
func Write(t *testing.T, repo string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(repo, filepath.FromSlash(name))
		if body == "" {
			_ = os.Remove(path)
			continue
		}
		MustDo(t, os.MkdirAll(filepath.Dir(path), 0o755))
		MustDo(t, os.WriteFile(path, []byte(body), 0o755))
	}
}

// Git runs git with args in repo, failing the test on an error.
func Git(t *testing.T, repo string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// MustDo fails the test on a non-nil err.
func MustDo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Fitness runs the documented one-liner, `fitness-install -- <args>`, in
// repo with env added, and returns what it printed and its exit code. With
// no args it runs the full suite.
func Fitness(t *testing.T, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	return Piped(t, repo, "", env, args...)
}

// Piped runs the one-liner like Fitness, with stdin as the text a user pipes
// in; empty stdin means none.
func Piped(t *testing.T, repo, stdin string, env []string, args ...string) (string, int) {
	t.Helper()
	if len(args) == 0 {
		args = []string{"--policy=external", "--all"}
	}
	return Execute(t, func() *exec.Cmd {
		cmd := exec.Command(Installer(t), append([]string{"--"}, args...)...)
		cmd.Dir, cmd.Env = repo, append(UserEnv(), env...)
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		return cmd
	})
}

// Execute runs the command build makes and returns its output, color codes
// dropped, and exit code, retrying a busy binary briefly.
func Execute(t *testing.T, build func() *exec.Cmd) (string, int) {
	t.Helper()
	var out []byte
	cmd := Started(t, build, func(cmd *exec.Cmd) (err error) {
		out, err = cmd.CombinedOutput()
		return err
	})
	return Ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// Started runs each command build makes with run until one starts, and
// returns it. A busy binary is retried as Execute says; any other start
// error fails the test.
func Started(t *testing.T, build func() *exec.Cmd, run func(*exec.Cmd) error) *exec.Cmd {
	t.Helper()
	for attempt := 0; ; attempt++ {
		cmd := build()
		err := run(cmd)
		if cmd.ProcessState != nil {
			return cmd
		}
		if !errors.Is(err, syscall.ETXTBSY) || attempt == 40 {
			t.Fatalf("start %s: %v", cmd.Path, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// UserEnv is this process's environment without the CI job's GITHUB_*
// variables, which a user's own terminal lacks. It names a local API host in
// front of GitHub's.
func UserEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GITHUB_") {
			env = append(env, kv)
		}
	}
	// A test that names its own API host appends it later, and the last wins.
	return append(env, "GITHUB_API_URL="+LocalAPI())
}

// LocalAPI is the address of one local server that passes each request on to
// GitHub's API. The runner's probe of GitHub's API host then reaches a local
// one however busy the machine is, while each read still reaches GitHub.
var LocalAPI = sync.OnceValue(func() string {
	github := &url.URL{Scheme: "https", Host: "api.github.com"}
	proxy := httputil.NewSingleHostReverseProxy(github)
	direct := proxy.Director
	proxy.Director = func(r *http.Request) {
		direct(r)
		r.Host = github.Host
	}
	return httptest.NewServer(proxy).URL
})

// Installer returns the candidate fitness-install the environment provides.
func Installer(t *testing.T) string {
	t.Helper()
	path := os.Getenv("FITNESS_INSTALL")
	if path == "" {
		t.Skip("set FITNESS_INSTALL to the candidate fitness-install to run requirement tests")
	}
	return path
}
