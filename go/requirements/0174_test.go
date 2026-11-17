package requirements

import (
	"crypto/sha256"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// action0174 is the text of the shared action name.
func action0174(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "actions", name, "action.yml"))
	mustDo(t, err)
	return string(data)
}

// step0174 is the text of the step named name in text, up to the next step.
func step0174(t *testing.T, text, name string) string {
	t.Helper()
	_, step, found := strings.Cut(text, "- name: "+name+"\n")
	if !found {
		t.Fatalf("no step named %q in:\n%s", name, text)
	}
	step, _, _ = strings.Cut(step, "\n    - ")
	return step
}

// hasLine reports whether a line of text, trimmed, is line.
func hasLine(text, line string) bool {
	return slices.ContainsFunc(strings.Split(text, "\n"), func(l string) bool { return strings.TrimSpace(l) == line })
}

// script0174 runs the shared action script at path with env added, and
// returns the outputs it wrote. It runs the script by its path, as the
// action's step does, so a missing execute bit fails.
func script0174(t *testing.T, path string, env ...string) map[string]string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "output")
	script, err := filepath.Abs(filepath.Join("..", "..", ".github", "actions", path))
	mustDo(t, err)
	cmd := exec.Command(script)
	cmd.Env = append(append(os.Environ(), "GITHUB_OUTPUT="+output), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", path, err, out)
	}
	data, _ := os.ReadFile(output)
	outputs := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			outputs[key] = value
		}
	}
	return outputs
}

// indexHash0174 is the SHA-256 of the release index GitHub serves now.
func indexHash0174(t *testing.T) string {
	t.Helper()
	response, err := http.Get("https://github.com/may-journal/fitness-checks/releases/download/fitness-check-index/index.json")
	mustDo(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	mustDo(t, err)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func Test0174_1(t *testing.T) {
	t.Parallel()
	keys := script0174(t, "fitness-install/index-key.sh", "RUNNER_OS=Linux", "RUNNER_ARCH=ARM64", "GITHUB_JOB=check")
	index := "fitness-checks-Linux-ARM64-" + indexHash0174(t) + "-"
	want := map[string]string{"base": "fitness-checks-Linux-ARM64-", "index": index, "job": index + "check-"}
	if !maps.Equal(keys, want) {
		t.Errorf("the checks cache keys must hold the release index hash, want %q, got %q", want, keys)
	}
	text := action0174(t, "fitness-install")
	restore := step0174(t, text, "Restore the installed checks")
	for _, key := range []string{"job", "index", "base"} {
		if !hasLine(restore, "${{ steps.index.outputs."+key+" }}") || strings.Contains(text, ".fitnessrc.json')") {
			t.Errorf("the restore must fall back to the %s set, never key on .fitnessrc.json:\n%s", key, restore)
		}
	}
}

// cache0174 is a home whose checks cache holds programs a and b, with a
// check set linking only a.
func cache0174(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cache := filepath.Join(home, ".cache", "fitness")
	write(t, cache, map[string]string{"checks/a-1/fitness-check-a": "a", "checks/b-1/fitness-check-b": "b"})
	mustDo(t, os.MkdirAll(filepath.Join(cache, "check-sets", "run"), 0o755))
	mustDo(t, os.Symlink(filepath.Join(cache, "checks", "a-1", "fitness-check-a"), filepath.Join(cache, "check-sets", "run", "fitness-check-a")))
	return home
}

func Test0174_3(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		restored string
		kept     []string
	}{
		{"fitness-checks-Linux-ARM64-old-check-1", []string{"a-1"}},
		{"fitness-checks-Linux-ARM64-new-check-1", []string{"a-1", "b-1"}},
	} {
		home := cache0174(t)
		keys := script0174(t, "save-checks/set-key.sh", "HOME="+home, "XDG_CACHE_HOME=", "FITNESS_INDEX_KEY=fitness-checks-Linux-ARM64-new-", "FITNESS_JOB_KEY=fitness-checks-Linux-ARM64-new-check-", "FITNESS_RESTORED_KEY="+c.restored)
		entries, err := os.ReadDir(filepath.Join(home, ".cache", "fitness", "checks"))
		mustDo(t, err)
		var kept []string
		for _, entry := range entries {
			kept = append(kept, entry.Name())
		}
		if !slices.Equal(kept, c.kept) || !strings.HasPrefix(keys["key"], "fitness-checks-Linux-ARM64-new-check-") {
			t.Errorf("restored from %s, the set must keep %q under the job's prefix, kept %q as %q", c.restored, c.kept, kept, keys["key"])
		}
	}
}

func Test0174_2(t *testing.T) {
	t.Parallel()
	install := step0174(t, action0174(t, "fitness-command"), "Install fitness")
	if want := "with-checks: ${{ github.event_name == 'workflow_dispatch' && 'pinned' || 'on-demand' }}"; !strings.Contains(install, want) {
		t.Errorf("fitness-command must install on demand for events and every pinned check for a manual run, want %q:\n%s", want, install)
	}
	ci, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	mustDo(t, err)
	suite := jobs0167(string(ci))["fitness"]
	if !strings.Contains(string(ci), "push:\n    branches: [main]") || !strings.Contains(suite, `checks="$(githooks/checks-dir)"`) {
		t.Errorf("a push to main must install every check before the suite:\n%s", suite)
	}
	all, err := os.ReadFile(filepath.Join("..", "..", "githooks", "checks-dir"))
	mustDo(t, err)
	if !strings.Contains(string(all), "--with-checks --install-only") {
		t.Errorf("githooks/checks-dir must install every check:\n%s", all)
	}
}
