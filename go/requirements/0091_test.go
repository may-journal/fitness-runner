package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// workflow0091 is the reusable Auto-merge workflow every org repo runs.
const workflow0091 = "../../.github/workflows/auto-merge.yml"

// bot0091 is the login the org App comments as.
const bot0091 = "may-journal-automation[bot]"

// issues0091 is the REST path of PR 2's issue in the event's repo.
const issues0091 = "repos/" + repo0014 + "/issues"

// workflowText0091 returns the workflow file's text.
func workflowText0091(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(workflow0091)
	mustDo(t, err)
	return string(raw)
}

// step0091 returns the run script of the workflow step named name, its
// indentation dropped, as Actions hands it to bash.
func step0091(t *testing.T, name string) string {
	t.Helper()
	_, step, ok := strings.Cut(workflowText0091(t), "- name: "+name+"\n")
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
	lines := strings.Split(strings.TrimLeft(text, "\n"), "\n")
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

// note0091 is GitHub listing the bot's auto-merge note on PR 2 as comment
// 7 with body, or listing none when body is empty.
func note0091(body string) []response {
	if body == "" {
		return []response{{Match: []string{issues0091 + "/2/comments --paginate"}}}
	}
	return []response{
		{Match: []string{issues0091 + "/2/comments --paginate"}, Stdout: "7\n"},
		{Match: []string{issues0091 + "/comments/7 --jq .body"}, Stdout: body + "\n"},
	}
}

// autoMerge0091 runs the workflow's auto-merge step as Actions does, with
// bash -e and pipefail, on a pull_request event for PR 2 based on base in a
// repo whose default branch is main. GitHub answers merge with answer and
// lists note as the bot's earlier note, answers comment writes from writes,
// and accepts any other call.
func autoMerge0091(t *testing.T, base string, answer response, note string, writes ...response) (replays, string, int) {
	t.Helper()
	answer.Match = []string{"pr merge --auto --squash"}
	saved := append(append(append([]response{answer}, note0091(note)...), writes...), response{Match: []string{"api "}})
	rp := standIn(t, map[string][]response{"gh": saved})
	env := event0014(t, "pull_request", map[string]any{
		"action":       "synchronize",
		"repository":   map[string]any{"default_branch": "main"},
		"pull_request": map[string]any{"number": 2, "html_url": "https://github.com/" + repo0014 + "/pull/2", "base": map[string]any{"ref": base}},
	})
	env = append(env, "GH_TOKEN=app-token", "BOT="+bot0091, "GITHUB_ENV="+filepath.Join(t.TempDir(), "env"))
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", step0091(t, "Enable squash auto-merge"))
		cmd.Dir, cmd.Env = t.TempDir(), append(append(userEnv(), rp.env...), env...)
		return cmd
	})
	return rp, out, code
}

// logged0091 is a note as the stand-in logs the call that writes it.
func logged0091(text string) string {
	return "body=" + strings.ReplaceAll(text+"\n\n<!-- fitness:auto-merge -->", "\n", `\n`)
}

// posted0091 returns the note text the step posted on PR 2, or "".
func posted0091(rp replays) string {
	for _, c := range rp.calls("gh") {
		if _, body, ok := strings.Cut(c, "api "+issues0091+"/2/comments -f body="); ok {
			return body
		}
	}
	return ""
}

// stacked0091 is the note on a PR based on photos-grid.
const stacked0091 = "Auto-merge is off: this PR targets `photos-grid`, not `main`, and GitHub has no auto-merge for stacked pull requests. " +
	"Merge it by hand once it is approved and green, or merge its base PR first. Once GitHub moves this PR onto `main`, auto-merge turns on and this comment says so."

func Test0091_1(t *testing.T) {
	t.Parallel()
	rp, out, code := autoMerge0091(t, "photos-grid", response{}, "")
	sees(t, out, code, 0, "::notice::This PR targets photos-grid, not main", "Its auto-merge comment says how to merge it.")
	if rp.called("gh", "pr merge") {
		t.Fatalf("a stacked PR asked GitHub for auto-merge: %q", rp.calls("gh"))
	}
	if !rp.called("gh", "api "+issues0091+"/2/comments", logged0091(stacked0091)) {
		t.Fatalf("a stacked PR got no note saying why it has no auto-merge: %q", rp.calls("gh"))
	}
	rp, out, code = autoMerge0091(t, "photos-grid", response{}, stacked0091+"\n\n<!-- fitness:auto-merge -->")
	sees(t, out, code, 0)
	if posted0091(rp) != "" || rp.called("gh", "-X PATCH") {
		t.Fatalf("a second push to a stacked PR wrote its note again: %q", rp.calls("gh"))
	}
	refused := response{Match: []string{"-f body="}, Stderr: "gh: Resource not accessible by integration (HTTP 403)\n", Exit: 1}
	_, out, code = autoMerge0091(t, "photos-grid", response{}, "", refused)
	sees(t, out, code, 0, "HTTP 403", "::warning::GitHub refused a change to this PR's auto-merge comment")
}

func Test0091_2(t *testing.T) {
	t.Parallel()
	rp, out, code := autoMerge0091(t, "main", response{}, stacked0091+"\n\n<!-- fitness:auto-merge -->")
	sees(t, out, code, 0)
	if !rp.called("gh", "pr merge --auto --squash https://github.com/"+repo0014+"/pull/2") {
		t.Fatalf("a PR moved onto main did not get auto-merge: %q", rp.calls("gh"))
	}
	on := "Auto-merge is on: this PR squash-merges into `main` once it is approved and green."
	if !rp.called("gh", "-X PATCH "+issues0091+"/comments/7", logged0091(on)) {
		t.Fatalf("the note on a PR moved onto main was not updated: %q", rp.calls("gh"))
	}
	rp, out, code = autoMerge0091(t, "main", response{}, "")
	sees(t, out, code, 0)
	if posted0091(rp) != "" || rp.called("gh", "-X PATCH") {
		t.Fatalf("a PR that never had a note got one: %q", rp.calls("gh"))
	}
}

func Test0091_3(t *testing.T) {
	t.Parallel()
	refusal := "GraphQL: Auto-merge is not supported for stacked pull requests. (enablePullRequestAutoMerge)\n"
	rp, out, code := autoMerge0091(t, "main", response{Stderr: refusal, Exit: 1}, "")
	sees(t, out, code, 0, strings.TrimSpace(refusal), "::notice::GitHub still counts this PR as stacked, so it gets no auto-merge.")
	note := "Auto-merge is off: this PR targets `main`, but GitHub still counts it as part of a stack and refuses auto-merge. " +
		"Merge it by hand once it is approved and green, or push to it once GitHub moves it out of the stack."
	if !rp.called("gh", "api "+issues0091+"/2/comments", logged0091(note)) {
		t.Fatalf("a PR GitHub counts as stacked got no note: %q", rp.calls("gh"))
	}
}

func Test0091_4(t *testing.T) {
	t.Parallel()
	refusal := "GraphQL: Pull request is in clean status (enablePullRequestAutoMerge)\n"
	rp, out, code := autoMerge0091(t, "main", response{Stderr: refusal, Exit: 1}, "")
	sees(t, out, code, 1, strings.TrimSpace(refusal), "::error::GitHub refused auto-merge for this PR: "+strings.TrimSpace(refusal))
	if strings.Contains(out, "::notice::") || rp.called("gh", "comments") {
		t.Fatalf("another auto-merge failure passed with a notice or a note:\n%s\n%q", out, rp.calls("gh"))
	}
}

var (
	eventRef0091 = regexp.MustCompile(`github\.event\.[\w.]+`)
	null0091     = regexp.MustCompile(`\bnull\b`)
)

// runs0091 evaluates the job's `if:` as Actions does for a pull_request
// event whose fields, keyed by their path under github.event, hold event.
// Bash compares strings and Actions compares typed values, so the event
// must give every field the condition reads: "" for null, and "on" for an
// enabled auto_merge.
func runs0091(t *testing.T, event map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(workflowText0091(t), "    if: >-\n")
	if !ok {
		t.Fatalf("%s has no job-level if", workflow0091)
	}
	cond = strings.ReplaceAll(block0091(cond), "\n", " ")
	cond = null0091.ReplaceAllString(cond, `""`)
	cond = eventRef0091.ReplaceAllStringFunc(cond, func(ref string) string {
		value, given := event[strings.TrimPrefix(ref, "github.event.")]
		if !given {
			t.Fatalf("the job's if reads %s, which the test event does not give", ref)
		}
		return `"` + value + `"`
	})
	return exec.Command("bash", "-c", "[[ "+cond+" ]]").Run() == nil
}

func Test0091_5(t *testing.T) {
	t.Parallel()
	pr := func(action, autoMerge string) map[string]string {
		return map[string]string{"action": action, "pull_request.state": "open", "pull_request.draft": "false", "pull_request.auto_merge": autoMerge}
	}
	if !runs0091(t, pr("reopened", "on")) {
		t.Error("a reopened PR whose event still shows auto-merge skips the job, want it run")
	}
	if !runs0091(t, pr("reopened", "")) {
		t.Error("a reopened PR without auto-merge skips the job, want it run")
	}
	if runs0091(t, pr("synchronize", "on")) {
		t.Error("a push to a PR with auto-merge on runs the job, want it skipped")
	}
	if !runs0091(t, pr("synchronize", "")) {
		t.Error("a push to a PR without auto-merge skips the job, want it run")
	}
}
