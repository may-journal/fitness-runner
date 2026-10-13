// cspell:ignore autorelease
package requirements

import (
	"encoding/base64"
	"encoding/json"
	"os/exec"
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

// entryRelease is release 1.1.0 as Release Please writes it in CHANGELOG.md.
const entryRelease = "## [1.1.0](" + entryCompare + ") (2026-10-11)\n\n\n### Features\n\n* **app:** add one file\n\n"

// entryBranch is a branch off main with one commit per changelog, as a PR's
// commits write it, and GitHub's saved answer for that branch's open PR.
func entryBranch(t *testing.T, changelogs ...string) (string, response) {
	t.Helper()
	commits := make([]map[string]string, len(changelogs))
	for i, changelog := range changelogs {
		commits[i] = map[string]string{"CHANGELOG.md": changelog}
	}
	return entryCommits(t, nil, pr0014(11), commits...)
}

// entryCommits is a branch off main, with files over it, with one commit
// per set of changelog files, and GitHub's saved answer for its open PR.
func entryCommits(t *testing.T, files map[string]string, body string, commits ...map[string]string) (string, response) {
	t.Helper()
	repo := branchRepo(t, files)
	pr := branchAnswer(t, repo, "test(app): add one file", body)
	for i, changed := range commits {
		changed["src.txt"] = "change " + strconv.Itoa(i) + "\n"
		commit(t, repo, changed, "test(app): add one file")
	}
	return repo, pr
}

// entryActions runs pr-check in Actions on repo's branch, its PR described
// by body. GitHub answers the PR's files with the branch's CHANGELOG.md diff
// and changed entry files, as its files API sends them.
func entryActions(t *testing.T, repo, body string) (string, int) {
	t.Helper()
	files := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files"}, Stdout: entryPatch(t, repo, "CHANGELOG.md")}
	if patch := files.Stdout; patch != "" {
		data, err := json.Marshal(map[string]string{"patch": patch})
		mustDo(t, err)
		files.Stdout = string(data) + "\n"
	}
	entries := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files", "startswith"}, Stdout: entryFilesAnswer(t, repo)}
	rp := standIn(t, map[string][]response{"gh": {entries, files, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan")}})
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

// entryLocal runs branch-check on repo's branch before it is pushed.
func entryLocal(t *testing.T, repo string, pr response) (string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))})
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
// Release Please's branch at commit abc1234.
func entryReleasePR(t *testing.T, body string) response {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"number": 7, "state": "open", "body": body,
		"base":   map[string]any{"ref": "main"},
		"head":   map[string]any{"ref": "release-please--branches--main", "sha": "abc1234", "repo": map[string]any{"full_name": repo0014}},
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

func Test0069_10(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, strings.Replace(first, "- Feat: add one file.\n", "- Feat: add one file.\n- Test: cover the file.\n", 1))
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 0, "Passed.")
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
	entry := entryFileText(entryFirst, "Feat: add one file.")
	repo, pr = entryCommits(t, nil, entryFileBody("changelog/one-file.md"),
		map[string]string{"changelog/one-file.md": entry},
		map[string]string{"changelog/one-file.md": entry + "- Fix: name the file.\n"})
	out, code = entryActions(t, repo, entryFileBody("changelog/one-file.md"))
	sees(t, out, code, 0, "Passed.")
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
}

func Test0069_11(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, changelogOnTop(first, entryLater, "Test: cover the file."))
	want := "this PR adds 2 changelog entries; keep ### " + entryFirst + " and move the bullets of ### " + entryLater + " into it"
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, entryMarkdown(want))
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
	repo, pr = entryCommits(t, nil, entryFileBody("changelog/a-file.md"),
		map[string]string{"changelog/a-file.md": entryFileText(entryFirst, "Feat: add one file.")},
		map[string]string{"changelog/b-file.md": entryFileText(entryLater, "Test: cover the file.")})
	want = "this PR adds 2 changelog entries; keep changelog/a-file.md and move the bullets of changelog/b-file.md into it"
	out, code = entryActions(t, repo, entryFileBody("changelog/a-file.md"))
	sees(t, out, code, 1, want)
	local, localCode = entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
	repo, _ = entryCommits(t, nil, entryFileBody("changelog/a-file.md"),
		map[string]string{"changelog/a-file.md": entryFileText(entryFirst, "Feat: add one file.")},
		map[string]string{"CHANGELOG.md": changelogEntry(entryLater, "Test: cover the file.")})
	out, code = entryActions(t, repo, entryFileBody("changelog/a-file.md"))
	sees(t, out, code, 1, entryMarkdown("keep changelog/a-file.md and move the bullets of ### "+entryLater+" into it"))
}

func Test0069_12(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "- Docs: add the readme.", "- Docs: add the readme file.", 1))
	out, code := entryActions(t, repo, pr0014(11))
	// The Actions summary escapes the angle brackets of changelog/<name>.md.
	want := ".md entry file, or one ### yyyy.mm.dd.HHMM section in CHANGELOG.md, that describes the whole PR"
	sees(t, out, code, 1, "this PR adds no dated changelog entry", entryMarkdown(want))
	old := entryFileText(entryFirst, "Feat: add one file.")
	repo, _ = entryCommits(t, withEntries(map[string]string{"old-change": old}), pr0014(11),
		map[string]string{"changelog/old-change.md": old + "- Fix: name the file.\n"})
	out, code = entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, "this PR adds no dated changelog entry", entryMarkdown(want))
}

func Test0069_4(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "### 2026.10.08.1400", entryRelease+"### 2026.10.08.1400", 1))
	body := prBody0014("- What ships: the [CHANGELOG section](https://github.com/"+repo0014+"/blob/abc1234/CHANGELOG.md#110-2026-10-11) and [compare view]("+entryCompare+").", 11)
	out, code := entryActions(t, repo, body)
	sees(t, out, code, 0, "Passed.")
}

func Test0069_9(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, changelogEntry(entryLater, "Feat: add one file."))
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, entryMarkdown("PR description links CHANGELOG.md#202610100900; link the entry this PR adds"), entryMarkdown("CHANGELOG.md#202610101000"))
	repo, _ = entryBranch(t, changelogEntry(entryFirst, "Feat: add one file."))
	out, code = entryActions(t, repo, prBody0014("- The entry covers the change.", 11))
	sees(t, out, code, 1, entryMarkdown("PR description has no CHANGELOG.md link; link the entry this PR adds"), entryMarkdown("CHANGELOG.md#202610100900"))
	repo, _ = entryCommits(t, nil, entryFileBody("changelog/other-file.md"), map[string]string{"changelog/one-file.md": entryFileText(entryFirst, "Feat: add one file.")})
	out, code = entryActions(t, repo, entryFileBody("changelog/other-file.md"))
	sees(t, out, code, 1, "PR description links changelog/other-file.md; link the entry this PR adds", "changelog/one-file.md")
}

func Test0069_6(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, changelogEntry(entryFirst, "Feat: add one file to the app."))
	body := strings.Replace(pr0014(11), "\n\nCloses", "\n- Feat: add one file to the app.\n\nCloses", 1)
	out, code := entryActions(t, repo, body)
	sees(t, out, code, 1, "PR description copies the CHANGELOG.md line", "Feat: add one file to the app.", "link the entry instead")
}

func Test0069_7(t *testing.T) {
	t.Parallel()
	patch := response{Match: []string{"--method PATCH", "repos/" + repo0014 + "/pulls/7"}, Stdout: "{}\n"}
	rp := standIn(t, map[string][]response{"gh": {patch, entryReleasePR(t, entryCopiedNotes)}})
	env := append(rp.env, "RELEASE_PR={\"number\":7}", "GITHUB_REPOSITORY="+repo0014)
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), env, "release-pr")
	sees(t, out, code, 0)
	section := "[CHANGELOG section](https://github.com/" + repo0014 + "/blob/abc1234/CHANGELOG.md#110-2026-10-11)"
	if !rp.called("gh", "--method PATCH", section, "[compare view]("+entryCompare+")") {
		t.Errorf("release PR body does not link its changelog section and compare view:\n%s", strings.Join(rp.calls("gh"), "\n"))
	}
	if rp.called("gh", "--method PATCH", "add one file") {
		t.Errorf("release PR body still copies its notes:\n%s", strings.Join(rp.calls("gh"), "\n"))
	}
}

// entryFolded is the CHANGELOG.md the release tool committed, decoded from
// its one commit call, and that call's input; both are "" when it made none.
func entryFolded(t *testing.T, calls []string) (string, string) {
	t.Helper()
	for _, c := range calls {
		_, input, ok := strings.Cut(c, "--input - ")
		if !ok || !strings.Contains(c, "createCommitOnBranch") {
			continue
		}
		var sent struct {
			Variables struct {
				Input struct {
					FileChanges struct {
						Additions []struct{ Contents string } `json:"additions"`
					} `json:"fileChanges"`
				} `json:"input"`
			} `json:"variables"`
		}
		mustDo(t, json.Unmarshal([]byte(input), &sent))
		raw, err := base64.StdEncoding.DecodeString(sent.Variables.Input.FileChanges.Additions[0].Contents)
		mustDo(t, err)
		return string(raw), input
	}
	return "", ""
}

// entryFoldedUnderRelease asserts the release tool's one commit put the
// entry between the release notes and the older section, and deleted its
// file on the release branch it read.
func entryFoldedUnderRelease(t *testing.T, calls []string) {
	t.Helper()
	folded, input := entryFolded(t, calls)
	release, older := strings.Index(folded, "* **app:** add one file"), strings.Index(folded, "### 2026.10.08.1400")
	if at := strings.Index(folded, "### "+entryLater); at < release || at > older {
		t.Errorf("the entry is not committed under the version section:\n%s", folded)
	}
	for _, want := range []string{`"deletions":[{"path":"changelog/one-file.md"}]`, `"expectedHeadOid":"abc1234"`} {
		if !strings.Contains(input, want) {
			t.Errorf("the commit does not delete the entry file on the release branch:\n%s", input)
		}
	}
}

func Test0069_13(t *testing.T) {
	t.Parallel()
	entry := entryFileText(entryLater, "Feat: add one file.")
	repo := releaseToolRepo(t, withEntries(map[string]string{"one-file": entry}))
	contents := "repos/" + repo0014 + "/contents/"
	gh := []response{
		{Match: []string{"--method PATCH", "repos/" + repo0014 + "/pulls/7"}, Stdout: "{}\n"},
		{Match: []string{contents + "changelog?ref=abc1234"}, Stdout: "changelog/one-file.md\n"},
		{Match: []string{contents + "changelog/one-file.md?ref=abc1234"}, Stdout: entry},
		{Match: []string{contents + "CHANGELOG.md?ref=abc1234"}, Stdout: strings.Replace(changelogBase, "### 2026.10.08.1400", entryRelease+"### 2026.10.08.1400", 1)},
		{Match: []string{"graphql", "createCommitOnBranch"}, Stdout: "def5678\n"},
		entryReleasePR(t, entryCopiedNotes),
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	env := append(rp.env, "RELEASE_PR={\"number\":7}", "GITHUB_REPOSITORY="+repo0014)
	out, code := releaseToolRun(t, repo, env, "release-pr")
	sees(t, out, code, 0)
	entryFoldedUnderRelease(t, rp.calls("gh"))
	if !rp.called("gh", "--method PATCH", "/blob/def5678/CHANGELOG.md#110-2026-10-11") {
		t.Errorf("release PR body does not link the folded commit:\n%s", strings.Join(rp.calls("gh"), "\n"))
	}
}
