// Package aftereffect holds what a GitHub command does after its checks judge
// a blob: comment a verdict, label it, hide outdated verdicts, or reopen an
// issue. Each command names its effect by code reference, so every GitHub
// reaction lives here and the checks stay pure judgments.
package aftereffect

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Target is one issue or pull request a GitHub command judges, as the
// Actions event or the API returns it.
type Target struct {
	// ID is the issue's database id, which a duplicate close names.
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	// StateReason and ClosedAt are set on a closed issue's event payload.
	StateReason string `json:"state_reason"`
	ClosedAt    string `json:"closed_at"`
	// ClosedBy is the event sender who closed the issue.
	ClosedBy string `json:"-"`
	// Labels are the issue's label names, as a GitHub read lists them; the
	// event payload's label objects never fill them.
	Labels []string `json:"labelNames"`
	// EventLabels are the label objects an event payload carries; nil when
	// it carries none.
	EventLabels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	// Kind is the issue kind label the target is judged as, such as Plan.
	Kind string `json:"-"`
	// Gone marks an issue a read found deleted or moved away.
	Gone bool `json:"-"`
	// EntryFiles are a local branch's changed changelog entry files against
	// its PR base; nil reads them from GitHub.
	EntryFiles *[]FileChange `json:"-"`
	// Comment is set when the target is one comment on issue or PR Number.
	Comment *Comment `json:"-"`
	// State is open or closed, as the event payload and the issue list
	// spell it.
	State string `json:"state"`
	// Parent is the issue's parent issue number, 0 for none, as the issue
	// list reads it; an event payload never sets it.
	Parent int `json:"parentNumber"`
	// Opener is the login that opened the issue, as REST spells it, when the
	// issue list reads it.
	Opener string `json:"opener"`
	// Dupes are the older issues dupe-check found the issue resembles, best
	// first; CloseDupe closes it as a duplicate of the first.
	Dupes     []Dupe `json:"-"`
	CloseDupe bool   `json:"-"`
}

// Dupe is an older issue a new one resembles, and how alike they read.
type Dupe struct {
	ID     int64
	Number int
	Score  float64
	// Open marks an older issue still open, the only kind a new one closes
	// as a duplicate of.
	Open bool
}

// FileChange is one file a pull request changes: its path, GitHub's status
// for it (added, modified, removed, renamed), and its unified diff.
type FileChange struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Patch    string `json:"patch"`
}

// Closer is the pull request whose merge closed an issue; both fields are
// empty when a person or a commit closed it.
type Closer struct {
	Author string `json:"author"`
	Merger string `json:"merger"`
}

// Verdict is one issue-check comment. ID is the REST id for edits; NodeID is
// the GraphQL id for hiding.
type Verdict struct {
	ID       int64  `json:"id"`
	NodeID   string `json:"nodeId"`
	Body     string `json:"body"`
	IsHidden bool   `json:"isHidden"`
	Author   author `json:"author"`
}

// author is a comment's writer as GraphQL reports it: an app's login comes
// without the [bot] suffix REST gives it, and its type is Bot.
type author struct {
	Login string `json:"login"`
	Type  string `json:"__typename"`
}

// rest is the login as REST and BotLogin spell it.
func (a author) rest() string {
	if a.Type == "Bot" {
		return a.Login + "[bot]"
	}
	return a.Login
}

// Effect is a command's reaction to one judged target; errs are the checks'
// findings, empty when the target passed.
type Effect func(gh Client, t Target, errs []string) error

// Client reaches one repository through the gh CLI.
type Client struct {
	repo   string
	ctx    context.Context
	issues map[int]*issueResult
	// mu guards issues, so callers may read issues in parallel.
	mu *sync.Mutex
}

// issueResult is one cached issue read; once makes parallel callers of the
// same issue share a single read.
type issueResult struct {
	once   sync.Once
	body   string
	labels []string
	err    error
}

// ReadOnlyClient shares a bounded context and caches issue reads for local
// judgments. Workflow clients retain their existing effects.
func ReadOnlyClient(ctx context.Context, repo string) Client {
	return Client{repo: repo, ctx: ctx, issues: map[int]*issueResult{}, mu: &sync.Mutex{}}
}

// Seed records issue n as already read, so a ReadOnlyClient answers from it
// instead of GitHub. A client without a cache ignores it.
func (g Client) Seed(n int, body string, labels []string) {
	if g.issues == nil {
		return
	}
	r := g.entry(n)
	r.once.Do(func() { r.body, r.labels = body, labels })
}

// entry returns issue n's cache slot, adding an empty one.
func (g Client) entry(n int) *issueResult {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.issues[n]
	if !ok {
		r = &issueResult{}
		g.issues[n] = r
	}
	return r
}

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
	return g.gh(append([]string{"api"}, args...)...)
}

// gh runs gh within the client's context, passing its stderr through, and
// classifies a failure with GHErr.
func (g Client) gh(args ...string) ([]byte, error) {
	out, _, err := g.run(args...)
	return out, err
}

// run is gh that also returns the run's stderr, for callers that classify
// a failure further.
func (g Client) run(args ...string) ([]byte, string, error) {
	ctx := g.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Stderr = os.Stderr
	o := checkkit.Exec(cmd)
	if err := GHErr(ctx, o); err != nil {
		return nil, o.Stderr, err
	}
	return []byte(o.Stdout), o.Stderr, nil
}

// ErrIssueGone marks a GraphQL issue query that found no such issue in the
// repository, as after a transfer to another repo.
var ErrIssueGone = errors.New("the issue no longer exists in this repository")

// goneText is the stderr gh prints when a GraphQL repository.issue lookup finds no
// issue with that number.
const goneText = "could not resolve to an issue with the number of"

// issueQuery runs a GraphQL query on repository.issue, wrapping its error
// with ErrIssueGone when gh says the issue is gone. Only stderr is read, so
// comment text on stdout never counts.
func (g Client) issueQuery(args ...string) ([]byte, error) {
	out, stderr, err := g.run(args...)
	if err != nil && strings.Contains(strings.ToLower(stderr), goneText) {
		return nil, fmt.Errorf("%w: %w", ErrIssueGone, err)
	}
	return out, err
}

// goneREST reports whether a REST read's stderr says the issue is not there.
func goneREST(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "http 404") || strings.Contains(s, "http 410")
}

// ghEnvText is gh output that names its environment, not the request: no
// login or a rejected token, or a network that cannot reach GitHub.
var ghEnvText = []string{
	"gh auth login", "http 401", "bad credentials", "authentication required",
	"dial tcp", "no such host", "could not resolve host", "connection refused",
	"network is unreachable", "i/o timeout", "tls handshake timeout", "error connecting to",
}

// GHEnvironment reports whether a failed gh run says nothing about the
// repository: gh could not start, a signal ended it, or its output shows an
// auth or network failure.
func GHEnvironment(o checkkit.Outcome) bool {
	if o.EnvErr("gh") != nil {
		return true
	}
	text := strings.ToLower(o.Stdout + "\n" + o.Stderr)
	return slices.ContainsFunc(ghEnvText, func(s string) bool { return strings.Contains(text, s) })
}

// GHErr is a gh run's error: nil on success, a checkkit.Internal error when
// ctx ran out, gh is missing, or GHEnvironment holds, and the run's own
// error otherwise, such as a 404 or a 422.
func GHErr(ctx context.Context, o checkkit.Outcome) error {
	switch {
	case o.Err == nil:
		return nil
	case ctx.Err() != nil:
		return checkkit.Internal(ctx.Err())
	case o.SpawnErr != nil:
		return checkkit.ToolMissing("gh")
	case GHEnvironment(o):
		return checkkit.Internal(o.Err)
	}
	return o.Err
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

// Issue reads issue n's body and labels. A ReadOnlyClient reads each issue
// once, and is safe to call from several goroutines.
func (g Client) Issue(n int) (string, []string, error) {
	if g.issues == nil {
		return g.readIssue(n)
	}
	r := g.entry(n)
	r.once.Do(func() { r.body, r.labels, r.err = g.readIssue(n) })
	return r.body, r.labels, r.err
}

// IssueView is one issue with its comments, as `gh issue view` reads it.
type IssueView struct {
	URL      string         `json:"url"`
	Body     string         `json:"body"`
	Labels   []string       `json:"labels"`
	Comments []IssueComment `json:"comments"`
}

// IssueComment is one comment on an issue: its text, its author's login,
// and the author's association with the repo as GitHub reports it, such as
// OWNER, MEMBER or NONE.
type IssueComment struct {
	Body        string `json:"body"`
	Author      string `json:"author"`
	Association string `json:"association"`
}

// Repo is the owner/repo the issue's URL names, or "" for another URL.
func (v IssueView) Repo() string {
	path, ok := strings.CutPrefix(v.URL, "https://github.com/")
	parts := strings.Split(path, "/")
	if !ok || len(parts) != 4 || parts[2] != "issues" {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

// ViewIssue reads issue n of repo, an owner/repo slug, or of the working
// directory's repository when repo is empty. It reads the body, labels and
// comments with their authors in one `gh issue view`. The stderr of gh stays out of the
// caller's output.
func ViewIssue(ctx context.Context, repo, n string) (IssueView, error) {
	args := []string{"issue", "view", n}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	args = append(args, "--json", "url,body,labels,comments",
		"--jq", "{url, body, labels: [.labels[].name], comments: [.comments[] | {body, author: .author.login, association: .authorAssociation}]}")
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		return IssueView{}, err
	}
	var v IssueView
	err = json.Unmarshal(out, &v)
	return v, err
}

// readIssue reads issue n's body and labels. A 404 or 410, an issue deleted
// or never there, wraps ErrIssueGone.
func (g Client) readIssue(n int) (string, []string, error) {
	out, stderr, err := g.run("api", "--jq", "{body: (.body // \"\"), labels: [.labels[].name]}", "repos/"+g.repo+"/issues/"+strconv.Itoa(n))
	if err != nil && goneREST(stderr) {
		return "", nil, fmt.Errorf("%w: %w", ErrIssueGone, err)
	}
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

// PR reads pull request n's number, title and body as GitHub holds them now.
func (g Client) PR(n int) (Target, error) {
	out, err := g.api("--jq", "{number, title, body: (.body // \"\")}", "pulls/"+strconv.Itoa(n))
	if err != nil {
		return Target{}, err
	}
	var t Target
	err = json.Unmarshal(out, &t)
	return t, err
}

func (g Client) OpenPRs() ([]Target, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {number, title, body: (.body // \"\")}", "pulls?state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}

// DirFiles returns the files pull request n changes directly under dir,
// with each one's status and diff.
func (g Client) DirFiles(n int, dir string) ([]FileChange, error) {
	jq := fmt.Sprintf(`.[] | select(.filename | startswith(%q)) | {filename, status, patch: (.patch // "")}`, dir+"/")
	out, err := g.api("--paginate", "--jq", jq, "pulls/"+strconv.Itoa(n)+"/files?per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[FileChange](out)
}

// ReleasedFiles returns the newest GitHub release's tag and the files
// directly under dir at that tag. The tag is empty when the repo has no
// release, and the files are none when the tag has no such folder.
func (g Client) ReleasedFiles(dir string) (string, []string, error) {
	out, err := g.api("--jq", ".[0].tag_name // empty", "releases?per_page=1")
	tag := strings.TrimSpace(string(out))
	if err != nil || tag == "" {
		return "", nil, err
	}
	out, err = g.api("--jq", `.[] | select(.type == "dir") | .name`, "contents?ref="+tag)
	if err != nil || !slices.Contains(strings.Fields(string(out)), dir) {
		return tag, nil, err
	}
	out, err = g.api("--jq", `.[] | select(.type == "file") | .path`, "contents/"+dir+"?ref="+tag)
	return tag, strings.Fields(string(out)), err
}

// Repo is the owner/name the client reaches.
func (g Client) Repo() string {
	return g.repo
}

// OpenLabeled returns every open issue carrying label, with its labels.
func (g Client) OpenLabeled(label string) ([]Target, error) {
	jq := `.[] | select(.pull_request == null) | {number, body: (.body // ""), labelNames: [.labels[].name]}`
	out, err := g.api("--paginate", "--jq", jq, "issues?labels="+url.QueryEscape(label)+"&state=open&per_page=100")
	if err != nil {
		return nil, err
	}
	return decodeLines[Target](out)
}

// SubIssues returns issue n's sub-issues, each as owner/name#number, in the
// order GitHub lists them.
func (g Client) SubIssues(n int) ([]string, error) {
	jq := `.[] | {number, repo: (.repository_url | sub("^.*/repos/"; ""))}`
	out, err := g.api("--paginate", "--jq", jq, "issues/"+strconv.Itoa(n)+"/sub_issues?per_page=100")
	if err != nil {
		return nil, err
	}
	subs, err := decodeLines[struct {
		Number int    `json:"number"`
		Repo   string `json:"repo"`
	}](out)
	refs := make([]string, len(subs))
	for i, s := range subs {
		refs[i] = s.Repo + "#" + strconv.Itoa(s.Number)
	}
	return refs, err
}

// verdictsQuery lists an issue's comments with the hidden state, which only
// GraphQL exposes.
const verdictsQuery = `query($owner: String!, $name: String!, $number: Int!, $endCursor: String) {
  repository(owner: $owner, name: $name) {
    issue(number: $number) {
      comments(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes { id databaseId body isMinimized author { login __typename } }
      }
    }
  }
}`

// Verdicts returns the issue-check verdict comments for kind, oldest first.
// A verdict is known by its body-version marker, and counts only when the
// fitness bot wrote it, so a marker another author pastes is never edited,
// hidden, or taken as the live verdict.
func (g Client) Verdicts(n int, kind string) ([]Verdict, error) {
	owner, name, _ := strings.Cut(g.repo, "/")
	var tests []string
	for _, p := range MarkerPrefixes(kind) {
		tests = append(tests, fmt.Sprintf("contains(%q)", p))
	}
	jq := `.data.repository.issue.comments.nodes[]
		| select(.body | ` + strings.Join(tests, " or ") + `)
		| {id: .databaseId, nodeId: .id, body, isHidden: .isMinimized, author}`
	out, err := g.issueQuery("api", "graphql", "--paginate",
		"-f", "query="+verdictsQuery, "-F", "owner="+owner, "-F", "name="+name,
		"-F", "number="+strconv.Itoa(n), "--jq", jq)
	if err != nil {
		return nil, err
	}
	found, err := decodeLines[Verdict](out)
	return byBot(found, func(v Verdict) string { return v.Author.rest() }), err
}

// byBot keeps the items the fitness bot wrote.
func byBot[T any](items []T, writer func(T) string) []T {
	var kept []T
	for _, item := range items {
		if writer(item) == BotLogin() {
			kept = append(kept, item)
		}
	}
	return kept
}

// authoredBody is one comment's body and its REST author login.
type authoredBody struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

// CommentBodies returns the bodies of the comments the fitness bot wrote on
// issue n, so a marker another author pastes never counts. It selects
// objects, not bare bodies: gh prints a string result raw rather than as
// JSON, which would not decode.
func (g Client) CommentBodies(n int) ([]string, error) {
	out, err := g.api("--paginate", "--jq", ".[] | {body: (.body // \"\"), author: .user.login}", "issues/"+strconv.Itoa(n)+"/comments?per_page=100")
	if err != nil {
		return nil, err
	}
	all, err := decodeLines[authoredBody](out)
	comments := byBot(all, func(c authoredBody) string { return c.Author })
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
	out, err := g.issueQuery("api", "graphql",
		"-f", "query="+closingPRQuery, "-F", "owner="+owner, "-F", "name="+name, "-F", "n="+strconv.Itoa(n),
		"--jq", closingPRFilter)
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
	_, err := g.gh("api", "graphql", "--silent",
		"-f", "query=mutation($id: ID!) { minimizeComment(input: {subjectId: $id, classifier: OUTDATED}) { clientMutationId } }",
		"-f", "id="+nodeID)
	return err
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
