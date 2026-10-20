// cspell:ignore autorelease
package requirements

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// entryFirst and entryLater are the stamps of a branch's first and a later
// commit. The fixture PR description links entryFirst.
const (
	entryFirst = stamp0014
	entryLater = "2026.10.10.1000"
)

// entryCompare is the compare view of release 1.1.0.
const entryCompare = "https://github.com/" + repo0014 + "/compare/v1.0.0...v1.1.0"

// entryCommits is a branch off main, with files over it, with one commit
// per set of changelog files, and GitHub's saved answer for its open PR.
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

// entryActions runs pr-check in Actions on repo's branch, its PR described
// by body. GitHub answers the PR's files with the branch's changed entry
// files, as its files API sends them, and nothing for any other files read.
// It answers released first, then that the repo has no release.
func entryActions(t *testing.T, repo, body string, released ...response) (string, int) {
	t.Helper()
	files := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files"}}
	entries := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files", "startswith"}, Stdout: entryFilesAnswer(t, repo)}
	saved := append(released, entries, files, entryNoRelease, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))
	rp := standIn(t, map[string][]response{"gh": saved})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": body}
	env := event0014(t, "pull_request", map[string]any{"action": "synchronize", "pull_request": pr})
	return fitness(t, repo, append(rp.env, env...), "pr-check")
}

// entryPatch is the branch's diff of path as GitHub's files API sends it:
// from the first hunk on, empty when the branch leaves path alone.
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
	saved := append(released, branchGitHub(pr, entryNoRelease, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))...)
	rp := standIn(t, map[string][]response{"gh": saved})
	return fitness(t, repo, rp.env, "branch-check")
}

// entryMarkdown is a finding as the Actions summary escapes it.
func entryMarkdown(finding string) string {
	return strings.ReplaceAll(finding, "#", `\#`)
}

// entryFileBody is a PR description linking the entry file at path.
func entryFileBody(path string) string {
	return prBody0014(link0014(path), 11)
}

// entryReleasePR is GitHub's answer for release PR 7 with body, from
// Release Please's branch at commit head.
func entryReleasePR(t *testing.T, body, head string) response {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"number": 7, "state": "open", "body": body,
		"base":   map[string]any{"ref": "main"},
		"head":   map[string]any{"ref": "release-please--branches--main", "sha": head, "repo": map[string]any{"full_name": repo0014}},
		"labels": []map[string]string{{"name": "autorelease: pending"}},
	})
	mustDo(t, err)
	return response{Match: []string{"repos/" + repo0014 + "/pulls/7"}, Stdout: string(data) + "\n"}
}

// entryCopiedNotes is release PR 7's description as Release Please writes
// it, copying the release notes.
const entryCopiedNotes = "> Prepare a reviewed Fitness release.\n\n## Background\n\nRelease Please groups merged changes.\n\n" +
	"Closes #5. Agent: Release Please.\n\n<!-- fitness-release-issue:5 -->\n\n## Changelog\n\n---\n\n" +
	"### [1.1.0](" + entryCompare + ") (2026-10-11)\n\n\n### Features\n\n* **app:** add one file\n\n---\nRelease Please manages this release PR.\n"

func Test0069_14(t *testing.T) {
	t.Parallel()
	entry := entryFileText(entryFirst, "Feat: add one file.")
	repo, pr := entryCommits(t, nil, entryFileBody(entryFirstFile),
		map[string]string{entryFirstFile: entry},
		map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
	out, code := entryActions(t, repo, entryFileBody(entryFirstFile))
	sees(t, out, code, 0, "Passed.")
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
	repo, pr = entryCommits(t, withEntries(map[string]string{"2026-10-10-0900-one-file": entry}), pr0014(11),
		map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
	out, code = entryActions(t, repo, pr0014(11))
	sees(t, out, code, 0, "Passed.")
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
	repo, pr = entryCommits(t, map[string]string{"CHANGELOG.md": ""}, entryFileBody(entryFirstFile),
		map[string]string{entryFirstFile: entry, "CHANGELOG.md": changelogBase})
	out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
	sees(t, out, code, 0, "Passed.")
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
	repo, pr = entryRebased(t, withEntries(map[string]string{"2026-10-10-1000-merged-file": entryFileText(entryLater, "Fix: merged from another PR.")}),
		entryFileBody(entryFirstFile),
		map[string]string{entryFirstFile: entry},
		map[string]string{entryFirstFile: entry + "- Fix: name the file.\n"})
	out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
	sees(t, out, code, 0, "Passed.")
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
}

func Test0069_24(t *testing.T) {
	t.Parallel()
	repo, pr := entryCommits(t, nil, entryFileBody("changelog/2026-10-10-0900-a-file.md"),
		map[string]string{"changelog/2026-10-10-0900-a-file.md": entryFileText(entryFirst, "Feat: add one file.")},
		map[string]string{"changelog/2026-10-10-1000-b-file.md": entryFileText(entryLater, "Test: cover the file.")})
	want := "this PR adds 2 changelog entries; keep changelog/2026-10-10-0900-a-file.md and move the bullets of changelog/2026-10-10-1000-b-file.md into it"
	out, code := entryActions(t, repo, entryFileBody("changelog/2026-10-10-0900-a-file.md"))
	sees(t, out, code, 1, want)
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
	entryUndated(t)
}

func Test0069_17(t *testing.T) {
	t.Parallel()
	added := map[string]string{entryFirstFile: entryFileText(entryFirst, "Feat: add one file to the app.")}
	repo, _ := entryCommits(t, nil, entryFileBody("changelog/other-file.md"), added)
	out, code := entryActions(t, repo, entryFileBody("changelog/other-file.md"))
	sees(t, out, code, 1, "PR description links changelog/other-file.md; link the entry this PR adds", entryFirstFile)
	out, code = entryActions(t, repo, prBody0014("- The entry covers the change.", 11))
	sees(t, out, code, 1, "PR description has no changelog/ entry link; link the entry this PR adds", entryFirstFile)
	body := strings.Replace(entryFileBody(entryFirstFile), "\n\nCloses", "\n- Feat: add one file to the app.\n\nCloses", 1)
	out, code = entryActions(t, repo, body)
	sees(t, out, code, 1, "PR description copies the "+entryFirstFile+" line", "Feat: add one file to the app.", "link the entry instead")
}

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
	blob := "https://github.com/" + repo0014 + "/blob/" + head + "/"
	return "- What ships: the entry files added since v1.0.0, and the [compare view](https://github.com/" + repo0014 + "/compare/v1.0.0..." + head + ").\n" +
		"- [`" + entryLaterFile + "`](" + blob + entryLaterFile + ")\n- [`" + entryFirstFile + "`](" + blob + entryFirstFile + ")"
}

func Test0069_20(t *testing.T) {
	t.Parallel()
	release := prBody0014(entryShips("abc1234"), 11)
	repo, pr := entryCommits(t, nil, release, map[string]string{"version.txt": "1.1.0\n"})
	out, code := entryActions(t, repo, release)
	sees(t, out, code, 0, "Passed.")
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
	repo, pr = entryCommits(t, nil, entryFileBody(entryFirstFile), map[string]string{
		entryFirstFile: entryFileText(entryFirst, "Feat: add one file."),
		"CHANGELOG.md": changelogEntry(entryFirst, "Feat: add one file."),
	})
	out, code = entryActions(t, repo, entryFileBody(entryFirstFile))
	sees(t, out, code, 0, "Passed.")
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
}

func Test0069_21(t *testing.T) {
	t.Parallel()
	patch := response{Match: []string{"--method PATCH", "repos/" + repo0014 + "/pulls/7"}, Stdout: "{}\n"}
	repo, head := entryPending(t)
	rp := standIn(t, map[string][]response{"gh": {patch, entryReleasePR(t, entryCopiedNotes, head)}})
	env := append(rp.env, "RELEASE_PR={\"number\":7}", "GITHUB_REPOSITORY="+repo0014)
	out, code := releaseToolRun(t, repo, env, "release-pr")
	sees(t, out, code, 0)
	sent := strings.Join(rp.calls("gh"), "\n")
	want, err := json.Marshal(entryShips(head))
	mustDo(t, err)
	if !rp.called("gh", "--method PATCH", strings.Trim(string(want), `"`)) {
		t.Errorf("release PR body does not link each new entry file and the compare view at its head:\n%s", sent)
	}
	if rp.called("gh", "--method PATCH", "add one file") || strings.Contains(sent, entryOldFile) || rp.called("gh", "graphql") {
		t.Errorf("release PR body still copies its notes, links a shipped entry, or the tool committed:\n%s", sent)
	}
	entryAllResolve(t, repo, sent)
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

// entryPending is a repo whose v1.0.0 tag holds entryOldFile and whose main
// adds two entry files since, with no v1.1.0 tag yet. Release Please's
// branch adds one version commit, whose hash it returns too. Main then
// gains a third entry file the release branch lacks.
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
var entryLinks = regexp.MustCompile(`https://github\.com/` + regexp.QuoteMeta(repo0014) + `/(compare|tree|blob)/([^)\s"\\]+)`)

// entryResolves reports whether GitHub would show a page of kind for spec
// in repo: both ends of a compare view are commits, and a tree or blob
// path exists at its ref.
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
	contents := "repos/" + repo0014 + "/contents"
	return []response{
		{Match: []string{"releases?per_page=1"}, Stdout: "v1.0.0\n"},
		{Match: []string{contents + "?ref=v1.0.0"}, Stdout: "changelog\ngo\n"},
		{Match: []string{contents + "/changelog?ref=v1.0.0"}, Stdout: entryOldFile + "\n"},
	}
}

// entryShipped is the finding on an edit to entryOldFile, cut before the
// angle brackets the Actions summary escapes.
const entryShipped = entryOldFile + " shipped in v1.0.0, so its notes are published; leave it as it is and add a new changelog/"

func Test0069_23(t *testing.T) {
	t.Parallel()
	old := entryFileText("2026.10.01.0900", "Feat: ship the old file.")
	for _, edit := range []string{old + "- Fix: a late bullet.\n", ""} {
		repo, pr := entryCommits(t, map[string]string{entryOldFile: old}, pr0014(11), map[string]string{entryOldFile: edit})
		out, code := entryActions(t, repo, pr0014(11), entryReleased()...)
		sees(t, out, code, 1, entryShipped)
		local, localCode := entryLocal(t, repo, pr, entryReleased()...)
		sees(t, local, localCode, 1, entryShipped)
	}
}

// entryNotes runs release-notes for tag in repo, with GitHub answering that
// release 42 carries it. It returns the notes the tool sent.
func entryNotes(t *testing.T, repo, tag string) string {
	t.Helper()
	gh := []response{
		{Match: []string{"repos/" + repo0014 + "/releases?per_page=100", `"` + tag + `"`}, Stdout: "42\n"},
		{Match: []string{"--method PATCH", "repos/" + repo0014 + "/releases/42"}, Stdout: "{}\n"},
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	env := append(rp.env, "GITHUB_REF_NAME="+tag, "GITHUB_REPOSITORY="+repo0014)
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
		entryLaterFile: entryFileText(entryLater, "Fix: tidy the file."),
	}, "feat(app): add one file")
	git(t, repo, "tag", "v1.1.0")
	return repo
}

func Test0069_22(t *testing.T) {
	t.Parallel()
	newest := []string{"### " + entryLater, "Fix: tidy the file.", "### " + entryFirst, "Feat: add one file."}
	notes := entryNotes(t, entryTagged(t, true), "v1.1.0")
	if !entryOrdered(notes, append(newest, "[Compare view]("+entryCompare+")")...) {
		t.Errorf("the notes do not list the new entries newest first, then the compare view:\n%s", notes)
	}
	if strings.Contains(notes, "ship the old file") || strings.Contains(notes, "relatedConfigurations") {
		t.Errorf("the notes list an entry from an earlier release, or front matter:\n%s", notes)
	}
	first := entryNotes(t, entryTagged(t, false), "v1.1.0")
	if !entryOrdered(first, append(newest, "[Commits](https://github.com/"+repo0014+"/commits/v1.1.0)")...) {
		t.Errorf("a first release does not list every entry, then its commits:\n%s", first)
	}
}

func Test0069_25(t *testing.T) {
	t.Parallel()
	repo := entryTagged(t, true)
	shallow := t.TempDir()
	git(t, shallow, "clone", "-q", "--depth", "1", "--branch", "v1.1.0", "file://"+repo, ".")
	for _, c := range []struct{ repo, tag, want string }{
		{shallow, "v1.1.0", "the clone is shallow, so the previous release tag is unknown"},
		{repo, "v9.9.9", "v9.9.9"},
	} {
		rp := standIn(t, map[string][]response{"gh": {{Stdout: "{}\n"}}})
		env := append(rp.env, "GITHUB_REF_NAME="+c.tag, "GITHUB_REPOSITORY="+repo0014)
		out, code := releaseToolRun(t, c.repo, env, "release-notes")
		sees(t, out, code, 1, c.want)
		if len(rp.calls("gh")) > 0 {
			t.Errorf("the release tool reached GitHub after failing:\n%s", strings.Join(rp.calls("gh"), "\n"))
		}
	}
}
