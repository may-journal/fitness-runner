// Command fitness-check-workflow-ref-exists fails a workflow that calls a
// may-journal workflow or action at a tag or sha where that path does not
// exist yet. A caller that merges before its tag holds the file fails every
// event in the repo with a missing-workflow error.
//
// It reads each `uses:` line in .github/workflows and asks GitHub, through
// the gh CLI, whether the path exists at the ref. Local, branch-ref, and
// non-may-journal lines are ignored. When GitHub cannot answer, such as
// offline, without a token, or for a repo the token cannot read, it skips
// with a note and never passes or fails a ref it could not see.
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "workflow-ref-exists", TimeoutMs: 30000},
		Run:      run,
	})
}

// org is the owner whose workflows and actions are checked.
const org = "may-journal"

// workflowDir holds the workflow files the check reads.
const workflowDir = ".github/workflows/"

var (
	// usesRe matches a `uses:` key and its value, quoted or not.
	usesRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*['"]?([^'"\s]+)`)
	// shaRe matches a full commit sha.
	shaRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// call is one `uses:` line naming a may-journal repo.
type call struct {
	file string
	line int
	// value is the whole `uses:` value, as written.
	value string
	repo  string
	// dir is the path inside repo: a workflow file or an action folder.
	dir string
	ref string
}

// at prefixes msg with where the call is written and what it names.
func (c call) at(msg string) string {
	return fmt.Sprintf("%s: line %d: %s: %s", c.file, c.line, c.value, msg)
}

// verdict is what GitHub says about one call: a finding when its path is
// missing, a note when it was skipped, or neither when it holds.
type verdict struct {
	finding, note string
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := workflowFiles(root)
	var calls []call
	var unread []string
	for _, f := range files {
		found, err := readCalls(root, f)
		if err != nil {
			unread = append(unread, walkfs.Unreadable(f, err))
		}
		calls = append(calls, found...)
	}
	read := len(files) - len(unread)
	result := checkkit.Pass(read)
	switch {
	case len(calls) == 0:
	case !onPath("gh"):
		result = checkkit.Info(read, skipNote("the gh CLI is not installed"))
	default:
		result = judge(read, calls, &github{})
	}
	result.Info = append(unread, result.Info...)
	return result, nil
}

// onPath reports whether a program is on PATH.
func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// workflowFiles is the in-scope workflow files directly under
// .github/workflows.
func workflowFiles(root string) []string {
	var out []string
	for _, f := range walkfs.InScope(walkfs.FilesByExt(root, ".yml", ".yaml")) {
		if rest, ok := strings.CutPrefix(f, workflowDir); ok && !strings.Contains(rest, "/") {
			out = append(out, f)
		}
	}
	return out
}

// judge checks every call against GitHub, failing each missing path, or
// skips the whole run with a note when GitHub cannot answer.
func judge(files int, calls []call, gh *github) checkkit.Result {
	var errs, notes []string
	for _, c := range calls {
		v, err := gh.verify(c)
		if err != nil {
			return checkkit.Info(files, skipNote(err.Error()))
		}
		errs = appendAt(errs, c, v.finding)
		notes = appendAt(notes, c, v.note)
	}
	result := checkkit.Fail(files, errs...)
	result.Ok = len(errs) == 0
	result.Info = notes
	return result
}

// appendAt appends msg, placed at c, to list unless msg is empty.
func appendAt(list []string, c call, msg string) []string {
	if msg == "" {
		return list
	}
	return append(list, c.at(msg))
}

// skipNote is the info note for a run GitHub could not answer.
func skipNote(why string) string {
	return fmt.Sprintf("workflow refs were not verified: %s", why)
}

// readCalls returns the may-journal `uses:` lines of one workflow file,
// pinned to a literal ref.
func readCalls(root, file string) ([]call, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, err
	}
	var out []call
	for i, line := range strings.Split(string(content), "\n") {
		if c, ok := parseLine(line); ok {
			c.file, c.line = file, i+1
			out = append(out, c)
		}
	}
	return out, nil
}

// parseLine reads one workflow line as a may-journal call, if it is one.
func parseLine(line string) (call, bool) {
	m := usesRe.FindStringSubmatch(line)
	if m == nil {
		return call{}, false
	}
	return parseUses(m[1])
}

// parseUses splits `owner/repo[/path]@ref` into a call, keeping only
// may-journal repos at a literal ref.
func parseUses(value string) (call, bool) {
	target, ref, ok := literalRef(value)
	parts := strings.SplitN(target, "/", 3)
	if !ok || len(parts) < 2 || !strings.EqualFold(parts[0], org) {
		return call{}, false
	}
	c := call{value: value, repo: org + "/" + parts[1], ref: ref}
	if len(parts) == 3 {
		c.dir = strings.Trim(parts[2], "/")
	}
	return c, true
}

// literalRef splits value at its `@`; ok is false without a ref or when an
// expression builds the ref.
func literalRef(value string) (target, ref string, ok bool) {
	target, ref, ok = strings.Cut(value, "@")
	return target, ref, ok && ref != "" && !strings.Contains(ref, "${{")
}

// isWorkflow reports whether dir names a reusable workflow file rather than
// an action folder.
func isWorkflow(dir string) bool {
	ext := path.Ext(dir)
	return ext == ".yml" || ext == ".yaml"
}

// github answers ref and path questions through `gh api`, remembering each
// answer so a ref shared by many lines costs one read.
type github struct {
	answers map[string]answer
}

// answer is one cached `gh api` result: found is false on HTTP 404.
type answer struct {
	body  []byte
	found bool
	err   error
}

// get reads one API path. A 404 is an answer, not an error; any other
// failure means GitHub could not be asked.
func (g *github) get(api string) answer {
	if a, ok := g.answers[api]; ok {
		return a
	}
	if g.answers == nil {
		g.answers = map[string]answer{}
	}
	o := checkkit.Exec(exec.Command("gh", "api", api))
	a := answer{body: []byte(o.Stdout), found: o.Code == 0}
	if o.Code != 0 && !strings.Contains(o.Stderr, "HTTP 404") {
		a.err = fmt.Errorf("gh api %s: %s", api, firstLine(o.Stderr, o.Err))
	}
	g.answers[api] = a
	return a
}

// firstLine is the first line of a tool's error output, or the run error.
func firstLine(stderr string, err error) string {
	if line, _, _ := strings.Cut(strings.TrimSpace(stderr), "\n"); line != "" {
		return line
	}
	return fmt.Sprint(err)
}

// verify judges one call. A ref that is not a full sha must be a tag; the
// path must then exist at it. An error means GitHub cannot answer.
func (g *github) verify(c call) (verdict, error) {
	if !shaRe.MatchString(c.ref) {
		tag := g.get(fmt.Sprintf("repos/%s/git/ref/tags/%s", c.repo, c.ref))
		if tag.err != nil || !tag.found {
			return g.notTag(c, tag.err)
		}
	}
	return g.atRef(c)
}

// notTag explains a ref that is not a tag: a branch is out of scope, and a
// ref that is neither fails, unless the repo itself cannot be read.
func (g *github) notTag(c call, err error) (verdict, error) {
	if err != nil {
		return verdict{}, err
	}
	branch := g.get(fmt.Sprintf("repos/%s/git/ref/heads/%s", c.repo, c.ref))
	if branch.err != nil || branch.found {
		return verdict{}, branch.err
	}
	return g.missing(c, fmt.Sprintf("no tag or branch %s exists", c.ref))
}

// atRef judges whether the call's path exists at its ref.
func (g *github) atRef(c call) (verdict, error) {
	found, err := g.pathExists(c)
	if err != nil || found {
		return verdict{}, err
	}
	return g.missing(c, fmt.Sprintf("%s does not exist at %s", pathName(c), c.ref))
}

// missing is the finding for something GitHub answered 404, or a note when
// the repo itself answers 404: GitHub hides a repo the token cannot read,
// so its missing paths prove nothing.
func (g *github) missing(c call, finding string) (verdict, error) {
	repo := g.get("repos/" + c.repo)
	if repo.err != nil {
		return verdict{}, repo.err
	}
	if !repo.found {
		return verdict{note: fmt.Sprintf("skipped, %s cannot be read with this token", c.repo)}, nil
	}
	return verdict{finding: finding}, nil
}

// pathExists reports whether the workflow file, or an action folder's
// action.yml or action.yaml, exists at the ref.
func (g *github) pathExists(c call) (bool, error) {
	a := g.get(path.Join("repos", c.repo, "contents", c.dir) + "?ref=" + url.QueryEscape(c.ref))
	if a.err != nil || !a.found || isWorkflow(c.dir) {
		return a.found, a.err
	}
	return hasActionFile(a.body), nil
}

// hasActionFile reports whether a folder listing holds action metadata.
func hasActionFile(listing []byte) bool {
	var entries []struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(listing, &entries)
	for _, e := range entries {
		if e.Name == "action.yml" || e.Name == "action.yaml" {
			return true
		}
	}
	return false
}

// pathName is the path a finding names: the workflow file, or the action
// metadata file the folder lacks.
func pathName(c call) string {
	if isWorkflow(c.dir) {
		return c.dir
	}
	return path.Join(c.dir, "action.yml")
}
