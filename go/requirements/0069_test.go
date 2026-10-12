// cspell:ignore autorelease
package requirements

import (
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
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	for i, changelog := range changelogs {
		commit(t, repo, map[string]string{"CHANGELOG.md": changelog, "src.txt": "change " + strconv.Itoa(i) + "\n"}, "test(app): add one file")
	}
	return repo, pr
}

// entryActions runs pr-check in Actions on repo's branch, its PR described
// by body. GitHub answers the PR's files with the branch's CHANGELOG.md diff,
// as its files API sends it.
func entryActions(t *testing.T, repo, body string) (string, int) {
	t.Helper()
	diff, err := exec.Command("git", "-C", repo, "diff", "origin/main...HEAD", "--", "CHANGELOG.md").Output()
	mustDo(t, err)
	_, patch, _ := strings.Cut(string(diff), "\n@@")
	data, err := json.Marshal(map[string]string{"patch": "@@" + patch})
	mustDo(t, err)
	files := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files"}, Stdout: string(data) + "\n"}
	rp := standIn(t, map[string][]response{"gh": {files, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan")}})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": body}
	env := event0014(t, "pull_request", map[string]any{"action": "synchronize", "pull_request": pr})
	return fitness(t, repo, append(rp.env, env...), "pr-check")
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

func Test0069_1(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, strings.Replace(first, "- Feat: add one file.\n", "- Feat: add one file.\n- Test: cover the file.\n", 1))
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 0, "Passed.")
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
}

func Test0069_2(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, changelogOnTop(first, entryLater, "Test: cover the file."))
	want := "this PR adds 2 dated entries; keep ### " + entryFirst + " and move the bullets of ### " + entryLater + " into it"
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, entryMarkdown(want))
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
}

func Test0069_3(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "- Docs: add the readme.", "- Docs: add the readme file.", 1))
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, entryMarkdown("this PR adds no dated ### yyyy.mm.dd.HHMM entry; add one entry that describes the whole PR"))
}

func Test0069_4(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "### 2026.10.08.1400", entryRelease+"### 2026.10.08.1400", 1))
	body := prBody0014("- What ships: the [CHANGELOG section](https://github.com/"+repo0014+"/blob/abc1234/CHANGELOG.md#110-2026-10-11) and [compare view]("+entryCompare+").", 11)
	out, code := entryActions(t, repo, body)
	sees(t, out, code, 0, "Passed.")
}

func Test0069_5(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, changelogEntry(entryLater, "Feat: add one file."))
	out, code := entryActions(t, repo, pr0014(11))
	sees(t, out, code, 1, entryMarkdown("PR description links CHANGELOG.md#202610100900; link the entry this PR adds"), entryMarkdown("CHANGELOG.md#202610101000"))
}

func Test0069_8(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, changelogEntry(entryFirst, "Feat: add one file."))
	out, code := entryActions(t, repo, prBody0014("- The entry covers the change.", 11))
	sees(t, out, code, 1, entryMarkdown("PR description has no CHANGELOG.md link; link the entry this PR adds"), entryMarkdown("CHANGELOG.md#202610100900"))
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
	copied := "> Prepare a reviewed Fitness release.\n\n## Background\n\nRelease Please groups merged changes.\n\n" +
		"Closes #5. Agent: Release Please.\n\n<!-- fitness-release-issue:5 -->\n\n## Changelog\n\n---\n\n" +
		strings.Replace(entryRelease, "## [", "### [", 1) + "---\nRelease Please manages this release PR.\n"
	patch := response{Match: []string{"--method PATCH", "repos/" + repo0014 + "/pulls/7"}, Stdout: "{}\n"}
	rp := standIn(t, map[string][]response{"gh": {patch, entryReleasePR(t, copied)}})
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
