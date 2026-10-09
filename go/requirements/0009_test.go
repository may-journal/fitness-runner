package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hooked copies happyRepo with files over it, puts the installed fitness on
// PATH as a user's setup does, and runs `fitness init`. It returns the repo
// and the environment git needs to find fitness.
func hooked(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	out, err := exec.Command(installer(t), "--install-only").Output()
	if err != nil {
		t.Fatalf("installing fitness: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	env := []string{"PATH=" + lines[len(lines)-1] + string(os.PathListSeparator) + os.Getenv("PATH")}
	got, code := fitness(t, repo, env, "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	return repo, env
}

// user runs git in repo the way a person at the terminal does, with hooks,
// and returns what it printed and its exit code.
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
	repo, _ := hooked(t, nil)
	out, _ := exec.Command("git", "-C", repo, "config", "core.hooksPath").Output()
	if got := strings.TrimSpace(string(out)); got != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", got)
	}
	for _, name := range []string{"commit-msg", "pre-commit", "pre-push"} {
		data, _ := os.ReadFile(filepath.Join(repo, ".githooks", name))
		if !strings.Contains(string(data), "exec fitness hook "+name) {
			t.Errorf(".githooks/%s does not run fitness hook %s", name, name)
		}
	}
}

func Test0009_2(t *testing.T) {
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "docs(readme): add bold text")
	sees(t, out, code, 1, "README.md: disallowed **bold**")
}

func Test0009_3(t *testing.T) {
	repo, env := hooked(t, nil)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
	sees(t, out, code, 1, `Commit message: "added some stuff"`)
}

func Test0009_4(t *testing.T) {
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\necho local build broke\nexit 1\n"})
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 1, "local build broke")
}

func Test0009_5(t *testing.T) {
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 1, "pre-push: no 'Plan #NN' trailer")
}

func Test0009_6(t *testing.T) {
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): tidy up")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0, "HEAD -> main")
}

func Test0009_7(t *testing.T) {
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
	repo := example(t, "happyRepo", nil)
	cmd := exec.Command(installer(t), "--install-hook")
	cmd.Dir, cmd.Env = repo, userEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--install-hook: %v\n%s", err, out)
	}
	write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	user(t, repo, nil, "add", "-A")
	out, code := user(t, repo, nil, "commit", "-m", "docs(readme): add bold text")
	sees(t, out, code, 1, "README.md: disallowed **bold**")
}
