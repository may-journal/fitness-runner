package requirements

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// workflow0092 is the reusable Release Please workflow opted-in repos call.
const workflow0092 = "../../.github/workflows/release-please-reusable.yml"

// firstRelease0092 is what Release Please writes for a first release, which
// has no compare link: a `##` version heading and bold scopes.
const firstRelease0092 = "> Release the changes merged since the last version.\n\n## Background\n\nRelease Please groups merged changes.\n\n## Changelog\n---\n\n\n" +
	"## 1.0.0 (2026-10-10)\n\n\n### Features\n\n* **library:** one API over UIKit and AppKit\n\n---\nAgent: Release Please."

// section0092 returns the workflow text of the step named name, up to the
// next step.
func section0092(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(workflow0092)
	mustDo(t, err)
	_, rest, ok := strings.Cut(string(raw), "- name: "+name+"\n")
	if !ok {
		t.Fatalf("%s has no step %q", workflow0092, name)
	}
	section, _, _ := strings.Cut(rest, "      - name: ")
	return section
}

// run0092 runs the named step's script as Actions does for release PR 23 at
// commit abc123, with gh answering from saved responses.
func run0092(t *testing.T, name string, gh []response) (replays, string, int, string) {
	t.Helper()
	_, script, ok := strings.Cut(section0092(t, name), "run: |\n")
	if !ok {
		t.Fatalf("step %q has no run script", name)
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	output := filepath.Join(t.TempDir(), "output")
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-c", block0091(script))
		cmd.Dir = t.TempDir()
		cmd.Env = append(append(userEnv(), rp.env...), "PR=23", "GH_TOKEN=app-token", "GH_REPO="+repo0014, "GITHUB_OUTPUT="+output,
			"GITHUB_REPOSITORY="+repo0014, "MAJOR=v1", "SHA=abc123")
		return cmd
	})
	written, _ := os.ReadFile(output)
	return rp, out, code, string(written)
}

// view0092 is the CLI's answer to reading release PR 23's body.
func view0092(body string) response {
	return response{Match: []string{"pr view 23 --json body"}, Stdout: body + "\n"}
}

// edits0092 is the CLI's answer to any other pull request call.
var edits0092 = response{Match: []string{"pr "}}

// logged0092 is text as the stand-in logs it, newlines escaped.
func logged0092(text string) string {
	return strings.ReplaceAll(text, "\n", `\n`)
}

func Test0092_1(t *testing.T) {
	t.Parallel()
	rp, out, code, written := run0092(t, "Shape the release PR notes", []response{view0092(firstRelease0092), edits0092})
	sees(t, out, code, 0)
	shaped := "> Release the changes merged since the last version.\n\n## Background\n\nRelease Please groups merged changes.\n\n## Changelog\n\n---\n\n\n" +
		"### 1.0.0 (2026-10-10)\n\n\n### Features\n\n* library: one API over UIKit and AppKit\n\n---\nAgent: Release Please."
	if !rp.called("gh", "pr edit 23 --body "+logged0092(shaped)) {
		t.Errorf("the release PR body was not shaped; gh calls: %q", rp.calls("gh"))
	}
	if written != "edited=true\n" {
		t.Errorf("shape step output = %q, want edited=true", written)
	}
	rp, out, code, _ = run0092(t, "Rerun pr-check on the shaped body", []response{edits0092})
	sees(t, out, code, 0)
	if calls := rp.calls("gh"); strings.Join(calls, "\n") != "pr close 23\npr reopen 23" {
		t.Errorf("gh calls after the rewrite = %q, want the PR closed then reopened", calls)
	}
}

func Test0092_2(t *testing.T) {
	t.Parallel()
	const title = "Publish the release from PR #23"
	body := "> Release the changes.\n\nAgent: Release Please."
	rp, out, code, written := run0092(t, "Link the release PR to its tracking issue", []response{
		{Match: []string{"issue list"}, Stdout: "[]\n"},
		{Match: []string{"issue create --title " + title}, Stdout: "https://github.com/" + repo0014 + "/issues/41\n"},
		view0092(body), edits0092,
	})
	sees(t, out, code, 0)
	if !rp.called("gh", "pr edit 23 --body "+logged0092(body+"\n\nCloses #41.")) {
		t.Errorf("the release PR does not close its new tracking issue; gh calls: %q", rp.calls("gh"))
	}
	if written != "edited=true\n" {
		t.Errorf("link step output = %q, want edited=true", written)
	}
	if !rp.called("gh", "issue list --state all --search \""+title+"\" in:title") {
		t.Errorf("the tracking issue lookup does not search by title; gh calls: %q", rp.calls("gh"))
	}
	linkedAgain0092(t, title, body+"\n\nCloses #41.")
}

// linkedAgain0092 runs the link step again once the tracking issue exists
// as issue 41 and the release PR body already closes it.
func linkedAgain0092(t *testing.T, title, body string) {
	t.Helper()
	tracked, err := json.Marshal([]map[string]any{{"number": 41, "title": title}})
	mustDo(t, err)
	rp, out, code, written := run0092(t, "Link the release PR to its tracking issue", []response{
		{Match: []string{"issue list"}, Stdout: string(tracked) + "\n"},
		view0092(body), edits0092,
	})
	sees(t, out, code, 0)
	if rp.called("gh", "issue create") || rp.called("gh", "pr edit") || written != "" {
		t.Errorf("a linked release PR was changed again; gh calls: %q, output %q", rp.calls("gh"), written)
	}
}

var (
	outputRef0092 = regexp.MustCompile(`steps\.\w+\.outputs\.\w+`)
	fromJSON0092  = regexp.MustCompile(`fromJSON\(([^)]*)\)`)
)

// runs0092 evaluates the step's `if:` as Actions does, given the outputs of
// the earlier steps keyed step.output; a step without `if:` runs.
func runs0092(t *testing.T, name string, outputs map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(section0092(t, name), "if: ${{ ")
	if !ok {
		return true
	}
	cond, _, _ = strings.Cut(cond, " }}\n")
	cond = outputRef0092.ReplaceAllStringFunc(cond, func(ref string) string {
		return `"` + outputs[strings.TrimPrefix(strings.ReplaceAll(ref, ".outputs.", "."), "steps.")] + `"`
	})
	return exec.Command("bash", "-c", "[[ "+cond+" ]]").Run() == nil
}

// readsEmptyPR reports whether every fromJSON in the step's env gets valid
// JSON when Release Please sets no `pr` output. Actions evaluates env even
// for a skipped step, and fromJSON of an empty string fails the job.
func readsEmptyPR(t *testing.T, name string) bool {
	t.Helper()
	for _, m := range fromJSON0092.FindAllStringSubmatch(section0092(t, name), -1) {
		arg := "" // steps.release.outputs.pr is empty
		if _, fallback, ok := strings.Cut(m[1], " || '"); ok {
			arg, _, _ = strings.Cut(fallback, "'")
		}
		if !json.Valid([]byte(arg)) {
			return false
		}
	}
	return true
}

func Test0092_3(t *testing.T) {
	t.Parallel()
	noPR := map[string]string{"release.release_created": "true"}
	for _, step := range []string{"Shape the release PR notes", "Link the release PR to its tracking issue", "Rerun pr-check on the shaped body"} {
		if runs0092(t, step, noPR) {
			t.Errorf("with no release PR, step %q runs, want it skipped", step)
		}
		if !readsEmptyPR(t, step) {
			t.Errorf("with no release PR, step %q cannot read its env: fromJSON gets an empty string", step)
		}
	}
	if !runs0092(t, "Move the major version tag", noPR) {
		t.Error("with a release created, the major version tag step is skipped, want it run")
	}
}

// tag0092 runs the major tag step with GitHub answering the tag lookup with
// lookup, and accepting any change to a ref.
func tag0092(t *testing.T, lookup response) (replays, string, int) {
	t.Helper()
	lookup.Match = []string{"git/ref/tags/v1"}
	rp, out, code, _ := run0092(t, "Move the major version tag", []response{lookup, {Match: []string{"git/refs"}}})
	return rp, out, code
}

func Test0092_4(t *testing.T) {
	t.Parallel()
	refs := "repos/" + repo0014 + "/git/refs"
	rp, out, code := tag0092(t, response{})
	sees(t, out, code, 0)
	if !rp.called("gh", "-X PATCH "+refs+"/tags/v1 -f sha=abc123") {
		t.Errorf("an existing major tag was not moved; gh calls: %q", rp.calls("gh"))
	}
	rp, out, code = tag0092(t, response{Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1})
	sees(t, out, code, 0)
	if !rp.called("gh", "-X POST "+refs+" -f ref=refs/tags/v1 -f sha=abc123") {
		t.Errorf("a missing major tag was not created; gh calls: %q", rp.calls("gh"))
	}
	rp, out, code = tag0092(t, response{Stderr: "gh: Resource not accessible by integration (HTTP 403)\n", Exit: 1})
	sees(t, out, code, 1, "::error::Could not read tag v1: gh: Resource not accessible by integration (HTTP 403)")
	if rp.called("gh", "-X ") {
		t.Errorf("a tag lookup error still changed a ref; gh calls: %q", rp.calls("gh"))
	}
}
