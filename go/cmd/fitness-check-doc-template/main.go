// Command fitness-check-doc-template enforces a convention: a template file
// defines a markdown document's shape, and its neighbors must match it. A
// template is a file named template.md or *.template.md, or a GitHub PR or
// Issue template under .github. Every template's level-2 section must carry a
// guiding HTML comment. For a template.md-style file, every markdown file in
// its folder or a descendant folder — the nearest such template winning — must
// have the same sections, via the shared mdtemplate engine. GitHub templates
// govern PR and Issue bodies (checked by pr-structure and plan-structure), so
// they get the comment rule but do not pull sibling files into a match.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "doc-template"},
		Run:      run,
	})
}

// doc is one markdown file: its slash-relative path and content.
type doc struct {
	path string
	body string
}

func run(root string, _ []string) (checkkit.Result, error) {
	docs, err := readAll(root, walkfs.FilesByExt(root, ".md"))
	if err != nil {
		return checkkit.Result{}, err
	}
	errs := validateAll(docs)
	if len(errs) > 0 {
		return checkkit.Fail(len(docs), errs...), nil
	}
	return checkkit.Pass(len(docs)), nil
}

// readAll loads every markdown file into a doc.
func readAll(root string, files []string) ([]doc, error) {
	docs := make([]doc, 0, len(files))
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc{path: f, body: string(b)})
	}
	return docs, nil
}

// validateAll runs the comment rule over every template and the conformance
// rule over every governed file.
func validateAll(docs []doc) []string {
	templates := conformanceTemplates(docs)
	var errs []string
	errs = append(errs, commentErrors(docs)...)
	errs = append(errs, conformanceErrors(docs, templates)...)
	return errs
}

// commentErrors flags every template section without a guiding HTML comment.
func commentErrors(docs []doc) []string {
	var errs []string
	for _, d := range docs {
		if !isTemplate(d.path) {
			continue
		}
		for _, h := range mdtemplate.MissingCommentSections(d.body) {
			errs = append(errs, fmt.Sprintf("%s: section `## %s` needs a guiding <!-- comment -->", d.path, h))
		}
	}
	return errs
}

// conformanceTemplates indexes each conformance template by its directory.
func conformanceTemplates(docs []doc) map[string]doc {
	m := map[string]doc{}
	for _, d := range docs {
		if isConformanceTemplate(d.path) {
			m[dirOf(d.path)] = d
		}
	}
	return m
}

// conformanceErrors validates every governed file against its nearest
// conformance template.
func conformanceErrors(docs []doc, templates map[string]doc) []string {
	var errs []string
	for _, d := range docs {
		errs = append(errs, docConformance(d, templates)...)
	}
	return errs
}

// docConformance returns one file's section-match errors, or nil when it is a
// template or has no governing template above it.
func docConformance(d doc, templates map[string]doc) []string {
	if isTemplate(d.path) {
		return nil
	}
	t, ok := nearestTemplate(d.path, templates)
	if !ok {
		return nil
	}
	var errs []string
	for _, msg := range mdtemplate.Validate(d.body, mdtemplate.SpecFromTemplate(t.body, "document")) {
		errs = append(errs, d.path+": "+msg)
	}
	return errs
}

// nearestTemplate walks up from the file's directory to the repo root,
// returning the first conformance template found.
func nearestTemplate(path string, templates map[string]doc) (doc, bool) {
	for dir := dirOf(path); ; {
		if t, ok := templates[dir]; ok {
			return t, true
		}
		if dir == "." || dir == "" {
			return doc{}, false
		}
		dir = dirOf(dir)
	}
}

// isConformanceTemplate reports whether the file names a template that governs
// its neighbors: template.md or *.template.md.
func isConformanceTemplate(path string) bool {
	base := filepath.Base(path)
	return base == "template.md" || strings.HasSuffix(base, ".template.md")
}

// isGithubTemplate reports whether the file is a GitHub PR or Issue template.
func isGithubTemplate(path string) bool {
	if filepath.Base(path) == "PULL_REQUEST_TEMPLATE.md" {
		return true
	}
	return strings.Contains(path, ".github/ISSUE_TEMPLATE/") && strings.HasSuffix(path, ".md")
}

// isTemplate reports whether the file is any kind of template.
func isTemplate(path string) bool {
	return isConformanceTemplate(path) || isGithubTemplate(path)
}

// dirOf returns the slash directory of a slash path; the repo root is ".".
func dirOf(path string) string {
	return filepath.ToSlash(filepath.Dir(path))
}
