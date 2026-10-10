// Command fitness-check-doc-template enforces a convention: template files
// define a markdown document's shape. A template is template.md,
// *.template.md, any markdown file in a templates folder (governing its
// parent), or a GitHub PR or Issue template. Every template's level-2 section
// needs a guiding HTML comment. Walking up from each markdown file, the first
// folder with templates governs it, and the file must have the same sections
// as any one of them. GitHub templates get only the comment rule.
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
	byDir := templatesByDir(docs)
	errs := commentErrors(docs)
	for _, d := range docs {
		if !isTemplate(d.path) {
			errs = append(errs, docErrors(d, nearestTemplates(d.path, byDir))...)
		}
	}
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

// templatesByDir indexes conformance templates by the folder they govern: a
// templates folder's files govern its parent.
func templatesByDir(docs []doc) map[string][]doc {
	m := map[string][]doc{}
	for _, d := range docs {
		dir := dirOf(d.path)
		if inTemplatesFolder(d.path) {
			dir = dirOf(dir)
		}
		if isConformanceTemplate(d.path) {
			m[dir] = append(m[dir], d)
		}
	}
	return m
}

// nearestTemplates walks up from the file's folder and returns the templates
// of the first folder that has any.
func nearestTemplates(path string, byDir map[string][]doc) []doc {
	for dir := dirOf(path); ; dir = dirOf(dir) {
		if ts, ok := byDir[dir]; ok || dir == "." {
			return ts
		}
	}
}

// docErrors returns nil when the doc matches any one template. Otherwise it
// lists the differences from a lone template, or names all of several.
func docErrors(d doc, templates []doc) []string {
	var diffs, names []string
	for _, t := range templates {
		diffs = mdtemplate.Validate(d.body, mdtemplate.SpecFromTemplate(t.body, "document"))
		if len(diffs) == 0 {
			return nil
		}
		names = append(names, t.path)
	}
	if len(templates) > 1 {
		return []string{d.path + ": sections match none of its templates: " + strings.Join(names, ", ")}
	}
	for i := range diffs {
		diffs[i] = d.path + ": " + diffs[i]
	}
	return diffs
}

// isConformanceTemplate reports whether the file names a template that governs
// its neighbors: template.md, *.template.md, or any markdown file directly in
// a templates folder.
func isConformanceTemplate(path string) bool {
	base := filepath.Base(path)
	return inTemplatesFolder(path) || base == "template.md" || strings.HasSuffix(base, ".template.md")
}

// inTemplatesFolder reports whether the file sits directly in a folder named
// templates.
func inTemplatesFolder(path string) bool {
	return filepath.Base(dirOf(path)) == "templates"
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
