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
	t.Parallel()
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

func Test0009_3(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
	sees(t, out, code, 1, `Commit message: "added some stuff"`)
}

func Test0009_4(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\necho local build broke\nexit 1\n"})
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 1, "local build broke")
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

// pushPlans are GitHub's saved answers about the issues a trailer may name:
// #218 is an approved Plan, #151 is a Plan nobody approved, and #196 is a bug.
var pushPlans = []response{
	{Match: []string{"issue view 218 ", "labels"}, Stdout: "Plan\n"},
	{Match: []string{"issue view 218 ", "comments"}, Stdout: "Looks good.\nApproved, go ahead.\n"},
	{Match: []string{"issue view 151 ", "labels"}, Stdout: "Plan\nenhancement\n"},
	{Match: []string{"issue view 151 ", "comments"}, Stdout: "Could this reuse the existing parser?\n"},
	{Match: []string{"issue view 196 ", "labels"}, Stdout: "bug\n"},
	{Match: []string{"issue view 196 ", "comments"}, Stdout: "Approved.\n"},
}

// pushRepo is a hooked repo whose pre-push asks the gh stand-in about Plans.
// It pushes main, so later pushes start from a shared base.
func pushRepo(t *testing.T) (string, []string, replays) {
	t.Helper()
	repo, hookEnv := hooked(t, nil)
	rp := standIn(t, map[string][]response{"gh": pushPlans})
	env := pushEnv(hookEnv, rp)
	remote(t, repo)
	if out, code := user(t, repo, env, "push", "-q", "origin", "HEAD:main"); code != 0 {
		t.Fatalf("pushing main: %s", out)
	}
	return repo, env, rp
}

// pushEnv puts the stand-ins on PATH ahead of the installed fitness.
func pushEnv(hookEnv []string, rp replays) []string {
	sep := string(os.PathListSeparator)
	fitnessPath := strings.TrimPrefix(hookEnv[0], "PATH=")
	var env []string
	for _, kv := range rp.env {
		if path, ok := strings.CutPrefix(kv, "PATH="); ok {
			kv = "PATH=" + strings.SplitN(path, sep, 2)[0] + sep + fitnessPath
		}
		env = append(env, kv)
	}
	return env
}

// pushFeature commits a feature naming the Plan trailer and pushes it.
func pushFeature(t *testing.T, trailer string) (string, int, replays) {
	t.Helper()
	repo, env, rp := pushRepo(t)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): add a feature\n\n"+trailer)
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	return out, code, rp
}

// pushAsked fails t unless the hook asked GitHub about issue n's field.
func pushAsked(t *testing.T, rp replays, n, field string) {
	t.Helper()
	if !rp.called("gh", "issue view "+n+" ", field) {
		t.Errorf("pre-push never read issue %s's %s; gh calls: %q", n, field, rp.calls("gh"))
	}
}

func Test0009_14(t *testing.T) {
	t.Parallel()
	repo, env, _ := pushRepo(t)
	user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
	out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
	sees(t, out, code, 0, "[deleted]", "topic")
}

// boldCommit commits a README change the markdown check refuses.
func boldCommit(t *testing.T, repo string, env []string) (string, int) {
	t.Helper()
	write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	user(t, repo, env, "add", "-A")
	return user(t, repo, env, "commit", "-m", "docs(readme): add bold text")
}

func Test0009_15(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	out, code := boldCommit(t, repo, env)
	sees(t, out, code, 1, "README.md: disallowed **bold**")

	repo = example(t, "happyRepo", nil)
	cmd := exec.Command(installer(t), "--install-hook")
	cmd.Dir, cmd.Env = repo, userEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--install-hook: %v\n%s", err, out)
	}
	out, code = boldCommit(t, repo, nil)
	sees(t, out, code, 1, "README.md: disallowed **bold**")
}

func Test0009_16(t *testing.T) {
	t.Parallel()
	out, code, rp := pushFeature(t, "Plan #218")
	sees(t, out, code, 0, "HEAD -> main")
	pushAsked(t, rp, "218", "comments")

	out, code, rp = pushFeature(t, "Plan #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
	pushAsked(t, rp, "196", "labels")

	out, code, rp = pushFeature(t, "Plan #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)")
	pushAsked(t, rp, "151", "comments")

	out, code, _ = pushFeature(t, "")
	sees(t, out, code, 1, "pre-push: no 'Plan #NN' trailer")
}

func Test0009_17(t *testing.T) {
	t.Parallel()
	repo, env, rp := pushRepo(t)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): tidy up")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0, "HEAD -> main")

	user(t, repo, env, "switch", "-q", "-c", "merged")
	user(t, repo, env, "switch", "-q", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): merged by review (#12)")
	user(t, repo, env, "push", "-q", "--no-verify", "origin", "HEAD:main")
	user(t, repo, env, "switch", "-q", "merged")
	user(t, repo, env, "merge", "-q", "--no-edit", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): tidy up")
	out, code = user(t, repo, env, "push", "origin", "merged")
	sees(t, out, code, 0, "merged -> merged")

	user(t, repo, env, "switch", "-q", "-c", "planned", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start\n\nPlan #218")
	if out, code := user(t, repo, env, "push", "-q", "origin", "planned"); code != 0 {
		t.Fatalf("first push: %s", out)
	}
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
	out, code = user(t, repo, env, "push", "origin", "planned")
	sees(t, out, code, 0, "planned -> planned")
	pushAsked(t, rp, "218", "comments")
}
