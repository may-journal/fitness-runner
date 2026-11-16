package requirements

import (
	"os"
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

// hashesIndex fails unless the action's index step hashes the release index.
func hashesIndex(t *testing.T, text string) {
	t.Helper()
	hash := step0174(t, text, "Hash the release index")
	if !strings.Contains(hash, "/releases/download/fitness-check-index/index.json") || !strings.Contains(hash, "shasum -a 256") {
		t.Errorf("the index hash step must hash the release index:\n%s", hash)
	}
}

func Test0174_1(t *testing.T) {
	t.Parallel()
	text := action0174(t, "fitness-install")
	hashesIndex(t, text)
	cache := step0174(t, text, "Restore the installed checks")
	prefix := "fitness-checks-${{ runner.os }}-${{ runner.arch }}-"
	if !strings.Contains(cache, "key: "+prefix+"${{ steps.index.outputs.hash }}") || strings.Contains(text, ".fitnessrc.json')") {
		t.Errorf("the checks cache must key on the release index hash, not .fitnessrc.json:\n%s", cache)
	}
	_, restore, _ := strings.Cut(cache, "restore-keys: |\n")
	if !hasLine(restore, prefix) {
		t.Errorf("the checks cache must fall back to any older index's set, %q:\n%s", prefix, cache)
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
