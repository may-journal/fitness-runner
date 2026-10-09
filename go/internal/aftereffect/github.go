// Package aftereffect holds what a GitHub command does after its checks judge
// a blob: comment a verdict, label it, hide outdated verdicts, or reopen an
// issue. Each command names its effect by code reference, so every GitHub
// reaction lives here and the checks stay pure judgments.
package aftereffect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Target is one issue or pull request a GitHub command judges, as the
// Actions event or the API returns it.
type Target struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	// StateReason and ClosedAt are set on a closed issue's event payload.
	StateReason string `json:"state_reason"`
	ClosedAt    string `json:"closed_at"`
	// ClosedBy is the event sender who closed the issue.
	ClosedBy string `json:"-"`
}

// Closer is the pull request whose merge closed an issue; both fields are
// empty when a person or a commit closed it.
type Closer struct {
	Author string `json:"author"`
	Merger string `json:"merger"`
}

// Verdict is one plan-check comment. ID is the REST id for edits; NodeID is
// the GraphQL id for hiding.
type Verdict struct {
	ID       int64  `json:"id"`
	NodeID   string `json:"nodeId"`
	Body     string `json:"body"`
	IsHidden bool   `json:"isHidden"`
}

// Effect is a command's reaction to one judged target; errs are the checks'
// findings, empty when the target passed.
type Effect func(gh Client, t Target, errs []string) error

// Client reaches one repository through the gh CLI.
type Client struct{ repo string }

// NewClient targets the repository the workflow runs in.
func NewClient() (Client, error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return Client{}, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}
	return Client{repo: repo}, nil
}

// api runs `gh api` against a path under the repository.
func (g Client) api(args ...string) ([]byte, error) {
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

func (g Client) Issue(n int) (string, []string, error) {
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

func (g Client) OpenPRs() ([]Target, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {number, title, body: (.body // \"\")}", "pulls?state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}

func (g Client) OpenPlans() ([]Target, error) {
	out, err := g.api("--paginate", "--jq", ".[] | select(.pull_request == null) | {number, body: (.body // \"\")}", "issues?labels=Plan&state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}

// verdictsQuery lists an issue's comments with the hidden state, which only
// GraphQL exposes.
const verdictsQuery = `query($owner: String!, $name: String!, $number: Int!, $endCursor: String) {
  repository(owner: $owner, name: $name) {
    issue(number: $number) {
      comments(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes { id databaseId body isMinimized }
      }
    }
  }
}`

// Verdicts returns the plan-check verdict comments, oldest first. A verdict
// is known by its body-version marker, whoever's token posted it.
func (g Client) Verdicts(n int) ([]Verdict, error) {
	owner, name, _ := strings.Cut(g.repo, "/")
	jq := `.data.repository.issue.comments.nodes[]
		| select(.body | contains("fitness:plan-structure:"))
		| {id: .databaseId, nodeId: .id, body, isHidden: .isMinimized}`
	cmd := exec.Command("gh", "api", "graphql", "--paginate",
		"-f", "query="+verdictsQuery, "-F", "owner="+owner, "-F", "name="+name,
		"-F", "number="+strconv.Itoa(n), "--jq", jq)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return decodeLines[Verdict](out)
}

// CommentBodies selects objects, not bare bodies: gh prints a string result
// raw rather than as JSON, which would not decode.
func (g Client) CommentBodies(n int) ([]string, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {body: (.body // \"\")}", "issues/"+strconv.Itoa(n)+"/comments?per_page=100")
	if err != nil {
		return nil, err
	}
	comments, err := decodeLines[Target](out)
	bodies := make([]string, len(comments))
	for i, c := range comments {
		bodies[i] = c.Body
	}
	return bodies, err
}

func (g Client) Comment(n int, body string) error {
	_, err := g.api("-X", "POST", "-f", "body="+body, "issues/"+strconv.Itoa(n)+"/comments")
	return err
}

func (g Client) Reopen(n int) error {
	_, err := g.api("-X", "PATCH", "-f", "state=open", "issues/"+strconv.Itoa(n))
	return err
}

// closingPRQuery reads the latest closed event's closer. Only GraphQL
// exposes it; REST names the actor but not the closing pull request. A merge
// records either the PR or its merge commit as the closer, so a commit is
// traced back to its PR.
const closingPRQuery = `query($owner: String!, $name: String!, $n: Int!) {
  repository(owner: $owner, name: $name) {
    issue(number: $n) {
      timelineItems(itemTypes: CLOSED_EVENT, last: 1) {
        nodes { ... on ClosedEvent { closer {
          ... on PullRequest { author { login } mergedBy { login } }
          ... on Commit { associatedPullRequests(first: 1) { nodes { author { login } mergedBy { login } } } }
        } } }
      }
    }
  }
}`

// closingPRFilter picks the PR from either closer shape, or nothing.
const closingPRFilter = `.data.repository.issue.timelineItems.nodes[0].closer as $c
  | ($c.associatedPullRequests.nodes[0] // $c // {})
  | {author: (.author.login // ""), merger: (.mergedBy.login // "")}`

func (g Client) ClosingPR(n int) (Closer, error) {
	owner, name, _ := strings.Cut(g.repo, "/")
	cmd := exec.Command("gh", "api", "graphql",
		"-f", "query="+closingPRQuery, "-F", "owner="+owner, "-F", "name="+name, "-F", "n="+strconv.Itoa(n),
		"--jq", closingPRFilter)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return Closer{}, err
	}
	var c Closer
	err = json.Unmarshal(out, &c)
	return c, err
}

func (g Client) EditComment(id int64, body string) error {
	_, err := g.api("-X", "PATCH", "-f", "body="+body, "issues/comments/"+strconv.FormatInt(id, 10))
	return err
}

// HideComment collapses a comment as outdated.
func (g Client) HideComment(nodeID string) error {
	cmd := exec.Command("gh", "api", "graphql", "--silent",
		"-f", "query=mutation($id: ID!) { minimizeComment(input: {subjectId: $id, classifier: OUTDATED}) { clientMutationId } }",
		"-f", "id="+nodeID)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EnsureLabel creates the repo label when it is missing.
func (g Client) EnsureLabel(name, color string) error {
	if _, err := exec.Command("gh", "api", "--silent", "repos/"+g.repo+"/labels/"+name).Output(); err == nil {
		return nil
	}
	_, err := g.api("-X", "POST", "-f", "name="+name, "-f", "color="+color, "labels")
	return err
}

func (g Client) AddLabels(n int, names []string) error {
	args := []string{"-X", "POST", "--silent"}
	for _, l := range names {
		args = append(args, "-f", "labels[]="+l)
	}
	_, err := g.api(append(args, "issues/"+strconv.Itoa(n)+"/labels")...)
	return err
}

func (g Client) RemoveLabel(n int, name string) error {
	_, err := g.api("-X", "DELETE", "--silent", "issues/"+strconv.Itoa(n)+"/labels/"+name)
	return err
}
