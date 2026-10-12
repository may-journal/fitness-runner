package requirements

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// repo0014 is the repository the Actions event comes from.
const repo0014 = "may-journal/example"

func itoa(n int) string { return strconv.Itoa(n) }

// event0014 saves an Actions event payload and returns the environment
// a workflow step sees: event name, payload path, and repository.
func event0014(t *testing.T, event string, payload any) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	return []string{"GITHUB_EVENT_NAME=" + event, "GITHUB_EVENT_PATH=" + path, "GITHUB_REPOSITORY=" + repo0014}
}

// plan0014 is a Plan that follows the template, with one task.
func plan0014(task string) string {
	return "> One clear pitch for this Plan.\n\n## Background\n\nThis Plan is a test fixture.\n\n## What needs to happen\n\n" + task + "\n"
}

// stamp0014 is the dated changelog entry a fixture PR adds and links.
const stamp0014 = "2026.10.10.0900"

// pr0014 is a PR description that follows the template, links the entry
// stamped stamp0014, and closes issue n.
func pr0014(n int) string {
	return prBody0014(link0014("CHANGELOG.md#"+strings.ReplaceAll(stamp0014, ".", "")), n)
}

// link0014 is a `## Changelog` item linking target on the PR's branch.
func link0014(target string) string {
	return "- [CHANGELOG entry](https://github.com/" + repo0014 + "/blob/feature/" + target + ")"
}

// prBody0014 is a PR description with changelog as its `## Changelog`
// section, closing issue n.
func prBody0014(changelog string, n int) string {
	return fmt.Sprintf("> One clear summary of the change.\n\n## Background\n\nThis change is a test fixture.\n\n## Changelog\n\n%s\n\nCloses #%d\n", changelog, n)
}

// issue0014 is what gh prints for issue n read through fitness's filter:
// its body and label names.
func issue0014(t *testing.T, n int, body string, labels ...string) response {
	t.Helper()
	out, err := json.Marshal(map[string]any{"body": body, "labels": append([]string{}, labels...)})
	mustDo(t, err)
	return response{Match: []string{"repos/" + repo0014 + "/issues/" + itoa(n)}, Stdout: string(out) + "\n"}
}

// noChangelog0014 is GitHub listing no CHANGELOG.md change in a PR.
var noChangelog0014 = response{Match: []string{"/files"}}

// judge0014 runs pr-check on a pull_request event with title and body,
// GitHub answering from issues and the PR leaving the changelog alone.
func judge0014(t *testing.T, title, body string, issues ...response) (string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": append(issues, noChangelog0014)})
	pr := map[string]any{"number": 1, "title": title, "body": body}
	env := event0014(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr})
	return fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
}

func Test0014_1(t *testing.T) {
	t.Parallel()
	out, code := judge0014(t, "test(app): add one file", pr0014(11), issue0014(t, 11, plan0014("- [x] Write the tests."), "Plan"))
	sees(t, out, code, 0, "Passed.")
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"))})
	local, localCode := fitness(t, repo, rp.env, "branch-check")
	sees(t, local, localCode, 0, "pr-check #1", "plan-check #11", "https://github.com/"+repo0014+"/pull/1")
	if rp.called("gh", "-X") || rp.called("gh", "graphql") {
		t.Fatalf("local validation wrote GitHub state: %q", rp.calls("gh"))
	}
}

func Test0014_2(t *testing.T) {
	t.Parallel()
	out, code := judge0014(t, "added one file", pr0014(11), issue0014(t, 11, plan0014("- [x] Write the tests."), "Plan"))
	sees(t, out, code, 1, `"added one file"`)
}

func Test0014_3(t *testing.T) {
	t.Parallel()
	out, code := judge0014(t, "test(app): add one file", pr0014(11), issue0014(t, 11, plan0014("- [ ] Write the tests."), "Plan"))
	sees(t, out, code, 1, `11, which has an unchecked item: "Write the tests."`)
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [ ] Write the tests."), "Plan"))})
	local, localCode := fitness(t, repo, rp.env, "branch-check")
	sees(t, local, localCode, 1, `11, which has an unchecked item: "Write the tests."`)
}

func Test0014_4(t *testing.T) {
	t.Parallel()
	plan := plan0014("- [x] Fix the bug.") + "\nCloses #12\n"
	out, code := judge0014(t, "test(app): add one file", pr0014(11), issue0014(t, 11, plan, "Plan"))
	sees(t, out, code, 1, "#12")
}

func Test0014_5(t *testing.T) {
	t.Parallel()
	var open string
	for _, n := range []int{5, 6} {
		line, err := json.Marshal(map[string]any{"number": n, "title": "test(app): add one file", "body": pr0014(10 + n)})
		mustDo(t, err)
		open += string(line) + "\n"
	}
	done := plan0014("- [x] Write the tests.")
	rp := standIn(t, map[string][]response{"gh": {
		{Match: []string{"--paginate", "repos/" + repo0014 + "/pulls?state=open"}, Stdout: open},
		issue0014(t, 15, done, "Plan"),
		issue0014(t, 16, done, "Plan"),
		noChangelog0014,
	}})
	env := event0014(t, "workflow_dispatch", map[string]any{})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
	sees(t, out, code, 0, "pr-check #5", "pr-check #6")
}
