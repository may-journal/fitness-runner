package aftereffect

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checklist"
	"github.com/may-journal/fitness-runner/go/internal/epic"
)

// Ref is one issue in one repository.
type Ref struct {
	Repo   string
	Number int
}

// String is the owner/name#number spelling.
func (r Ref) String() string {
	return r.Repo + "#" + strconv.Itoa(r.Number)
}

// parseRef reads an owner/name#number reference; a bare #number is in repo.
func parseRef(s, repo string) (Ref, error) {
	where, num, ok := strings.Cut(strings.ToLower(epic.Key(s, repo)), "#")
	n, err := strconv.Atoi(num)
	if !ok || err != nil {
		return Ref{}, fmt.Errorf("not an issue reference: %q", s)
	}
	return Ref{Repo: where, Number: n}, nil
}

// State is an issue's state as REST reads it: open or closed, why it
// closed, and its body.
type State struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
	Body   string `json:"body"`
}

// SettleClose reacts to a closed issue: it reopens one closed with
// unfinished items, and otherwise closes the parents it completes.
func SettleClose(gh Client, t Target, errs []string) error {
	if err := ReopenUnfinished(gh, t, errs); err != nil || len(errs) > 0 {
		return err
	}
	return CloseParents(gh, Ref{Repo: strings.ToLower(gh.repo), Number: t.Number})
}

// CloseParents closes every open parent whose children are all closed as
// completed, then repeats for each parent it closed, so a grandparent
// follows. A parent is an issue that lists child as a sub-issue or in its
// `## What needs to happen` list.
func CloseParents(gh Client, child Ref) error {
	s := sweep{gh: gh, closed: map[Ref]bool{child: true}}
	return s.climb(child)
}

// sweep is one climb up the parents of a closed issue. closed holds the
// issues known closed as completed: the one that started it and each
// parent it closed, which also stops a cycle of parents.
type sweep struct {
	gh     Client
	closed map[Ref]bool
}

// climb closes the parents child completes, then climbs from each.
func (s sweep) climb(child Ref) error {
	parents, err := s.gh.in(child.Repo).parents(child.Number)
	if err != nil {
		return err
	}
	for _, p := range parents {
		if err := s.visit(p, child); err != nil {
			return err
		}
	}
	return nil
}

// visit closes parent if child finishes it, and then climbs from it.
func (s sweep) visit(parent, child Ref) error {
	if s.closed[parent] {
		return nil
	}
	done, err := s.closeIfDone(parent, child)
	if err != nil || !done {
		return err
	}
	s.closed[parent] = true
	return s.climb(parent)
}

// closeIfDone closes parent when child is one of its children and every
// child is closed as completed. It reports whether it closed the parent.
func (s sweep) closeIfDone(parent, child Ref) (bool, error) {
	pc := s.gh.in(parent.Repo)
	children, err := candidates(pc, parent, child)
	if err != nil || children == nil || !s.allDone(parent, children) {
		return false, err
	}
	return true, finish(pc, parent, children)
}

// finish comments each child's number and close reason on parent, then
// closes it as completed.
func finish(pc Client, parent Ref, children []Ref) error {
	lines := make([]string, len(children))
	for i, c := range children {
		lines[i] = fmt.Sprintf("- %s closed as completed", c)
	}
	body := "Closing this issue: every child is done.\n\n" + strings.Join(lines, "\n")
	if err := pc.Comment(parent.Number, body); err != nil {
		return err
	}
	return pc.Close(parent.Number)
}

// candidates are the children of parent when it is open, lists child, and
// has no unchecked box of its own; otherwise nil.
func candidates(pc Client, parent, child Ref) ([]Ref, error) {
	st, err := pc.State(parent.Number)
	if err != nil || st.State != "open" {
		return nil, err
	}
	children, err := childrenOf(pc, parent, st.Body)
	if err != nil {
		return nil, err
	}
	if !listed(st.Body, parent.Repo, children, child) {
		return nil, nil
	}
	return children, nil
}

// listed reports whether children holds child and body has no unchecked box
// of its own.
func listed(body, repo string, children []Ref, child Ref) bool {
	return has(children, child) && !ownUnchecked(body, repo, children)
}

// allDone reports whether every child is closed as completed. A child that
// cannot be read leaves the parent open, with a note.
func (s sweep) allDone(parent Ref, children []Ref) bool {
	for _, c := range children {
		if !s.done(parent, c) {
			return false
		}
	}
	return true
}

// done reports whether child c of parent is closed as completed.
func (s sweep) done(parent, c Ref) bool {
	if s.closed[c] {
		return true
	}
	cs, err := s.gh.in(c.Repo).State(c.Number)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness close-check: cannot read child %s, so %s stays open: %v\n", c, parent, err)
		return false
	}
	return cs.State == "closed" && cs.Reason == "completed"
}

// has reports whether refs holds ref.
func has(refs []Ref, ref Ref) bool {
	for _, r := range refs {
		if r == ref {
			return true
		}
	}
	return false
}

// childrenOf is the parent's sub-issues followed by the issues its list
// names, each once.
func childrenOf(pc Client, parent Ref, body string) ([]Ref, error) {
	subs, err := pc.SubIssues(parent.Number)
	if err != nil {
		return nil, err
	}
	var out []Ref
	for _, s := range append(subs, epic.Refs(body)...) {
		r, err := parseRef(s, parent.Repo)
		if err != nil {
			return nil, err
		}
		if r != parent {
			out = withRef(out, r)
		}
	}
	return out, nil
}

// withRef appends ref to refs unless it is already there.
func withRef(refs []Ref, ref Ref) []Ref {
	if has(refs, ref) {
		return refs
	}
	return append(refs, ref)
}

// ownUnchecked reports whether body holds an unchecked box that is not one
// of its children, which Close check would reopen the parent for.
func ownUnchecked(body, repo string, children []Ref) bool {
	for _, item := range checklist.Unchecked(body) {
		ref, err := parseRef(epic.Lead(item), repo)
		if err != nil || !has(children, ref) {
			return true
		}
	}
	return false
}

// in is the client for another repository; its issue cache stays behind.
func (g Client) in(repo string) Client {
	g.repo, g.issues = repo, nil
	return g
}

// parents are the issues that may list issue n: its sub-issue parent and
// the open issues in this repo whose body mentions it. The caller confirms
// each against its children.
func (g Client) parents(n int) ([]Ref, error) {
	var out []Ref
	sub, err := g.subParent(n)
	if err != nil {
		return nil, err
	}
	if sub != nil {
		out = append(out, *sub)
	}
	hits, err := g.mentions(n)
	for _, h := range hits {
		out = withRef(out, h)
	}
	return out, err
}

// subParent is issue n's parent issue, or nil when it has none.
func (g Client) subParent(n int) (*Ref, error) {
	out, stderr, err := g.run("api", "--jq", `{number, repo: (.repository_url | sub("^.*/repos/"; ""))}`, "repos/"+g.repo+"/issues/"+strconv.Itoa(n)+"/parent")
	if err != nil {
		if goneREST(stderr) {
			return nil, nil
		}
		return nil, err
	}
	var p struct {
		Number int    `json:"number"`
		Repo   string `json:"repo"`
	}
	if err := json.Unmarshal(out, &p); err != nil {
		return nil, err
	}
	return &Ref{Repo: strings.ToLower(p.Repo), Number: p.Number}, nil
}

// mentions are the open issues in this repo whose body mentions issue n.
func (g Client) mentions(n int) ([]Ref, error) {
	q := fmt.Sprintf(`repo:%s is:issue is:open "#%d" in:body`, g.repo, n)
	out, err := g.gh("api", "--paginate", "--jq", `.items[] | {number}`, "search/issues?per_page=100&q="+url.QueryEscape(q))
	if err != nil {
		return nil, err
	}
	hits, err := decodeLines[struct {
		Number int `json:"number"`
	}](out)
	refs := make([]Ref, len(hits))
	for i, h := range hits {
		refs[i] = Ref{Repo: strings.ToLower(g.repo), Number: h.Number}
	}
	return refs, err
}

// State reads issue n's state, close reason and body.
func (g Client) State(n int) (State, error) {
	out, err := g.api("--jq", `{state, reason: (.state_reason // ""), body: (.body // "")}`, "issues/"+strconv.Itoa(n))
	if err != nil {
		return State{}, err
	}
	var s State
	err = json.Unmarshal(out, &s)
	return s, err
}

// Close closes issue n as completed.
func (g Client) Close(n int) error {
	_, err := g.api("-X", "PATCH", "-f", "state=closed", "-f", "state_reason=completed", "issues/"+strconv.Itoa(n))
	return err
}
