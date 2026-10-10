// Command fitness-check-doc-template enforces a convention: template files
// define a markdown document's shape, and their neighbors must match one. A
// template is a file named template.md or *.template.md, any markdown file in
// a templates folder, or a GitHub PR or Issue template under .github. Every
// template's level-2 section must carry a guiding HTML comment. A folder's
// templates are its own template.md-style files plus those in its templates
// folder. Walking up from each markdown file, the first folder with templates
// governs it, and the file must have the same sections as any one of them,
// via the shared mdtemplate engine. GitHub templates govern PR and Issue
// bodies (checked by pr-structure and plan-structure), so they get the comment
// rule but do not pull sibling files into a match.
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

// conformanceTemplates indexes every conformance template by its directory,
// keeping all of a folder's templates in path order.
func conformanceTemplates(docs []doc) map[string][]doc {
	m := map[string][]doc{}
	for _, d := range docs {
		if isConformanceTemplate(d.path) {
			dir := governedDir(d.path)
			m[dir] = append(m[dir], d)
		}
	}
	return m
}

// conformanceErrors validates every governed file against the templates of
// its nearest templated folder.
func conformanceErrors(docs []doc, templates map[string][]doc) []string {
	var errs []string
	for _, d := range docs {
		errs = append(errs, docConformance(d, templates)...)
	}
	return errs
}

// docConformance returns one file's section-match errors, or nil when it is a
// template, has no governing template above it, or matches any one of its
// folder's templates. Otherwise the errors are the closest template's: the one
// with the fewest differences.
func docConformance(d doc, templates map[string][]doc) []string {
	if isTemplate(d.path) {
		return nil
	}
	candidates := nearestTemplates(d.path, templates)
	closest, diffs := closestTemplate(d, candidates)
	prefix := diffPrefix(d.path, closest.path, len(candidates))
	errs := make([]string, len(diffs))
	for i, msg := range diffs {
		errs[i] = prefix + msg
	}
	return errs
}

// diffPrefix leads each of a file's errors with its path, and names the
// closest template when the folder holds several.
func diffPrefix(path, closest string, templates int) string {
	if templates > 1 {
		return fmt.Sprintf("%s: matches none of %d templates; closest is %s: ", path, templates, closest)
	}
	return path + ": "
}

// closestTemplate returns the candidate with the fewest differences from d,
// and those differences; the first template in path order wins a tie.
func closestTemplate(d doc, candidates []doc) (doc, []string) {
	var best doc
	var bestDiffs []string
	for i, t := range candidates {
		diffs := mdtemplate.Validate(d.body, mdtemplate.SpecFromTemplate(t.body, "document"))
		if i == 0 || len(diffs) < len(bestDiffs) {
			best, bestDiffs = t, diffs
		}
		if len(diffs) == 0 {
			break
		}
	}
	return best, bestDiffs
}

// nearestTemplates walks up from the file's directory to the repo root,
// returning the templates of the first folder that holds any.
func nearestTemplates(path string, templates map[string][]doc) []doc {
	for dir := dirOf(path); ; {
		if ts, ok := templates[dir]; ok {
			return ts
		}
		if dir == "." || dir == "" {
			return nil
		}
		dir = dirOf(dir)
	}
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

// governedDir returns the folder a conformance template governs: its own
// folder, or the parent of its templates folder.
func governedDir(path string) string {
	if inTemplatesFolder(path) {
		return dirOf(dirOf(path))
	}
	return dirOf(path)
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
