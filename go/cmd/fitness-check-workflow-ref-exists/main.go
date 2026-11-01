package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "workflow-ref-exists", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsStripped, TimeoutMs: 30000},
		Run:      run,
	})
}

// org is the owner whose workflows and actions are checked.
const org = "may-journal"

// workflowDir holds the workflow files the check reads.
const workflowDir = ".github/workflows/"

// askBudget bounds every GitHub read in a run, inside the check's timeout, so a stalled network ends in a note.
const askBudget = 20 * time.Second

var (
	// usesRe matches a `uses:` key and its value, quoted or not.
	usesRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*['"]?([^'"\s]+)`)
	// blockRe matches a line whose value is a `|` or `>` block scalar, such as a `run:` script.
	blockRe = regexp.MustCompile(`:\s*[|>][-+0-9]*\s*(?:#.*)?$`)
	// shaRe matches a full commit sha.
	shaRe = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)
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

// verdict is what GitHub says about one call: a finding if its path is missing, a note if skipped, else neither.
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
	if len(calls) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), askBudget)
		defer cancel()
		result = judge(read, calls, &github{ctx: ctx})
	}
	result.Info = append(unread, result.Info...)
	return result, nil
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

// judge checks every call against GitHub and fails each missing path, keeping findings so far and noting calls it could not verify.
func judge(files int, calls []call, gh *github) checkkit.Result {
	if note := checkkit.Offline("the workflow ref reads"); note != "" {
		return checkkit.Info(files, note)
	}
	var errs, notes []string
	for i, c := range calls {
		v, err := gh.verify(c)
		if err != nil {
			notes = append(notes, skipNote(calls, i, err))
			break
		}
		errs = appendAt(errs, c, v.finding)
		notes = appendAt(notes, c, v.note)
	}
	result := checkkit.Pass(files)
	if len(errs) > 0 {
		result = checkkit.Fail(files, errs...)
	}
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

// skipNote is the info note for the calls, from calls[i] on, that GitHub
// could not answer.
func skipNote(calls []call, i int, err error) string {
	if i == 0 {
		return fmt.Sprintf("workflow refs were not verified: %v", err)
	}
	return fmt.Sprintf("workflow refs from %s line %d on were not verified: %v", calls[i].file, calls[i].line, err)
}

// readCalls returns the may-journal `uses:` lines of one workflow file,
// pinned to a literal ref, skipping the text of block scalars.
func readCalls(root, file string) ([]call, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, err
	}
	var out []call
	block := -1
	for i, line := range strings.Split(string(content), "\n") {
		if inBlock(line, &block) {
			continue
		}
		if c, ok := parseLine(line); ok {
			c.file, c.line = file, i+1
			out = append(out, c)
		}
	}
	return out, nil
}

// inBlock reports whether line is text inside a block scalar. block holds
// the indent of the key that opened the current block, or -1 outside one.
func inBlock(line string, block *int) bool {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if *block >= 0 && (strings.TrimSpace(line) == "" || indent > *block) {
		return true
	}
	*block = -1
	if blockRe.MatchString(line) {
		*block = indent
	}
	return false
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
	ctx     context.Context
	answers map[string]answer
}

// answer is one cached `gh api` result: found is false on a 404, refused holds other refusals, and err a failure to reach GitHub.
type answer struct {
	body     []byte
	found    bool
	notFound string
	refused  string
	err      error
}

// get reads one API path.
func (g *github) get(api string) answer {
	if a, ok := g.answers[api]; ok {
		return a
	}
	if g.answers == nil {
		g.answers = map[string]answer{}
	}
	a := g.ask(api)
	g.answers[api] = a
	return a
}

// ask runs `gh api` once and sorts its outcome: found, a 404, a refusal of
// this request, or an environment that cannot reach GitHub.
func (g *github) ask(api string) answer {
	o := checkkit.Exec(exec.CommandContext(g.ctx, "gh", "api", api))
	err := aftereffect.GHErr(g.ctx, o)
	stderr := firstLine(o.Stderr, err)
	switch {
	case err == nil:
		return answer{body: []byte(o.Stdout), found: true}
	case checkkit.IsInternal(err) || strings.Contains(strings.ToLower(stderr), "rate limit"):
		return answer{err: fmt.Errorf("gh api %s: %s", api, stderr)}
	case strings.Contains(stderr, "HTTP 404"):
		return answer{notFound: stderr}
	}
	return answer{refused: stderr}
}

// firstLine is the first line of a tool's error output, or the run error.
func firstLine(stderr string, err error) string {
	if line, _, _ := strings.Cut(strings.TrimSpace(stderr), "\n"); line != "" {
		return line
	}
	return fmt.Sprint(err)
}

// verify judges one call: a ref that is no full sha must be a tag, and the path must exist at it.
func (g *github) verify(c call) (verdict, error) {
	v, err := g.judgeRef(c)
	var refused refusal
	if errors.As(err, &refused) {
		return verdict{note: "skipped, GitHub answered " + string(refused)}, nil
	}
	return v, err
}

// refusal is a GitHub answer that leaves one call without a verdict.
type refusal string

func (r refusal) Error() string { return string(r) }

// read is get that turns a refusal into an error.
func (g *github) read(api string) (answer, error) {
	a := g.get(api)
	if a.refused != "" {
		return a, refusal(a.refused)
	}
	return a, a.err
}

// judgeRef checks the ref, then the path at it.
func (g *github) judgeRef(c call) (verdict, error) {
	if !shaRe.MatchString(c.ref) {
		tag, err := g.read(fmt.Sprintf("repos/%s/git/ref/tags/%s", c.repo, c.ref))
		if err != nil || !tag.found {
			return g.notTag(c, err)
		}
	}
	return g.atRef(c)
}

// notTag explains a non-tag ref: a branch is out of scope, and any other ref fails unless the repo is hidden.
func (g *github) notTag(c call, err error) (verdict, error) {
	if err != nil {
		return verdict{}, err
	}
	branch, err := g.read(fmt.Sprintf("repos/%s/git/ref/heads/%s", c.repo, c.ref))
	if err != nil || branch.found {
		return verdict{}, err
	}
	return g.missing(c, fmt.Sprintf("%s is not a tag, a branch, or a full commit sha", c.ref))
}

// atRef judges whether the call's path exists at its ref, telling a sha naming no commit from a missing path.
func (g *github) atRef(c call) (verdict, error) {
	found, err := g.pathExists(c)
	if err != nil || found {
		return verdict{}, err
	}
	if strings.Contains(g.get(contentsAPI(c)).notFound, "No commit found") {
		return g.missing(c, fmt.Sprintf("no commit %s exists", c.ref))
	}
	return g.missing(c, fmt.Sprintf("%s does not exist at %s", pathName(c), c.ref))
}

// contentsAPI is the API path of the call's path at its ref.
func contentsAPI(c call) string {
	return path.Join("repos", c.repo, "contents", c.dir) + "?ref=" + url.QueryEscape(c.ref)
}

// missing is the finding for a 404 answer, or a note when the repo answers 404, since GitHub hides unreadable repos.
func (g *github) missing(c call, finding string) (verdict, error) {
	repo, err := g.read("repos/" + c.repo)
	if err != nil {
		return verdict{}, err
	}
	if !repo.found {
		return verdict{note: fmt.Sprintf("skipped, %s cannot be read with this token", c.repo)}, nil
	}
	return verdict{finding: finding}, nil
}

// pathExists reports whether the workflow file, or an action folder's
// action.yml or action.yaml, exists at the ref.
func (g *github) pathExists(c call) (bool, error) {
	a, err := g.read(contentsAPI(c))
	if err != nil || !a.found || isWorkflow(c.dir) {
		return a.found, err
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
