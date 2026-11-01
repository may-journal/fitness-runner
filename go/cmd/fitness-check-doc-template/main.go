package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/architecture/templates"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "doc-template", Scope: checkkit.ScopeFiles},
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
	byDir, named := templatesByDir(docs), namedTemplates(docs)
	errs := commentErrors(docs)
	for _, d := range docs {
		if !mdtemplate.IsTemplate(d.path) {
			errs = append(errs, judge(d, byDir, named)...)
		}
	}
	return errs
}

// commentErrors flags every template section without a guiding HTML comment.
func commentErrors(docs []doc) []string {
	var errs []string
	for _, d := range docs {
		if !mdtemplate.IsTemplate(d.path) {
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
		if mdtemplate.InTemplatesFolder(d.path) {
			dir = dirOf(dir)
		}
		if mdtemplate.IsConformanceTemplate(d.path) {
			m[dir] = append(m[dir], d)
		}
	}
	return m
}

// judge holds an architecture view with no copy of its template to the
// shipped one. Otherwise it walks up from the doc's folder to the first
// folder with a template that governs it, and judges it there. A doc no
// template governs passes, as in a repo with no templates.
func judge(d doc, byDir map[string][]doc, named map[string]bool) []string {
	if t, ok := shipped(d, named); ok {
		return shapeErrors(d, t)
	}
	for dir := dirOf(d.path); ; dir = dirOf(dir) {
		if errs, governed := judgeAt(d, byDir[dir], named); governed || dir == "." {
			return errs
		}
	}
}

// shipped returns the template fitness ships for an architecture view, unless
// the repo keeps its own copy beside the view.
func shipped(d doc, named map[string]bool) (doc, bool) {
	name, body, ok := templates.For(d.path)
	if !ok || named[path.Join(dirOf(d.path), "templates", path.Base(d.path))] {
		return doc{}, false
	}
	return doc{name, body}, true
}

// judgeAt holds a doc to the named template whose namesake it is, else to any
// one of the folder's other templates.
// A named template governs only its namesake.
// So false means none of templates governs the doc, as when all are named for other docs.
func judgeAt(d doc, templates []doc, named map[string]bool) ([]string, bool) {
	var rest []doc
	for _, t := range templates {
		if !named[t.path] {
			rest = append(rest, t)
		} else if namesake(t.path) == d.path {
			return shapeErrors(d, t), true
		}
	}
	if len(rest) == 0 {
		return nil, false
	}
	return docErrors(d, rest), true
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

// namedTemplates marks each file in a templates folder whose namesake, the
// file of the same name in the folder it governs, exists.
func namedTemplates(docs []doc) map[string]bool {
	paths := map[string]bool{}
	for _, d := range docs {
		paths[d.path] = true
	}
	named := map[string]bool{}
	for _, d := range docs {
		if mdtemplate.InTemplatesFolder(d.path) && paths[namesake(d.path)] {
			named[d.path] = true
		}
	}
	return named
}

// namesake is the path a templates-folder file would govern alone: its own
// name in the folder above.
func namesake(template string) string {
	return path.Join(dirOf(dirOf(template)), filepath.Base(template))
}

// shapeErrors judges a doc against the template that shares its file name:
// its opening sections, legends, callout table columns, and captions.
func shapeErrors(d doc, t doc) []string {
	var errs []string
	for _, f := range mdtemplate.ShapeFromTemplate(t.path, t.body).Validate(d.body) {
		at := d.path
		if f.Line > 0 {
			at += ":" + strconv.Itoa(f.Line)
		}
		errs = append(errs, at+": "+f.Text)
	}
	return errs
}

// dirOf returns the slash directory of a slash path; the repo root is ".".
func dirOf(path string) string {
	return filepath.ToSlash(filepath.Dir(path))
}
