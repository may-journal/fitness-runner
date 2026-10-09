package requirements

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sandbox is the public repo where the GitHub commands run for real: tests
// open issues and pull requests there and read back what the commands did.
const sandbox = "may-journal/fitness-sandbox"

// gh runs the gh CLI as the test's own user and returns what it printed. It
// skips the test without a candidate installer, before touching GitHub.
func gh(t *testing.T, args ...string) string {
	t.Helper()
	installer(t)
	cmd := exec.Command("gh", args...)
	cmd.Env = userEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("gh %v: %v\n%s", args, err, stderrOf(err))
	}
	return strings.TrimSpace(string(out))
}

func stderrOf(err error) string {
	if exit, ok := err.(*exec.ExitError); ok {
		return string(exit.Stderr)
	}
	return ""
}

// newIssue opens a sandbox issue, closed as not planned when the test ends.
func newIssue(t *testing.T, body string, labels ...string) int {
	t.Helper()
	args := []string{"api", "repos/" + sandbox + "/issues", "-f", "title=" + t.Name(), "-f", "body=" + body, "-q", ".number"}
	for _, l := range labels {
		args = append(args, "-f", "labels[]="+l)
	}
	n, err := strconv.Atoi(gh(t, args...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = setIssue(n, "state=closed", "state_reason=not_planned") })
	return n
}

// setIssue edits sandbox issue n's fields, such as its body or state.
func setIssue(n int, fields ...string) error {
	args := []string{"api", "-X", "PATCH", "--silent", fmt.Sprintf("repos/%s/issues/%d", sandbox, n)}
	for _, f := range fields {
		args = append(args, "-f", f)
	}
	cmd := exec.Command("gh", args...)
	cmd.Env = userEnv()
	return cmd.Run()
}

// newPR opens a sandbox pull request from a fresh branch, closed and its
// branch deleted when the test ends.
func newPR(t *testing.T, title, body string) int {
	t.Helper()
	branch := "test/" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	sha := gh(t, "api", "repos/"+sandbox+"/git/ref/heads/main", "-q", ".object.sha")
	gh(t, "api", "repos/"+sandbox+"/git/refs", "-f", "ref=refs/heads/"+branch, "-f", "sha="+sha)
	t.Cleanup(func() {
		_ = exec.Command("gh", "api", "-X", "DELETE", "repos/"+sandbox+"/git/refs/heads/"+branch).Run()
	})
	content := base64.StdEncoding.EncodeToString([]byte(t.Name() + "\n"))
	gh(t, "api", "-X", "PUT", "repos/"+sandbox+"/contents/"+t.Name()+".txt", "-f", "message=test: add a file", "-f", "content="+content, "-f", "branch="+branch)
	n, err := strconv.Atoi(gh(t, "api", "repos/"+sandbox+"/pulls", "-f", "title="+title, "-f", "body="+body, "-f", "head="+branch, "-f", "base=main", "-q", ".number"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = exec.Command("gh", "api", "-X", "PATCH", fmt.Sprintf("repos/%s/pulls/%d", sandbox, n), "-f", "state=closed").Run()
	})
	return n
}

// listed waits until the sandbox's list at path shows number n, since
// GitHub's lists catch up a few seconds after an issue or PR opens.
func listed(t *testing.T, path string, n int) {
	t.Helper()
	for range 30 {
		for _, got := range strings.Fields(gh(t, "api", "repos/"+sandbox+"/"+path+"&per_page=100", "-q", ".[].number")) {
			if got == itoa(n) {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("#%d never appeared in %s", n, path)
}

// action runs `fitness-install -- <command>` the way its Actions step does,
// on the sandbox, with the event payload GitHub would deliver.
func action(t *testing.T, command, event string, payload any) (string, int) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	env := []string{"GITHUB_EVENT_NAME=" + event, "GITHUB_EVENT_PATH=" + path, "GITHUB_REPOSITORY=" + sandbox}
	return fitness(t, example(t, "happyRepo", nil), env, command)
}

// issueEvent is the payload of an `issues` event about issue n with body.
func issueEvent(n int, body string) map[string]any {
	return map[string]any{"action": "edited", "issue": map[string]any{"number": n, "title": "t", "body": body}}
}

// comment is one comment on a sandbox issue as a reader sees it.
type comment struct {
	Body   string `json:"body"`
	Hidden bool   `json:"isMinimized"`
}

// sandboxIssue is a sandbox issue's state, labels, and comments, oldest
// comment first.
type sandboxIssue struct {
	State    string    `json:"state"`
	Labels   []string  `json:"labels"`
	Comments []comment `json:"comments"`
}

// issueNow reads sandbox issue n as it stands.
func issueNow(t *testing.T, n int) sandboxIssue {
	t.Helper()
	owner, name, _ := strings.Cut(sandbox, "/")
	query := `query($o: String!, $r: String!, $n: Int!) { repository(owner: $o, name: $r) { issue(number: $n) {
		state labels(first: 20) { nodes { name } } comments(first: 50) { nodes { body isMinimized } } } } }`
	jq := `.data.repository.issue | {state, labels: [.labels.nodes[].name], comments: .comments.nodes}`
	var issue sandboxIssue
	mustDo(t, json.Unmarshal([]byte(gh(t, "api", "graphql", "-f", "query="+query, "-F", "o="+owner, "-F", "r="+name, "-F", "n="+strconv.Itoa(n), "--jq", jq)), &issue))
	return issue
}

// hasLabel reports whether labels includes name.
func hasLabel(labels []string, name string) bool {
	for _, l := range labels {
		if l == name {
			return true
		}
	}
	return false
}

// planBody is a Plan that follows the template, with one task.
func planBody(task string) string {
	return "> One clear pitch for this Plan.\n\n## Background\n\nThis Plan is a test fixture in the sandbox.\n\n## What needs to happen\n\n" + task + "\n"
}

// prBody is a PR description that follows the template and closes issue n.
func prBody(n int) string {
	return fmt.Sprintf("> One clear summary of the change.\n\n## Background\n\nThis change is a test fixture in the sandbox.\n\n## Changelog\n\n- Test: add one file.\n\nCloses #%d\n", n)
}
