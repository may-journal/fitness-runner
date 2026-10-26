package requirements

import (
	"os/exec"
	"strings"
	"testing"
)

// hpShims returns the three hook shims for folder dir, each running the
// installer the way the org's synced hooks do.
func hpShims(dir string) map[string]string {
	files := map[string]string{}
	for _, name := range []string{"commit-msg", "pre-commit", "pre-push"} {
		files[dir+"/"+name] = "#!/bin/sh\nexec \"$FITNESS_INSTALL_DIR/fitness-install\" --version latest -- hook " + name + " \"$@\"\n"
	}
	return files
}

// hpStale copies happyRepo with every folder in shipped committed as hook
// shims, puts the installer where a user's setup does, and leaves the clone's
// core.hooksPath at stale, as a clone set up before the folder moved has it.
func hpStale(t *testing.T, stale string, shipped ...string) (string, []string) {
	t.Helper()
	files := map[string]string{}
	for _, dir := range shipped {
		for name, body := range hpShims(dir) {
			files[name] = body
		}
	}
	repo := example(t, "happyRepo", files)
	git(t, repo, "config", "core.hooksPath", stale)
	return repo, installerEnv(t)
}

// hpPath reads the clone's core.hooksPath.
func hpPath(t *testing.T, repo string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", repo, "config", "core.hooksPath").Output()
	return strings.TrimSpace(string(out))
}

// hpWants fails unless the clone's core.hooksPath is want.
func hpWants(t *testing.T, repo, want string) {
	t.Helper()
	if got := hpPath(t, repo); got != want {
		t.Errorf("core.hooksPath = %q, want %q", got, want)
	}
}

func Test0122_1(t *testing.T) {
	t.Parallel()
	repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 0)
	hpWants(t, repo, ".fitness-hooks")
}

func Test0122_2(t *testing.T) {
	t.Parallel()
	repo, env := hpStale(t, "old-hooks", ".githooks")
	write(t, repo, hpShims("old-hooks"))
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 0)
	hpWants(t, repo, ".githooks")
}

func Test0122_3(t *testing.T) {
	t.Parallel()
	outside := t.TempDir()
	write(t, outside, hpShims("hooks"))
	repo, env := hpStale(t, outside+"/hooks", ".fitness-hooks")
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 0, "core.hooksPath "+outside+"/hooks is outside the repo")
	hpWants(t, repo, outside+"/hooks")
}

func Test0122_4(t *testing.T) {
	t.Parallel()
	repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
	sees(t, out, code, 1, "fitness hook: core.hooksPath was .githooks, now .fitness-hooks", `Commit message: "added some stuff"`)
	if strings.Count(out, "core.hooksPath") != 1 {
		t.Errorf("want one line naming the hooks path:\n%s", out)
	}
	if strings.Index(out, "now .fitness-hooks") > strings.Index(out, `Commit message: "added some stuff"`) {
		t.Errorf("the repair line comes after the hook's errors:\n%s", out)
	}
}

func Test0122_5(t *testing.T) {
	t.Parallel()
	repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
	out, code := fitness(t, repo, env, "init")
	sees(t, out, code, 0, "set core.hooksPath to .fitness-hooks")
	hpWants(t, repo, ".fitness-hooks")
}
