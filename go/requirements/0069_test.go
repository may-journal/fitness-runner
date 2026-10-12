package requirements

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// entryFirst and entryLater are the stamps of a branch's first and a later
// commit.
const (
	entryFirst = "2026.10.10.0900"
	entryLater = "2026.10.10.1000"
)

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

// entryActions runs pr-check in Actions on repo's branch. GitHub answers the
// PR's files with the branch's CHANGELOG.md diff, as its files API sends it.
func entryActions(t *testing.T, repo string) (string, int) {
	t.Helper()
	diff, err := exec.Command("git", "-C", repo, "diff", "origin/main...HEAD", "--", "CHANGELOG.md").Output()
	mustDo(t, err)
	_, patch, _ := strings.Cut(string(diff), "\n@@")
	data, err := json.Marshal(map[string]string{"patch": "@@" + patch})
	mustDo(t, err)
	files := response{Match: []string{"repos/" + repo0014 + "/pulls/1/files"}, Stdout: string(data) + "\n"}
	rp := standIn(t, map[string][]response{"gh": {files, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan")}})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": pr0014(11)}
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

func Test0069_1(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, strings.Replace(first, "- Feat: add one file.\n", "- Feat: add one file.\n- Test: cover the file.\n", 1))
	out, code := entryActions(t, repo)
	sees(t, out, code, 0, "Passed.")
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 0, "pr-check #1")
}

func Test0069_2(t *testing.T) {
	t.Parallel()
	first := changelogEntry(entryFirst, "Feat: add one file.")
	repo, pr := entryBranch(t, first, changelogOnTop(first, entryLater, "Test: cover the file."))
	want := "this PR adds 2 dated entries; keep ### " + entryFirst + " and move the bullets of ### " + entryLater + " into it"
	out, code := entryActions(t, repo)
	sees(t, out, code, 1, entryMarkdown(want))
	local, localCode := entryLocal(t, repo, pr)
	sees(t, local, localCode, 1, want)
}

func Test0069_3(t *testing.T) {
	t.Parallel()
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "- Docs: add the readme.", "- Docs: add the readme file.", 1))
	out, code := entryActions(t, repo)
	sees(t, out, code, 1, entryMarkdown("this PR adds no dated ### yyyy.mm.dd.HHMM entry; add one entry that describes the whole PR"))
}

func Test0069_4(t *testing.T) {
	t.Parallel()
	release := "## [1.1.0](https://github.com/" + repo0014 + "/compare/v1.0.0...v1.1.0) (2026-10-11)\n\n\n" +
		"### Features\n\n* **app:** add one file\n\n"
	repo, _ := entryBranch(t, strings.Replace(changelogBase, "### 2026.10.08.1400", release+"### 2026.10.08.1400", 1))
	out, code := entryActions(t, repo)
	sees(t, out, code, 0, "Passed.")
}
