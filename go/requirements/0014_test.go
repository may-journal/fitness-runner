package requirements

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// pr0014 is a PR description that follows the template and closes issue n.
func pr0014(n int) string {
	return fmt.Sprintf("> One clear summary of the change.\n\n## Background\n\nThis change is a test fixture.\n\n## Changelog\n\n- Test: add one file.\n\nCloses #%d\n", n)
}

// issue0014 is what gh prints for issue n read through fitness's filter:
// its body and label names.
func issue0014(t *testing.T, n int, body string, labels ...string) response {
	t.Helper()
	out, err := json.Marshal(map[string]any{"body": body, "labels": append([]string{}, labels...)})
	mustDo(t, err)
	return response{Match: []string{"repos/" + repo0014 + "/issues/" + itoa(n)}, Stdout: string(out) + "\n"}
}

// judge0014 runs pr-check on a pull_request event with title and body,
// GitHub answering from issues.
func judge0014(t *testing.T, title, body string, issues ...response) (string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": issues})
	pr := map[string]any{"number": 1, "title": title, "body": body}
	env := event0014(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr})
	return fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
}

func Test0014_1(t *testing.T) {
	t.Parallel()
	out, code := judge0014(t, "test(app): add one file", pr0014(11), issue0014(t, 11, plan0014("- [x] Write the tests."), "Plan"))
	sees(t, out, code, 0, "Passed.")
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
	}})
	env := event0014(t, "workflow_dispatch", map[string]any{})
	out, code := fitness(t, example(t, "happyRepo", nil), append(rp.env, env...), "pr-check")
	sees(t, out, code, 0, "pr-check #5", "pr-check #6")
}
