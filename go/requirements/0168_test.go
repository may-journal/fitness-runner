package requirements

import (
	"os"
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
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(engine, "--policy=external", "--all", "--checks=markdown-links")
		cmd.Dir, cmd.Env = repo, cleanHome0168(t)
		return cmd
	})
	sees(t, out, code, 1, "installed 1 checks in ", "missing.md", "0 of 1 checks passed")
}

func Test0168_2(t *testing.T) {
	t.Parallel()
	bin := filepath.Join(t.TempDir(), "fitness-install")
	build := exec.Command("go", "build", "-o", bin, "./cmd/fitness-install")
	build.Dir = ".."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, _ := exec.Command(bin, "--help").CombinedOutput()
	if !strings.Contains(string(out), `(default "latest")`) {
		t.Errorf("an unstamped fitness-install must default to latest:\n%s", out)
	}
	_ = os.Remove(bin)
}
