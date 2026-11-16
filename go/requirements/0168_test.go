package requirements

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// cleanHome0168 is the user's environment with an empty home, and no Fitness
// cache or installed checks handed down.
func cleanHome0168(t *testing.T) []string {
	t.Helper()
	env := slices.DeleteFunc(userEnv(), func(kv string) bool {
		return strings.HasPrefix(kv, "FITNESS_CACHE_DIR=") || strings.HasPrefix(kv, "FITNESS_CHECKS_DIR=") || strings.HasPrefix(kv, "XDG_CACHE_HOME=")
	})
	return append(env, "HOME="+t.TempDir())
}

func Test0168_1(t *testing.T) {
	t.Parallel()
	repo := pinned0164(t, `{"markdown-links": "v1.0.0"}`)
	commit(t, repo, map[string]string{"broken.md": "# Broken\n\nSee [the guide](missing.md).\n"}, "docs(broken): link a missing guide")
	engine := filepath.Join(installOnly(t), "fitness")
	env := cleanHome0168(t)
	run := func() (string, int) {
		return execute(t, func() *exec.Cmd {
			cmd := exec.Command(engine, "--policy=external", "--all", "--checks=markdown-links")
			cmd.Dir, cmd.Env = repo, env
			return cmd
		})
	}
	out, code := run()
	sees(t, out, code, 1, "fetched 1 checks this run reaches", "missing.md", "0 of 1 checks passed")
	again, code := run()
	sees(t, again, code, 1, "missing.md")
	if strings.Contains(again, "downloading fitness-check-") {
		t.Errorf("a second run must reuse the cached check and download nothing:\n%s", again)
	}
	bare, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(engine, "--no-builtins", "--policy=external", "--all", "--checks=markdown-links")
		cmd.Dir, cmd.Env = repo, cleanHome0168(t)
		return cmd
	})
	sees(t, bare, code, 1, "no check is installed, and --no-builtins turns the built-in checks off")
}

func Test0168_2(t *testing.T) {
	t.Parallel()
	bin := filepath.Join(t.TempDir(), "fitness-install")
	build := exec.Command("go", "build", "-o", bin, "./cmd/fitness-install")
	build.Dir = ".."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--help").CombinedOutput()
	if !strings.Contains(string(out), `(default "latest")`) {
		t.Errorf("an unstamped fitness-install must default to latest (%v):\n%s", err, out)
	}
}
