package requirements

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// installerEnv puts the installer where a user's setup does, without fitness on
// the PATH, and returns the environment the hooks need.
func installerEnv(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	mustDo(t, os.Symlink(installer(t), filepath.Join(dir, "fitness-install")))
	lookups := rlReleases(t, http.StatusOK, rlCandidate(t))
	return slices.Concat([]string{"FITNESS_INSTALL_DIR=" + dir}, lookups.env, fiPrivateCache(t))
}

// hooked copies happyRepo with files over it, runs `fitness init` with the
// installer placed by installerEnv, and returns the repo and that environment.
func hooked(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	env := installerEnv(t)
	got, code := fitness(t, repo, env, "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	return repo, env
}

// user runs git in repo as a person at the terminal does, with hooks, returning
// its output and exit code.
func user(t *testing.T, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Dir, cmd.Env = repo, append(userEnv(), env...)
	out, _ := cmd.CombinedOutput()
	return ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// remote gives repo an empty bare origin to push to, so no test needs GitHub.
func remote(t *testing.T, repo string) {
	t.Helper()
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "add", "origin", bare)
}

func Test0009_1(t *testing.T) {
	t.Parallel()
	repo, _ := hooked(t, nil)
	out, _ := exec.Command("git", "-C", repo, "config", "core.hooksPath").Output()
	if got := strings.TrimSpace(string(out)); got != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", got)
	}
	for _, name := range []string{"commit-msg", "pre-commit", "pre-push"} {
		data, _ := os.ReadFile(filepath.Join(repo, ".githooks", name))
		if !strings.Contains(string(data), `fitness-install" --version latest -- hook `+name) {
			t.Errorf(".githooks/%s does not run the installer for hook %s", name, name)
		}
	}
}

func Test0009_2(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "docs(readme): add bold text")
	sees(t, out, code, 1, "README.md: disallowed **bold**")
}

func Test0009_3(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
	sees(t, out, code, 1, `Commit message: "added some stuff"`)
}

func Test0009_10(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff\n\nPlan 63")
	sees(t, out, code, 1, `Commit message: "added some stuff"`, `plan trailer "Plan 63" must read exactly "Plan #<number>"`)
}

func Test0009_4(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\necho local build broke\nexit 1\n"})
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 1, "local build broke")
	localHookStops(t)
}

// localHookStops asserts a pre-commit.local ignoring the stop signal, with git
// awaiting a password, is killed at the 2s timeoutMs and the commit refused.
func localHookStops(t *testing.T) {
	t.Helper()
	repo, env := hooked(t, map[string]string{".fitnessrc.json": `{"timeoutMs": 2000}` + "\n"})
	git(t, repo, "remote", "add", "origin", promptedRemote)
	path, pidFile := promptingTools(t, "fetch")
	// Git puts its own folder first on a hook's PATH, so the hook names the waiting git.
	tools := strings.SplitN(strings.TrimPrefix(path, "PATH="), string(os.PathListSeparator), 2)[0]
	write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\ntrap '' TERM\n\"" + tools + "/git\" fetch origin\n"})
	out, code := user(t, repo, append(env, path, "GIT_TERMINAL_PROMPT=0"), "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 1, "fitness hook: .githooks/pre-commit.local timed out after 2s")
	processStopped(t, pidFile)
}

func Test0009_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	mine := "#!/bin/sh\necho my own hook\n"
	write(t, repo, map[string]string{".git/hooks/pre-commit": mine})
	cmd := exec.Command(installer(t), "--install-hook")
	cmd.Dir, cmd.Env = repo, userEnv()
	out, _ := cmd.CombinedOutput()
	sees(t, string(out), cmd.ProcessState.ExitCode(), 1, "preserve existing hook")
	if data, _ := os.ReadFile(filepath.Join(repo, ".git", "hooks", "pre-commit")); string(data) != mine {
		t.Errorf("--install-hook replaced the existing hook:\n%s", data)
	}
}

func Test0009_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repo0014+".git")
	rp := standIn(t, map[string][]response{"gh": nil})
	cmd := exec.Command(installer(t), "--install-hook")
	cmd.Dir, cmd.Env = repo, userEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--install-hook: %v\n%s", err, out)
	}
	write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	user(t, repo, rp.env, "add", "-A")
	out, code := user(t, repo, rp.env, "commit", "-m", "docs(readme): add bold text")
	sees(t, out, code, 1, "README.md: disallowed **bold**")
	if len(rp.calls("gh")) != 0 {
		t.Fatalf("commit hook queried PR metadata: %q", rp.calls("gh"))
	}
}

func Test0009_9(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["semantic-commit"]}` + "\n"})
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
	sees(t, out, code, 0)
}
