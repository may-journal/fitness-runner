package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// docChecks are the prose and markdown checks the plan-check and pr-check
// subcommands run over an issue or PR body. They read the body through
// --body-file, with --root at the checkout so their config resolves.
var docChecks = []string{
	"prose-budget",
	"text-readability",
	"markdown-no-bold-italic",
	"mermaid-callouts",
	"mermaid-callout-why",
	"mermaid-diagram-prose",
	"mermaid-diagram-table-gap",
	"mermaid-legend",
	"cspell",
}

// target is one issue or PR body the workflow subcommands validate.
type target struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// githubAPI is the slice of GitHub the workflow subcommands need; the real
// one shells out to the gh CLI, and tests swap in a fake.
type githubAPI interface {
	issue(n int) (body string, labels []string, err error)
	openPRs() ([]target, error)
	openPlans() ([]target, error)
	commentBodies(n int) ([]string, error)
	comment(n int, body string) error
}

// bodyChecker runs check binaries against a body. exec is swappable so tests
// can fake the binaries.
type bodyChecker struct {
	root string
	exec func(name string, args []string, stdin string) []byte
}

// newBodyChecker runs the installed fitness-check-* binaries from root.
func newBodyChecker(root string) bodyChecker {
	return bodyChecker{root: root, exec: execCheck}
}

// execCheck runs fitness-check-<name> and returns its stdout. A failing check
// still prints its JSON result, so a non-zero exit is not an error here.
func execCheck(name string, args []string, stdin string) []byte {
	bin, err := findCheckBinary(name)
	if err != nil {
		return nil
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, os.Stderr
	_ = cmd.Run()
	return stdout.Bytes()
}

// errorsOf runs one check and returns its errors; a check that prints no
// result fails closed with an error naming it.
func (c bodyChecker) errorsOf(name string, args []string, stdin string) []string {
	var res checkkit.Result
	if err := json.Unmarshal(lastJSONLine(c.exec(name, args, stdin)), &res); err != nil {
		return []string{fmt.Sprintf("%s: no result from the check", name)}
	}
	return res.Errors
}

// docErrors runs the prose and markdown checks over the body in bodyFile.
func (c bodyChecker) docErrors(bodyFile string) []string {
	var errs []string
	for _, name := range docChecks {
		errs = append(errs, c.errorsOf(name, []string{"--root", c.root, "--body-file", bodyFile}, "")...)
	}
	return errs
}

// closedIssues lists the issues a body closes, parsed by pr-closes-issue so
// the Plan side and PR side share one keyword set.
func (c bodyChecker) closedIssues(bodyFile string) []int {
	var nums []int
	_ = json.Unmarshal(lastJSONLine(c.exec("pr-closes-issue", []string{"--emit-closed", "--body-file", bodyFile}, "")), &nums)
	return nums
}

// writeBody saves body to a temp file for the --body-file checks; the caller
// removes it.
func writeBody(body string) (string, error) {
	f, err := os.CreateTemp("", "fitness-body-*.md")
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(body)
	return f.Name(), err
}

// ghEvent is the part of the Actions event payload the subcommands read.
type ghEvent struct {
	name        string
	PullRequest *target `json:"pull_request"`
	Issue       *target `json:"issue"`
}

// readEvent loads the triggering event from the Actions environment.
func readEvent() (ghEvent, error) {
	ev := ghEvent{name: os.Getenv("GITHUB_EVENT_NAME")}
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return ev, fmt.Errorf("read the event payload: %w", err)
	}
	return ev, json.Unmarshal(data, &ev)
}

// ghClient implements githubAPI over the gh CLI for one repository.
type ghClient struct{ repo string }

// newGHClient targets the repository the workflow runs in.
func newGHClient() (ghClient, error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return ghClient{}, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}
	return ghClient{repo: repo}, nil
}

// api runs `gh api` against a path under the repository.
func (g ghClient) api(args ...string) ([]byte, error) {
	args[len(args)-1] = "repos/" + g.repo + "/" + args[len(args)-1]
	cmd := exec.Command("gh", append([]string{"api"}, args...)...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// decodeLines decodes the --jq output of gh: one JSON object per line.
func decodeLines[T any](out []byte) ([]T, error) {
	var vals []T
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var v T
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func (g ghClient) issue(n int) (string, []string, error) {
	out, err := g.api("--jq", "{body: (.body // \"\"), labels: [.labels[].name]}", "issues/"+strconv.Itoa(n))
	if err != nil {
		return "", nil, err
	}
	var v struct {
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
	}
	err = json.Unmarshal(out, &v)
	return v.Body, v.Labels, err
}

func (g ghClient) openPRs() ([]target, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {number, title, body: (.body // \"\")}", "pulls?state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[target](out)
}

func (g ghClient) openPlans() ([]target, error) {
	out, err := g.api("--paginate", "--jq", ".[] | select(.pull_request == null) | {number, body: (.body // \"\")}", "issues?labels=Plan&state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[target](out)
}

// commentBodies selects objects, not bare bodies: gh prints a string result
// raw rather than as JSON, which would not decode.
func (g ghClient) commentBodies(n int) ([]string, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {body: (.body // \"\")}", "issues/"+strconv.Itoa(n)+"/comments?per_page=100")
	if err != nil {
		return nil, err
	}
	comments, err := decodeLines[target](out)
	bodies := make([]string, len(comments))
	for i, c := range comments {
		bodies[i] = c.Body
	}
	return bodies, err
}

func (g ghClient) comment(n int, body string) error {
	_, err := g.api("-X", "POST", "-f", "body="+body, "issues/"+strconv.Itoa(n)+"/comments")
	return err
}
