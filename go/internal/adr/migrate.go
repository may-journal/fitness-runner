package adr

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/repocheck"
)

// Migration moves a repo's ADR files into `ADR` issues. It opens one issue
// per file, points each link at its issue, and removes or trims the file.
type Migration struct {
	// Root is the repo's folder.
	Root string
	// Branch is the branch issue bodies link the repo's other files on.
	Branch string
	// GitHub opens and reads the issues.
	GitHub GitHub
	// DryRun prints what a migration would do and changes nothing.
	DryRun bool
	// Out receives one line per step.
	Out io.Writer
}

// checkReadme matches a runner ADR, the README of a `fitness/adr-NNNN` check.
var checkReadme = regexp.MustCompile(`^` + repocheck.Folder + `/` + repocheck.ADRPrefix + `\d+/` + repocheck.Readme + `$`)

// source is one ADR file on its way to an issue.
type source struct {
	rel, content string
	record       Record
	// opened is the body its issue holds, or "" before it has one.
	opened string
	// kept marks an issue an earlier migration opened.
	kept bool
	// closed marks an issue that is already closed.
	closed bool
}

// Run migrates the ADR files among files, the repo's tracked paths.
func (m Migration) Run(files []string) error {
	sources := m.find(files)
	if len(sources) == 0 {
		fmt.Fprintln(m.Out, "no ADR files to move")
		return nil
	}
	issues, err := m.known()
	if err != nil {
		return err
	}
	urls := map[string]string{}
	if err := m.open(sources, issues, urls); err != nil {
		return err
	}
	if err := m.finish(sources, urls); err != nil {
		return err
	}
	return m.rewrite(files, sources, urls)
}

// find reads each ADR file among files.
func (m Migration) find(files []string) []*source {
	var out []*source
	for _, rel := range files {
		if !MayBe(rel) {
			continue
		}
		data, err := os.ReadFile(m.path(rel))
		if _, ok := Rule(rel, string(data)); err == nil && ok {
			out = append(out, &source{rel: rel, content: string(data), record: Parse(string(data))})
		}
	}
	return out
}

// path is rel's place on disk.
func (m Migration) path(rel string) string {
	return filepath.Join(m.Root, filepath.FromSlash(rel))
}

// known maps each file an `ADR` issue came from to that issue. Outside a
// dry run it also makes sure the label exists.
func (m Migration) known() (map[string]Issue, error) {
	issues, err := m.GitHub.Migrated()
	if err != nil || m.DryRun {
		return issues, err
	}
	return issues, m.GitHub.EnsureLabel()
}

// open opens an issue for each source no issue names yet, recording its URL.
// A dry run records a stand-in instead.
func (m Migration) open(sources []*source, issues map[string]Issue, urls map[string]string) error {
	for _, s := range sources {
		if issue, ok := issues[s.rel]; ok {
			urls[s.rel], s.opened, s.kept, s.closed = issue.URL, issue.Body, true, issue.State == "CLOSED"
			fmt.Fprintf(m.Out, "kept %s: %s\n", s.rel, issue.URL)
			continue
		}
		if err := m.openOne(s, urls); err != nil {
			return err
		}
	}
	return nil
}

// openOne opens the issue for one source.
func (m Migration) openOne(s *source, urls map[string]string) error {
	if m.DryRun {
		urls[s.rel] = "the new issue for " + s.rel
		fmt.Fprintf(m.Out, "would open %q from %s\n", s.record.IssueTitle(), s.rel)
		return nil
	}
	body := m.body(s, urls)
	url, err := m.GitHub.Open(s.record.IssueTitle(), body)
	if err != nil {
		return err
	}
	urls[s.rel], s.opened = url, body
	fmt.Fprintf(m.Out, "opened %s from %s\n", url, s.rel)
	return nil
}

// finish edits each issue whose links to ADR files now resolve to issues,
// and closes each superseded ADR's issue.
func (m Migration) finish(sources []*source, urls map[string]string) error {
	for _, s := range sources {
		if body := m.finalBody(s, urls); body != "" {
			if err := m.GitHub.Edit(urls[s.rel], body); err != nil {
				return err
			}
		}
		if err := m.retire(s, urls); err != nil {
			return err
		}
	}
	return nil
}

// retire closes a superseded ADR's open issue as not planned, since the
// issue's state is its status. The comment names the issue that replaces it.
func (m Migration) retire(s *source, urls map[string]string) error {
	if !s.record.Superseded() || s.closed {
		return nil
	}
	r := s.record
	r.StatusNote = RewriteLinks(r.StatusNote, s.rel, func(target, _ string) (string, bool) { return issueFor(target, urls) })
	if m.DryRun {
		fmt.Fprintf(m.Out, "would close the issue for %s as superseded\n", s.rel)
		return nil
	}
	if err := m.GitHub.Close(urls[s.rel], r.CloseComment()); err != nil {
		return err
	}
	fmt.Fprintf(m.Out, "closed %s as superseded\n", urls[s.rel])
	return nil
}

// finalBody is the body s's issue needs now, or "" when it needs no edit.
// A kept issue only has its links to ADR files on the branch pointed at
// their issues, so edits made since stay.
func (m Migration) finalBody(s *source, urls map[string]string) string {
	want := m.body(s, urls)
	if s.kept {
		want = m.blobsToIssues(s.opened, urls)
	}
	if s.opened == "" || want == s.opened {
		return ""
	}
	return want
}

// blobsToIssues points each link in body to an ADR file on the branch at
// that file's issue.
func (m Migration) blobsToIssues(body string, urls map[string]string) string {
	prefix := "https://github.com/" + m.GitHub.Repo + "/blob/" + m.Branch + "/"
	blob := regexp.MustCompile(regexp.QuoteMeta(prefix) + `([^\s)#]+)(#[^\s)]*)?`)
	return blob.ReplaceAllStringFunc(body, func(link string) string {
		if url, ok := issueFor(blob.FindStringSubmatch(link)[1], urls); ok {
			return url
		}
		return link
	})
}

// body is the issue body for s. Its links point at issues, or at files on
// the branch, since a relative link means nothing in an issue.
func (m Migration) body(s *source, urls map[string]string) string {
	resolve := func(target, anchor string) (string, bool) {
		if url, ok := issueFor(target, urls); ok {
			return url, true
		}
		link := "https://github.com/" + m.GitHub.Repo + "/blob/" + m.Branch + "/" + target
		if anchor != "" {
			link += "#" + anchor
		}
		return link, true
	}
	r := s.record
	r.StatusNote = RewriteLinks(r.StatusNote, s.rel, resolve)
	return RewriteLinks(r.IssueBody(s.rel), s.rel, resolve)
}

// issueFor is the issue URL for a link to target: an ADR file, or the
// folder of a check whose README is one.
func issueFor(target string, urls map[string]string) (string, bool) {
	if url, ok := urls[target]; ok {
		return url, true
	}
	url, ok := urls[path.Join(target, repocheck.Readme)]
	return url, ok
}

// rewrite removes or trims each ADR file and points every Markdown link to
// one at its issue.
func (m Migration) rewrite(files []string, sources []*source, urls map[string]string) error {
	bySource := map[string]*source{}
	for _, s := range sources {
		bySource[s.rel] = s
	}
	for _, rel := range files {
		if err := m.rewriteOne(rel, bySource[rel], urls); err != nil {
			return err
		}
	}
	return nil
}

// rewriteOne rewrites one tracked file, s being its ADR source or nil.
func (m Migration) rewriteOne(rel string, s *source, urls map[string]string) error {
	if s != nil && !checkReadme.MatchString(rel) {
		return m.remove(rel)
	}
	if !mdx.IsMarkdown(strings.ToLower(rel)) {
		return nil
	}
	return m.relink(rel, s, urls)
}

// relink points rel's Markdown links at issues. A check README with an ADR
// source s is first trimmed to a pointer.
func (m Migration) relink(rel string, s *source, urls map[string]string) error {
	data, err := os.ReadFile(m.path(rel))
	if err != nil {
		return err
	}
	text := string(data)
	if frozen(rel, text) {
		return nil
	}
	if s != nil {
		text = s.record.Pointer(Title(s.content), urls[rel])
	}
	text = RewriteLinks(text, rel, func(target, _ string) (string, bool) { return issueFor(target, urls) })
	if text == string(data) {
		return nil
	}
	return m.write(rel, text)
}

// generated matches the header of a generated file.
var generated = regexp.MustCompile(`(?m)^.*Code generated .* DO NOT EDIT`)

// frozen reports whether a file keeps its links: release history, which is
// never edited, and generated files, which their sources rewrite.
func frozen(rel, text string) bool {
	return path.Base(rel) == "CHANGELOG.md" || strings.HasPrefix(rel, "changelog/") || generated.MatchString(text)
}

// remove deletes an ADR file whose decision is now an issue.
func (m Migration) remove(rel string) error {
	if m.DryRun {
		fmt.Fprintf(m.Out, "would remove %s\n", rel)
		return nil
	}
	fmt.Fprintf(m.Out, "removed %s\n", rel)
	return os.Remove(m.path(rel))
}

// write saves a file whose links or text changed.
func (m Migration) write(rel, text string) error {
	if m.DryRun {
		fmt.Fprintf(m.Out, "would rewrite %s\n", rel)
		return nil
	}
	fmt.Fprintf(m.Out, "rewrote %s\n", rel)
	return os.WriteFile(m.path(rel), []byte(text), 0o644)
}
