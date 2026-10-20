package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// workflow0093 is the reusable workflow that approves green release PRs.
const workflow0093 = "../../.github/workflows/release-approve.yml"

// author0093 is the login gh shows for the automation App's PRs.
const author0093 = "app/may-journal-automation"

// section0093 returns the workflow text of the step named name, up to the
// next step.
func section0093(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(workflow0093)
	mustDo(t, err)
	_, rest, ok := strings.Cut(string(raw), "- name: "+name+"\n")
	if !ok {
		t.Fatalf("%s has no step %q", workflow0093, name)
	}
	section, _, _ := strings.Cut(rest, "      - name: ")
	return section
}

// run0093 runs the named step's script as Actions does, with bash -e and
// pipefail, for release PR 23 checked at commit abc123 in Actions run 99.
// GitHub's CLI answers from gh, and env adds to or
// overrides the step's environment. It returns the stand-ins, the output,
// the exit code, and what the step wrote to GITHUB_OUTPUT.
func run0093(t *testing.T, name string, gh []response, env ...string) (replays, string, int, string) {
	t.Helper()
	_, script, ok := strings.Cut(section0093(t, name), "run: |\n")
	if !ok {
		t.Fatalf("step %q has no run script", name)
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	output := filepath.Join(t.TempDir(), "output")
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", block0091(script))
		cmd.Dir = t.TempDir()
		cmd.Env = append(append(append(userEnv(), rp.env...), "PR=23", "SHA=abc123", "GH_TOKEN=app-token", "GH_REPO="+repo0014,
			"GITHUB_REPOSITORY="+repo0014, "GITHUB_OUTPUT="+output, "GITHUB_RUN_ID=99", "AUTHOR="+author0093,
			"TRIES=2", "WAIT_SECONDS=0", "KIND=release"), env...)
		return cmd
	})
	written, _ := os.ReadFile(output)
	return rp, out, code, string(written)
}

// view0093 is the CLI's answer to reading PR 23 by author, open as a ready PR
// at commit abc123.
func view0093(author string) response {
	return response{Match: []string{"pr view 23 --json author"},
		Stdout: `{"author":{"login":"` + author + `"},"state":"OPEN","isDraft":false,"headRefOid":"abc123"}` + "\n"}
}

// files0093 is GitHub's paged answer listing PR 23's files, one page each.
func files0093(pages ...string) response {
	return response{Match: []string{"pulls/23/files --paginate"}, Stdout: strings.Join(pages, "\n") + "\n"}
}

// releaseFiles0093 is a release PR's files, over two pages.
var releaseFiles0093 = files0093(`[{"filename":"CHANGELOG.md"},{"filename":"version.txt"}]`, `[{"filename":".release-please-manifest.json"}]`)

// guard0093 runs the release PR check with gh answering from gh.
func guard0093(t *testing.T, gh ...response) (replays, string, int, string) {
	t.Helper()
	return run0093(t, "Check the release PR", gh)
}

// checks0093 is the CLI's answer listing PR 23's checks as JSON, exiting as gh
// does: 8 while any is pending.
func checks0093(json string, exit int) response {
	return response{Match: []string{"pr checks 23 --json bucket,link"}, Stdout: json + "\n", Exit: exit}
}

// head0093 is the CLI's answer reading PR 23's head commit.
func head0093(sha string) response {
	return response{Match: []string{"pr view 23 --json headRefOid"}, Stdout: sha + "\n"}
}

// wait0093 runs the green-checks wait with gh answering from gh.
func wait0093(t *testing.T, gh ...response) (replays, string, int, string) {
	t.Helper()
	return run0093(t, "Wait for green checks", gh)
}

// reviews0093 is GitHub naming the approver's token as its machine user,
// listing PR 23's reviews as reviews, and accepting a new one.
func reviews0093(reviews string) []response {
	return []response{{Match: []string{"api user --jq .login"}, Stdout: "may-journal-approver\n"}, {Match: []string{"pulls/23/reviews --paginate"}, Stdout: reviews + "\n"}, {Match: []string{"-X POST"}, Stdout: "{}\n"}}
}

// approved0093 is a page of reviews holding the approver's approval of
// commit abc123.
const approved0093 = `[{"user":{"login":"jasonvanderslice"},"commit_id":"abc000","state":"APPROVED"},` +
	`{"user":{"login":"may-journal-approver"},"commit_id":"abc123","state":"APPROVED"}]`

func Test0093_1(t *testing.T) {
	t.Parallel()
	_, out, code, written := guard0093(t, view0093(author0093), releaseFiles0093)
	sees(t, out, code, 0)
	if written != "sha=abc123\n" {
		t.Fatalf("a release PR of release files alone was not passed on: output %q\n%s", written, out)
	}
	_, out, code, written = wait0093(t, checks0093(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"skipping","link":"https://github.com/may-journal/example/actions/runs/12/job/2"}]`, 0), head0093("abc123"))
	sees(t, out, code, 0)
	if written != "green=true\n" {
		t.Fatalf("green checks were not reported green: output %q\n%s", written, out)
	}
	rp, out, code, _ := run0093(t, "Approve the PR", reviews0093(`[]`))
	sees(t, out, code, 0, "Approved PR #23 at abc123.")
	if !rp.called("gh", "-X POST repos/"+repo0014+"/pulls/23/reviews -f event=APPROVE -f commit_id=abc123") {
		t.Fatalf("the release PR was not approved at the checked commit: %q", rp.calls("gh"))
	}
	rp, out, code, _ = run0093(t, "Approve the PR", reviews0093(`[]`+"\n"+approved0093))
	sees(t, out, code, 0, "PR #23 is already approved at abc123.")
	if rp.called("gh", "-X POST") {
		t.Fatalf("a release PR approved at abc123 was approved again: %q", rp.calls("gh"))
	}
}

func Test0093_2(t *testing.T) {
	t.Parallel()
	for name, files := range map[string]response{
		".github/workflows/ci.yml": files0093(`[{"filename":"CHANGELOG.md"},{"filename":".github/workflows/ci.yml"}]`),
		".github/release.sh":       files0093(`[{"filename":"version.txt","previous_filename":".github/release.sh"}]`),
	} {
		rp, out, code, written := guard0093(t, view0093(author0093), files)
		sees(t, out, code, 0, "::notice::PR #23 changes more than the release files, so a person approves it: "+name)
		if written != "" || rp.called("gh", "pr checks") || rp.called("gh", "reviews") {
			t.Errorf("a release PR changing %s went on toward approval: output %q, gh calls %q", name, written, rp.calls("gh"))
		}
	}
}

func Test0093_3(t *testing.T) {
	t.Parallel()
	rp, out, code, written := guard0093(t, view0093("json-claude-bot"), releaseFiles0093)
	sees(t, out, code, 0, "::notice::PR #23 is by json-claude-bot, not "+author0093+", so a person approves it.")
	if written != "" || rp.called("gh", "pulls/23/files") {
		t.Errorf("a PR by another author went on toward approval: output %q, gh calls %q", written, rp.calls("gh"))
	}
}

func Test0093_4(t *testing.T) {
	t.Parallel()
	_, out, code, written := wait0093(t, checks0093(`[{"bucket":"fail","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/13/job/3"}]`, 1), head0093("abc123"))
	sees(t, out, code, 0, "::notice::PR #23 has a failed or cancelled check, so it gets no approval.")
	if written != "" {
		t.Errorf("a release PR with a failed check was reported green: %q", written)
	}
	rp, out, code, written := wait0093(t, checks0093(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/13/job/3"}]`, 8), head0093("abc123"))
	sees(t, out, code, 0, "::warning::PR #23 still had checks running after 2 tries, so a person approves it.")
	if written != "" || rp.count("gh", "pr checks") != 2 {
		t.Errorf("unfinished checks were not read once per try, or were reported green: output %q, gh calls %q", written, rp.calls("gh"))
	}
	none := response{Match: []string{"pr checks 23"}, Stderr: "no checks reported on the 'release-please--branches--main' branch\n", Exit: 1}
	_, out, code, written = wait0093(t, none, head0093("abc123"))
	sees(t, out, code, 0, "::warning::PR #23 still had checks running after 2 tries")
	if written != "" {
		t.Errorf("a release PR with no checks yet was reported green: %q", written)
	}
}

func Test0093_5(t *testing.T) {
	t.Parallel()
	own := checks0093(`[{"bucket":"pass","link":"https://github.com/may-journal/example/actions/runs/11/job/1"},{"bucket":"pending","link":"https://github.com/may-journal/example/actions/runs/99/job/4"},{"bucket":"pass","link":null}]`, 8)
	_, out, code, written := wait0093(t, own, head0093("abc123"))
	sees(t, out, code, 0)
	if written != "green=true\n" {
		t.Errorf("Release approve's own running check held up the approval: output %q\n%s", written, out)
	}
	_, out, code, written = wait0093(t, own, head0093("def456"))
	sees(t, out, code, 0, "::notice::PR #23 moved to def456 while its checks ran, so this run approves nothing.")
	if written != "" {
		t.Errorf("a release PR pushed to while its checks ran was reported green: %q", written)
	}
}

var output0093 = regexp.MustCompile(`(?:steps\.(\w+)\.outputs|inputs)\.(\w+)`)

// runs0093 evaluates the step's `if:` as Actions does, given the earlier
// steps' outputs keyed step.output and its inputs keyed inputs.name; a
// missing value is empty.
func runs0093(t *testing.T, name string, outputs map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(section0093(t, name), "if: ${{ ")
	if !ok {
		t.Fatalf("step %q has no if", name)
	}
	cond, _, _ = strings.Cut(cond, " }}\n")
	if cond == "always()" {
		return true
	}
	cond = output0093.ReplaceAllStringFunc(cond, func(ref string) string {
		m := output0093.FindStringSubmatch(ref)
		key := m[1] + "." + m[2]
		if m[1] == "" {
			key = "inputs." + m[2]
		}
		return `"` + outputs[key] + `"`
	})
	return exec.Command("bash", "-c", "[[ "+strings.ReplaceAll(cond, "'", `"`)+" ]]").Run() == nil
}

// skipsAllButReport0093 asserts that with no step outputs set, as when the
// token is missing, every step after the token check skips but the
// report.
func skipsAllButReport0093(t *testing.T) {
	t.Helper()
	for _, step := range []string{"Check the release PR", "Check the dependency bump", "Wait for green checks", "Approve the PR"} {
		if runs0093(t, step, map[string]string{}) {
			t.Errorf("without the approver token, step %q runs, want it skipped", step)
		}
	}
	if !runs0093(t, "Report job outcome", map[string]string{}) {
		t.Error("without the approver token, the report step skips, want it run")
	}
}

func Test0093_6(t *testing.T) {
	t.Parallel()
	_, out, code, written := run0093(t, "Check the approver token", nil, "TOKEN=")
	sees(t, out, code, 0, "::notice::APPROVER_TOKEN is not available to this repo, so a person approves the PR.")
	if written != "" {
		t.Errorf("a missing token was reported ready: %q", written)
	}
	skipsAllButReport0093(t)
	_, _, _, written = run0093(t, "Check the approver token", nil, "TOKEN=token")
	if written != "ready=true\n" {
		t.Errorf("a present token was not reported ready: %q", written)
	}
}
