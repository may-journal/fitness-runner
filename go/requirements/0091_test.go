package requirements

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// workflow0091 is the reusable Auto-merge workflow every org repo runs.
const workflow0091 = "../../.github/workflows/auto-merge.yml"

// step0091 returns the run script of the workflow step named name, its
// indentation dropped, as Actions hands it to bash.
func step0091(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(workflow0091)
	mustDo(t, err)
	_, step, ok := strings.Cut(string(raw), "- name: "+name+"\n")
	if ok {
		_, step, ok = strings.Cut(step, "run: |\n")
	}
	if !ok {
		t.Fatalf("%s has no run script for step %q", workflow0091, name)
	}
	return block0091(step)
}

// block0091 returns text's leading lines indented at least as deep as its
// first, with that indentation dropped: a YAML block scalar's content.
func block0091(text string) string {
	lines := strings.Split(text, "\n")
	indent := lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " "))]
	var script []string
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, indent) {
			break
		}
		script = append(script, strings.TrimPrefix(line, indent))
	}
	return strings.Join(script, "\n")
}

// autoMerge0091 runs the workflow's auto-merge step as Actions does, with
// bash -e and pipefail, on a pull_request event for PR 2 based on base in a
// repo whose default branch is main, GitHub answering from saved responses.
func autoMerge0091(t *testing.T, base string) (replays, string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": {{Match: []string{"pr merge --auto --squash"}}}})
	env := event0014(t, "pull_request", map[string]any{
		"action":       "synchronize",
		"repository":   map[string]any{"default_branch": "main"},
		"pull_request": map[string]any{"number": 2, "html_url": "https://github.com/" + repo0014 + "/pull/2", "base": map[string]any{"ref": base}},
	})
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", step0091(t, "Enable squash auto-merge"))
		cmd.Dir, cmd.Env = t.TempDir(), append(append(userEnv(), rp.env...), append(env, "GH_TOKEN=app-token")...)
		return cmd
	})
	return rp, out, code
}

func Test0091_1(t *testing.T) {
	t.Parallel()
	rp, out, code := autoMerge0091(t, "photos-grid")
	sees(t, out, code, 0, "::notice::This PR is stacked on photos-grid", "It turns on once the PR targets main.")
	if calls := rp.calls("gh"); len(calls) != 0 {
		t.Fatalf("a stacked PR asked GitHub for auto-merge: %q", calls)
	}
}

func Test0091_2(t *testing.T) {
	t.Parallel()
	rp, out, code := autoMerge0091(t, "main")
	sees(t, out, code, 0)
	if !rp.called("gh", "pr merge --auto --squash https://github.com/"+repo0014+"/pull/2") {
		t.Fatalf("a PR moved onto main did not get auto-merge: %q", rp.calls("gh"))
	}
}
