// cspell:ignore autorelease
// cspell:ignore fbody
// cspell:ignore zorblat

package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// planIssue is the Plan number each event and saved response in Test0006_4 names.
const planIssue = "7"

// brokenPlan has no Background section.
const brokenPlan = "> One clear pitch for this Plan.\n\n## What needs to happen\n\n- [ ] Write the tests.\n"

// planCheck runs plan-check the way its Actions step does, on the event
// payload GitHub would deliver, with gh answering from saved responses.
func planCheck(t *testing.T, event string, payload map[string]any, saved []response) (string, int, replays) {
	t.Helper()
	return ghRun(t, "plan-check", event, payload, nil, saved)
}

// ghRun runs a GitHub command as its Actions step does, in happyRepo with files
// over it, on GitHub's event payload and saved replies.
func ghRun(t *testing.T, command, event string, payload map[string]any, files map[string]string, saved []response) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	rp := standIn(t, map[string][]response{"gh": saved})
	env := append(rp.env, "GITHUB_EVENT_NAME="+event, "GITHUB_EVENT_PATH="+path, "GITHUB_REPOSITORY=may-journal/example")
	out, code := fitness(t, repo, env, command)
	return out, code, rp
}

// judgePlan runs plan-check on the Plan with body, as an issue edit triggers
// it.
func judgePlan(t *testing.T, body string, saved []response) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 7, "title": "t", "body": body}
	return planCheck(t, "issues", bodyEdit(issue), saved)
}

// bodyEdit is the event GitHub delivers when issue's body is edited: its
// changes name the body's earlier text.
func bodyEdit(issue map[string]any) map[string]any {
	return map[string]any{"action": "edited", "issue": issue, "changes": map[string]any{"body": map[string]any{"from": "An earlier body."}}}
}

// idleEdits are Plan 7's issues events that cannot change its verdict: a title
// edit and each added label.
func idleEdits(labels ...string) []map[string]any {
	issue := map[string]any{"number": 7, "title": "t", "body": brokenPlan, "labels": []map[string]string{{"name": "Plan"}}}
	events := []map[string]any{{"action": "edited", "issue": issue, "changes": map[string]any{"title": map[string]any{"from": "s"}}}}
	for _, l := range labels {
		events = append(events, map[string]any{"action": "labeled", "issue": issue, "label": map[string]any{"name": l}})
	}
	return events
}

// skipsIdle runs command on each idle event and wants it to note why, call
// GitHub not at all, and exit 0.
func skipsIdle(t *testing.T, command string, events []map[string]any) {
	t.Helper()
	for _, ev := range events {
		out, code, rp := ghRun(t, command, "issues", ev, nil, planGitHub(""))
		sees(t, out, code, 0, "#7 ", "so it is not checked.")
		if calls := rp.calls("gh"); len(calls) > 0 {
			t.Errorf("want no GitHub call for %v, got %q", ev["action"], calls)
		}
	}
}

// planGitHub is the saved GitHub holding the Plan's verdict comments, as the
// verdicts query prints them, and its labels, with every write succeeding.
func planGitHub(verdicts string, labels ...string) []response {
	return issueGitHub(planIssue, "fitness:plan-structure:", verdicts, append([]string{"Plan"}, labels...)...)
}

// issueGitHub is the saved GitHub holding issue n's labels and verdict comments,
// found by a query naming marker, where every write succeeds.
func issueGitHub(n, marker, verdicts string, labels ...string) []response {
	issue, _ := json.Marshal(map[string]any{"body": "", "labels": labels})
	return []response{
		{Match: []string{marker}, Stdout: verdicts},
		{Match: []string{"minimizeComment"}},
		{Match: []string{"labels: [.labels[].name]", "issues/" + n}, Stdout: string(issue) + "\n"},
		{Match: []string{"-X POST"}},
		{Match: []string{"-X PATCH", "issues/comments/"}},
		{Match: []string{"-X DELETE", "issues/" + n + "/labels/"}},
		{Match: []string{"--silent", "/labels/fitness"}},
	}
}

// planVerdict is one fitness bot verdict comment, id 11, starting with text and
// marked as judging body.
func planVerdict(text, body string) string {
	return verdictBy("github-actions", "Bot", 11, text, body)
}

// verdictBy is planVerdict with comment id, written by login, an account of kind,
// where an app's login has no [bot] suffix.
func verdictBy(login, kind string, id int, text, body string) string {
	sum := sha256.Sum256([]byte(body))
	mark := "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
	line, _ := json.Marshal(map[string]any{"id": id, "nodeId": fmt.Sprintf("IC_%d", id), "body": text + "\n\n<sub>Checked by fitness-runner</sub>\n\n" + mark, "isHidden": false,
		"author": map[string]any{"login": login, "__typename": kind}})
	return string(line) + "\n"
}

// commented reports whether plan-check posted a comment on the Plan
// holding every want.
func commented(rp replays, want ...string) bool {
	return commentedOn(rp, planIssue, want...)
}

// commentedOn reports whether a GitHub command posted a comment on issue n
// holding every want.
func commentedOn(rp replays, n string, want ...string) bool {
	return rp.called("gh", append([]string{"-X POST", "body=", "issues/" + n + "/comments"}, want...)...)
}

// labeled reports whether plan-check added label to the Plan.
func labeled(rp replays, label string) bool {
	return labeledOn(rp, planIssue, label)
}

// labeledOn reports whether a GitHub command added label to issue n.
func labeledOn(rp replays, n, label string) bool {
	return rp.called("gh", "-X POST", "labels[]="+label, "issues/"+n+"/labels")
}

// touched reports whether plan-check edited or hid comment id.
func touched(rp replays, id string) bool {
	return rp.called("gh", "IC_"+id) || rp.called("gh", "issues/comments/"+id)
}

// readOnce fails unless branch-check fetched issue n exactly once.
func readOnce(t *testing.T, rp replays, n string) {
	t.Helper()
	count := 0
	for _, call := range rp.calls("gh") {
		if strings.Contains(call, "issues/"+n) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("linked Plan #%s fetched %d times: %q", n, count, rp.calls("gh"))
	}
}

// repoPR is the repository the Actions event comes from.
const repoPR = "may-journal/example"

func itoa(n int) string { return strconv.Itoa(n) }

// actionsEvent saves an Actions event payload and returns the environment
// a workflow step sees: event name, payload path, and repository.
func actionsEvent(t *testing.T, event string, payload any) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	return []string{"GITHUB_EVENT_NAME=" + event, "GITHUB_EVENT_PATH=" + path, "GITHUB_REPOSITORY=" + repoPR}
}

// planPR is a Plan that follows the template, with one task.
func planPR(task string) string {
	return "> One clear pitch for this Plan.\n\n## Background\n\nThis Plan is a test fixture.\n\n## What needs to happen\n\n" + task + "\n"
}

// stampPR is the dated changelog entry a fixture PR adds and links.
const stampPR = "2026.10.10.0900"

// prPR is a PR description that follows the template, links the entry
// stamped stampPR, and closes issue n.
func prPR(n int) string {
	return prBodyPR(linkPR("CHANGELOG.md#"+strings.ReplaceAll(stampPR, ".", "")), n)
}

// linkPR is a `## Changelog` item linking target on the PR's branch.
func linkPR(target string) string {
	return "- [CHANGELOG entry](https://github.com/" + repoPR + "/blob/feature/" + target + ")"
}

// prBodyPR is a PR description with changelog as its `## Changelog`
// section, closing issue n.
func prBodyPR(changelog string, n int) string {
	return fmt.Sprintf("> One clear summary of the change.\n\n## Background\n\nThis change is a test fixture.\n\n## Changelog\n\n%s\n\nCloses #%d\n", changelog, n)
}

// issuePR is what gh prints for issue n read through fitness's filter:
// its body and label names.
func issuePR(t *testing.T, n int, body string, labels ...string) response {
	t.Helper()
	out, err := json.Marshal(map[string]any{"body": body, "labels": append([]string{}, labels...)})
	mustDo(t, err)
	return response{Match: []string{"repos/" + repoPR + "/issues/" + itoa(n)}, Stdout: string(out) + "\n"}
}

// noChangelogPR is GitHub listing no CHANGELOG.md change in a PR.
var noChangelogPR = response{Match: []string{"/files"}}

// livePR is GitHub answering PR 1's read with its title and body now.
func livePR(t *testing.T, title, body string) response {
	t.Helper()
	out, err := json.Marshal(map[string]any{"number": 1, "title": title, "body": body})
	mustDo(t, err)
	return response{Match: []string{"title, body", "repos/" + repoPR + "/pulls/1"}, Stdout: string(out) + "\n"}
}

// judgePR runs pr-check on a pull_request event with title and body,
// GitHub answering from issues and the PR leaving the changelog alone.
func judgePR(t *testing.T, title, body string, issues ...response) (string, int) {
	t.Helper()
	out, code, _ := rerunPR(t, title, body, issues...)
	return out, code
}

// rerunPR runs pr-check on a saved pull_request event carrying title and
// body, as "Re-run job" replays it, GitHub answering from saved first.
func rerunPR(t *testing.T, title, body string, saved ...response) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": append(saved, noChangelogPR)})
	pr := map[string]any{"number": 1, "title": title, "body": body}
	env := actionsEvent(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
	return out, code, rp
}

// entryFirst and entryLater are the stamps of a branch's first and a later
// commit. The fixture PR description links entryFirst.
const (
	entryFirst = stampPR
	entryLater = "2026.10.10.1000"
)

// entryCompare is the compare view of release 1.1.0.
const entryCompare = "https://github.com/" + repoPR + "/compare/v1.0.0...v1.1.0"

// entryCommits is a branch off main with files, one commit per set of
// changelog files, and GitHub's saved answer for its PR.
func entryCommits(t *testing.T, files map[string]string, body string, commits ...map[string]string) (string, response) {
	t.Helper()
	repo := branchRepo(t, files)
	pr := branchAnswer(t, repo, "test(app): add one file", body)
	entryApply(t, repo, commits)
	return repo, pr
}

// entryRebased is entryCommits after the base branch gained merged, another
// PR's files, past the base commit GitHub recorded for the open PR.
func entryRebased(t *testing.T, merged map[string]string, body string, commits ...map[string]string) (string, response) {
	t.Helper()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", body)
	commit(t, repo, merged, "fix(app): merged from another PR")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	entryApply(t, repo, commits)
	return repo, pr
}

// entryApply commits each set of changelog files to repo as one of a PR's
// commits.
func entryApply(t *testing.T, repo string, commits []map[string]string) {
	t.Helper()
	for i, changed := range commits {
		changed["src.txt"] = "change " + strconv.Itoa(i) + "\n"
		commit(t, repo, changed, "test(app): add one file")
	}
}

// entryActions runs pr-check in Actions on repo's branch, GitHub answering
// with the branch's changed entry files, then the released responses.
func entryActions(t *testing.T, repo, body string, released ...response) (string, int) {
	t.Helper()
	files := response{Match: []string{"repos/" + repoPR + "/pulls/1/files"}}
	entries := response{Match: []string{"repos/" + repoPR + "/pulls/1/files", "startswith"}, Stdout: entryFilesAnswer(t, repo)}
	saved := append(released, entries, files, entryNoRelease, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"))
	rp := standIn(t, map[string][]response{"gh": saved})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": body}
	env := actionsEvent(t, "pull_request", map[string]any{"action": "synchronize", "pull_request": pr})
	return fitness(t, repo, append(rp.env, env...), "pr-check")
}

// entryPatch is the branch's diff of path as GitHub's files API sends it:
// from the first hunk on, empty if unchanged.
func entryPatch(t *testing.T, repo, path string) string {
	t.Helper()
	diff, err := exec.Command("git", "-C", repo, "diff", "origin/main...HEAD", "--", path).Output()
	mustDo(t, err)
	if _, patch, ok := strings.Cut(string(diff), "\n@@"); ok {
		return "@@" + patch
	}
	return ""
}

// entryFilesAnswer is GitHub's filtered answer for the entry files the
// branch changes: one line per file, with its status and diff.
func entryFilesAnswer(t *testing.T, repo string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "diff", "--name-status", "origin/main...HEAD", "--", "changelog").Output()
	mustDo(t, err)
	statuses := map[string]string{"A": "added", "M": "modified", "D": "removed"}
	var answer strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		code, path, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		data, err := json.Marshal(map[string]string{"filename": path, "status": statuses[code], "patch": entryPatch(t, repo, path)})
		mustDo(t, err)
		answer.WriteString(string(data) + "\n")
	}
	return answer.String()
}

// entryLocal runs branch-check on repo's branch before it is pushed, with
// GitHub answering released first, then that the repo has no release.
func entryLocal(t *testing.T, repo string, pr response, released ...response) (string, int) {
	t.Helper()
	saved := append(released, branchGitHub(pr, entryNoRelease, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"))...)
	rp := standIn(t, map[string][]response{"gh": saved})
	return fitness(t, repo, rp.env, "branch-check")
}

// entryMarkdown is a finding as the Actions summary escapes it.
func entryMarkdown(finding string) string {
	return strings.ReplaceAll(finding, "#", `\#`)
}

// entryFileBody is a PR description linking the entry file at path.
func entryFileBody(path string) string {
	return prBodyPR(linkPR(path), 11)
}

// entryReleasePR is GitHub's answer for release PR 7 with body, from
// Release Please's branch at commit head.
func entryReleasePR(t *testing.T, body, head string) response {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"number": 7, "state": "open", "body": body,
		"base":   map[string]any{"ref": "main"},
		"head":   map[string]any{"ref": "release-please--branches--main", "sha": head, "repo": map[string]any{"full_name": repoPR}},
		"labels": []map[string]string{{"name": "autorelease: pending"}},
	})
	mustDo(t, err)
	return response{Match: []string{"repos/" + repoPR + "/pulls/7"}, Stdout: string(data) + "\n"}
}

// entryCopiedNotes is release PR 7's description as Release Please writes
// it, copying the release notes.
const entryCopiedNotes = "> Prepare a reviewed Fitness release.\n\n## Background\n\nRelease Please groups merged changes.\n\n" +
	"Closes #5. Agent: Release Please.\n\n<!-- fitness-release-issue:5 -->\n\n## Changelog\n\n---\n\n" +
	"### [1.1.0](" + entryCompare + ") (2026-10-11)\n\n\n### Features\n\n* **app:** add one file\n\n---\nRelease Please manages this release PR.\n"

// entryUndated asserts pr-check and branch-check fail a PR adding an entry
// file whose name lacks its heading's date, naming the dated name.
func entryUndated(t *testing.T) {
	t.Helper()
	const undated = "changelog/one-file.md"
	repo, pr := entryCommits(t, nil, entryFileBody(undated), map[string]string{undated: entryFileText(entryFirst, "Feat: add one file.")})
	want := undated + " must be named " + entryFirstFile + ": its heading's date, then the change"
	out, code := entryActions(t, repo, entryFileBody(undated))
	sees(t, out, code, 1, want)
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
}

// entryFirstFile and entryLaterFile are the release branch's entry files,
// dated entryFirst and entryLater.
const (
	entryFirstFile = "changelog/2026-10-10-0900-one-file.md"
	entryLaterFile = "changelog/2026-10-10-1000-later-file.md"
)

// entryOrdered reports whether text holds each part, in the order given.
func entryOrdered(text string, parts ...string) bool {
	at := -1
	for _, p := range parts {
		next := strings.Index(text, p)
		if next <= at {
			return false
		}
		at = next
	}
	return true
}

// entryShips is the `## Changelog` list the release tool writes for release
// PR 7 at commit head, before v1.1.0 is tagged.
func entryShips(head string) string {
	blob := "https://github.com/" + repoPR + "/blob/" + head + "/"
	return "- What ships: the entry files added since v1.0.0, and the [compare view](https://github.com/" + repoPR + "/compare/v1.0.0..." + head + ").\n" +
		"- [`" + entryLaterFile + "`](" + blob + entryLaterFile + ")\n- [`" + entryFirstFile + "`](" + blob + entryFirstFile + ")"
}

// entryAllResolve fails for each link in sent to a compare, tree or blob
// page that does not resolve in repo.
func entryAllResolve(t *testing.T, repo, sent string) {
	t.Helper()
	for _, link := range entryLinks.FindAllStringSubmatch(sent, -1) {
		if !entryResolves(repo, link[1], link[2]) {
			t.Errorf("release PR body links %s, which does not resolve before the release", link[0])
		}
	}
}

// entryPending is a repo whose v1.0.0 tag holds entryOldFile, whose main adds
// two entry files and then a third, and which has no v1.1.0 tag. Release
// Please's branch lacks the third file and adds one version commit, whose hash
// it returns too.
func entryPending(t *testing.T) (string, string) {
	t.Helper()
	repo := entryTagged(t, true)
	git(t, repo, "tag", "-d", "v1.1.0")
	git(t, repo, "checkout", "-q", "-b", "release-please--branches--main")
	commit(t, repo, map[string]string{"version.txt": "1.1.0\n"}, "chore(main): release 1.1.0")
	head := iloHead(t, repo)
	git(t, repo, "checkout", "-q", "-")
	commit(t, repo, withEntries(map[string]string{"2026-10-10-1100-main-only": entryFileText("2026.10.10.1100", "Fix: land after the release branch.")}), "fix(app): land after the release branch")
	return repo, head
}

// entryLinks finds each link to the fixture repo's compare, tree or blob
// pages: the page kind, then its refs and path.
var entryLinks = regexp.MustCompile(`https://github\.com/` + regexp.QuoteMeta(repoPR) + `/(compare|tree|blob)/([^)\s"\\]+)`)

// entryResolves reports whether GitHub would show a page of kind for spec in
// repo: commits for compare, existing paths otherwise.
func entryResolves(repo, kind, spec string) bool {
	exists := func(object string) bool {
		return exec.Command("git", "-C", repo, "cat-file", "-e", object).Run() == nil
	}
	if kind == "compare" {
		from, to, ok := strings.Cut(spec, "...")
		return ok && exists(from+"^{commit}") && exists(to+"^{commit}")
	}
	ref, path, _ := strings.Cut(spec, "/")
	return exists(ref + ":" + path)
}

// entryOldFile is an entry file release v1.0.0 shipped.
const entryOldFile = "changelog/2026-10-01-0900-old-file.md"

// entryNoRelease is GitHub's answer for a repo with no release yet.
var entryNoRelease = response{Match: []string{"releases?per_page=1"}}

// entryReleased is GitHub's answer that release v1.0.0 is the newest and
// its tag holds entryOldFile.
func entryReleased() []response {
	contents := "repos/" + repoPR + "/contents"
	return []response{
		{Match: []string{"releases?per_page=1"}, Stdout: "v1.0.0\n"},
		{Match: []string{contents + "?ref=v1.0.0"}, Stdout: "changelog\ngo\n"},
		{Match: []string{contents + "/changelog?ref=v1.0.0"}, Stdout: entryOldFile + "\n"},
	}
}

// entryShipped is the finding on an edit to entryOldFile, cut before the
// angle brackets the Actions summary escapes.
const entryShipped = entryOldFile + " shipped in v1.0.0, so its notes are published; leave it as it is and add a new changelog/"

// entryFrontDropped runs pr-check and branch-check on a PR that only drops
// the front matter of the shipped entryOldFile, failing if either names it.
func entryFrontDropped(t *testing.T) {
	t.Helper()
	const item = "Feat: ship the old file."
	old := entryFileText("2026.10.01.0900", item)
	repo, pr := entryCommits(t, map[string]string{entryOldFile: old}, prPR(11), map[string]string{entryOldFile: bareEntryText("2026.10.01.0900", item)})
	out, _ := entryActions(t, repo, prPR(11), entryReleased()...)
	local, _ := entryLocal(t, repo, pr, entryReleased()...)
	for _, got := range []string{out, local} {
		if strings.Contains(read(got), "shipped in") {
			t.Errorf("an edit that only drops front matter was named as a shipped entry:\n%s", got)
		}
	}
}

// entryNotes runs release-notes for tag in repo, with GitHub answering that
// release 42 carries it. It returns the notes the tool sent.
func entryNotes(t *testing.T, repo, tag string) string {
	t.Helper()
	gh := []response{
		{Match: []string{"repos/" + repoPR + "/releases?per_page=100", `"` + tag + `"`}, Stdout: "42\n"},
		{Match: []string{"--method PATCH", "repos/" + repoPR + "/releases/42"}, Stdout: "{}\n"},
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	env := append(rp.env, "GITHUB_REF_NAME="+tag, "GITHUB_REPOSITORY="+repoPR)
	out, code := releaseToolRun(t, repo, env, "release-notes")
	sees(t, out, code, 0)
	for _, c := range rp.calls("gh") {
		if strings.Contains(c, "releases/42") {
			return c
		}
	}
	t.Fatalf("the release tool sent no notes:\n%s", strings.Join(rp.calls("gh"), "\n"))
	return ""
}

// entryTagged is a repo whose v1.0.0 tag, when old is set, holds one entry
// file, and whose v1.1.0 tag adds two more.
func entryTagged(t *testing.T, old bool) string {
	t.Helper()
	repo := releaseToolRepo(t, nil)
	if old {
		commit(t, repo, withEntries(map[string]string{"2026-10-01-0900-old-file": entryFileText("2026.10.01.0900", "Feat: ship the old file.")}), "feat(app): ship the old file")
		git(t, repo, "tag", "v1.0.0")
	}
	commit(t, repo, map[string]string{
		entryFirstFile: entryFileText(entryFirst, "Feat: add one file."),
		entryLaterFile: bareEntryText(entryLater, "Fix: tidy the file."),
	}, "feat(app): add one file")
	git(t, repo, "tag", "v1.1.0")
	return repo
}

// cleanText is a short draft every prose check accepts.
const cleanText = "Short and clean.\n"

// kindIssue is the issue number each event and saved response in Test0006_5 names.
const kindIssue = "21"

// judgeKind runs issue-check on issue 21 with body, as an issue edit
// triggers it, in happyRepo with files written over it.
func judgeKind(t *testing.T, body string, files map[string]string, saved []response) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 21, "title": "t", "body": body}
	return ghRun(t, "issue-check", "issues", bodyEdit(issue), files, saved)
}

// kindGitHub is the saved GitHub for issue 21 of kind, with no verdict yet,
// carrying labels.
func kindGitHub(kind string, labels ...string) []response {
	return issueGitHub(kindIssue, "fitness:issue-check:"+kind+":", "", labels...)
}

// epicGitHub is kindGitHub for an Epic whose sub-issues are subs, in this
// repository.
func epicGitHub(subs ...string) []response {
	var lines strings.Builder
	for _, n := range subs {
		line, _ := json.Marshal(map[string]any{"number": json.Number(n), "repo": "may-journal/example"})
		lines.Write(append(line, '\n'))
	}
	return append([]response{{Match: []string{"issues/" + kindIssue + "/sub_issues"}, Stdout: lines.String()}}, kindGitHub("epic", "Epic")...)
}

// epicBody is an Epic that follows the template, listing items.
func epicBody(items string) string {
	return "> Every fitness run finishes in a fraction of the time.\n\n## Background\n\nThis epic is a test fixture.\n\n## What needs to happen\n\n" + items + "\n"
}

// sloppy is a comment whose first line is a change-log note.
const sloppy = "Update: the parser now handles tabs.\n"

// sloppyError is the one finding comment-check reports on sloppy.
const sloppyError = `(comment): line 1: "Update:": change-log note; edit the text in place and drop the note`

// commentURL is where comment 31, which every comment event names, lives.
const commentURL = "https://github.com/may-journal/example/issues/9#issuecomment-31"

// commentCheck runs comment-check the way its Actions step does, on the event
// payload GitHub would deliver, with gh answering from saved responses.
func commentCheck(t *testing.T, event string, payload map[string]any, saved []response, extra ...string) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	rp := standIn(t, map[string][]response{"gh": saved})
	env := append(rp.env, "GITHUB_EVENT_NAME="+event, "GITHUB_EVENT_PATH="+path, "GITHUB_REPOSITORY=may-journal/example")
	out, code := fitness(t, repo, append(env, extra...), "comment-check")
	return out, code, rp
}

// eventComment is comment 31 with body, by login, an account of kind.
func eventComment(body, login, kind string) map[string]any {
	return map[string]any{"id": 31, "body": body, "html_url": commentURL, "user": map[string]any{"login": login, "type": kind}}
}

// judgeComment runs comment-check on comment 31 on issue 9, as posting it
// triggers it.
func judgeComment(t *testing.T, comment map[string]any, saved []response, extra ...string) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 9, "title": "t", "body": ""}
	return commentCheck(t, "issue_comment", map[string]any{"action": "created", "issue": issue, "comment": comment}, saved, extra...)
}

// commentGitHub is the saved GitHub holding notice, the bot's reply to comment
// 31 as the lookup prints it, and its thumbs-down, id 55. Every write to it
// succeeds.
func commentGitHub(notice string) []response {
	return []response{
		{Match: []string{"-X POST"}},
		{Match: []string{"-X PATCH"}},
		{Match: []string{"-X DELETE"}},
		{Match: []string{"reactions?content=-1"}, Stdout: "55\n"},
		{Match: []string{"fitness:comment-check:31:"}, Stdout: notice},
		{Match: []string{"fitness:comment-check:"}},
	}
}

// commentNotice is reply 41 stating verdict about comment 31.
func commentNotice(verdict string) string {
	sum := sha256.Sum256([]byte(verdict))
	mark := "<!-- fitness:comment-check:31:" + hex.EncodeToString(sum[:])[:12] + " -->"
	line, _ := json.Marshal(map[string]any{"id": 41, "body": "❌ reply\n\n" + mark})
	return string(line) + "\n"
}

// draft is a comment with a change-log note and a diagram whose table
// misses callout 2.
const draft = sloppy + "\n```mermaid\nflowchart TD\n  A[1 Start]:::s --> B[2 End]:::s\n  classDef s fill:#eef\n```\n\n" +
	"| # | Element | Description | Why |\n| - | - | - | - |\n| 1 | Start | First. | Begins. |\n"

// sweptComment is one comment as the repository comment list prints it.
func sweptComment(id int, parent string) string {
	line, _ := json.Marshal(map[string]any{"id": id, "body": sloppy, "url": commentURL, "author": "dev", "type": "User",
		"parent": "https://api.github.com/repos/may-journal/example/" + parent, "in_reply_to_id": 0})
	return string(line) + "\n"
}

// reviewComments is review 70's ten inline comments on PR 9; ids 31 and 32 are
// sloppy, the rest pass.
func reviewComments() string {
	var lines strings.Builder
	for id := 31; id <= 40; id++ {
		body := "The parser now handles tabs.\n"
		if id <= 32 {
			body = sloppy
		}
		line, _ := json.Marshal(map[string]any{"id": id, "body": body, "url": commentURL, "author": "dev",
			"parent": "https://api.github.com/repos/may-journal/example/pulls/9", "in_reply_to_id": 0})
		lines.Write(append(line, '\n'))
	}
	return lines.String()
}

// redComment is a comment markdown-no-bold-italic fails.
const redComment = "Some **bold** text."

// denied asserts the hook denies each command, naming the bold finding, in
// its own run named for the command.
func denied(t *testing.T, repo string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, repo, bashEvent(command))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "[markdown-no-bold-italic]")
		})
	}
}

// passed asserts the hook passes each command silently, in its own run
// named for the command.
func passed(t *testing.T, repo string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, repo, bashEvent(command))
			silentPass(t, out, code)
		})
	}
}

// adrIssueBody is an ADR issue body that follows the template.
const adrIssueBody = "> The app stores notes in SQLite.\n\n## Background\n\nWe sync offline.\n\n" +
	"## Decision\n\nStore notes in SQLite.\n\n## Currently Known Consequences\n\nOne file holds every note.\n"

// judgeADR runs issue-check on ADR issue 21 titled title with body, in
// happyRepo with files written over it.
func judgeADR(t *testing.T, title, body string, files map[string]string) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 21, "title": title, "body": body}
	return ghRun(t, "issue-check", "issues", bodyEdit(issue), files, kindGitHub("adr", "ADR"))
}

// allowedWord is a word only the repo's own cspell.json allows.
const allowedWord = "Zorblat"

// cleanPRBody is a PR body pr-structure and the doc checks pass.
const cleanPRBody = "> Adds the notes.\n\n## Background\n\nThe notes were missing.\n\n## Changelog\n\n- Notes.\n"

// wordRepo is a fitness repo on origin o/r whose cspell.json allows
// allowedWord.
func wordRepo(t *testing.T) string {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"cspell.json": `{"words": ["` + allowedWord + `"]}` + "\n"})
	git(t, repo, "remote", "add", "origin", "https://github.com/o/r.git")
	return repo
}

// mcpEvent is the PreToolUse event for an MCP tool with input.
func mcpEvent(tool string, input map[string]any) map[string]any {
	return map[string]any{"hook_event_name": "PreToolUse", "tool_name": tool, "tool_input": input}
}

// deniedWith asserts the hook denies each command run from dir, showing
// every want.
func deniedWith(t *testing.T, dir string, want []string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, dir, bashEvent(command))
			sees(t, out, code, 0, append([]string{`"permissionDecision":"deny"`}, want...)...)
		})
	}
}
