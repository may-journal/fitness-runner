// Command fitness-check-doc-template enforces a convention: template files
// define a markdown document's shape. A template is template.md,
// *.template.md, any markdown file in a templates folder (governing its
// parent), or a GitHub PR or Issue template. Every template's level-2 section
// needs a guiding HTML comment. Walking up from each markdown file, the first
// folder with a template that governs it decides, and the file must have the
// same sections as any one of them. A template in a templates folder that
// shares the file's name governs it alone, with its whole shape, and no other
// file. An architecture view with no such copy follows the template fitness
// ships for it. GitHub templates get only the comment rule.
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
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
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
	enforce, err := conf.ViewRulesFail(root)
	if err != nil {
		return checkkit.Fail(len(docs), err.Error()), nil
	}
	return result(len(docs), validateAll(docs, enforce)), nil
}

// verdict is what the rules found: errors fail the run, and warnings, from
// staged rules a repo has not opted in to, only inform.
type verdict struct {
	errs, warnings []string
}

// add files staged findings as errors when enforce is set, else as
// warnings.
func (v *verdict) add(errs, staged []string, enforce bool) {
	v.errs = append(v.errs, errs...)
	if enforce {
		v.errs = append(v.errs, staged...)
	} else {
		v.warnings = append(v.warnings, staged...)
	}
}

// result fails on any error and carries each warning as an info note,
// closed by one note on how to make them fail.
func result(files int, v verdict) checkkit.Result {
	r := checkkit.Pass(files)
	if len(v.errs) > 0 {
		r = checkkit.Fail(files, v.errs...)
	}
	r.Warn(v.warnings, conf.StagedNote(len(v.warnings)))
	return r
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
// rule over every governed file. enforce fails the staged rules too.
func validateAll(docs []doc, enforce bool) verdict {
	byDir, named := templatesByDir(docs), namedTemplates(docs)
	v := verdict{errs: commentErrors(docs)}
	for _, d := range docs {
		if !mdtemplate.IsTemplate(d.path) {
			errs, staged := judge(d, byDir, named, enforce)
			v.add(errs, staged, enforce)
		}
	}
	return v
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
// template governs passes, as in a repo with no templates. The shipped
// templates are a staged rule: until enforce, a view without a copy is
// still judged by the walk, and every shipped finding is staged.
func judge(d doc, byDir map[string][]doc, named map[string]bool, enforce bool) (errs, staged []string) {
	t, ok := shipped(d, named)
	if !ok {
		return walk(d, byDir, named)
	}
	settled, more := shapeErrors(d, t)
	if enforce {
		return settled, more
	}
	errs, _ = walk(d, byDir, named)
	return errs, append(settled, more...)
}

// walk judges d at the first folder, from its own up, with a template that
// governs it.
func walk(d doc, byDir map[string][]doc, named map[string]bool) (errs, staged []string) {
	for dir := dirOf(d.path); ; dir = dirOf(dir) {
		if errs, staged, governed := judgeAt(d, byDir[dir], named); governed || dir == "." {
			return errs, staged
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
// one of the folder's other templates. A named template governs only its
// namesake, so false means none of templates governs the doc, as when all
// are named for other docs.
func judgeAt(d doc, templates []doc, named map[string]bool) (errs, staged []string, governed bool) {
	var rest []doc
	for _, t := range templates {
		if !named[t.path] {
			rest = append(rest, t)
		} else if namesake(t.path) == d.path {
			errs, staged = shapeErrors(d, t)
			return errs, staged, true
		}
	}
	if len(rest) == 0 {
		return nil, nil, false
	}
	return docErrors(d, rest), nil, true
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

// shapeErrors judges a doc against the template that shares its file name.
// errs covers its opening sections, legends, callout table columns, and
// captions; staged covers its diagram types and palette.
func shapeErrors(d doc, t doc) (errs, staged []string) {
	settled, more := mdtemplate.ShapeFromTemplate(t.path, t.body).Validate(d.body)
	return located(d.path, settled), located(d.path, more)
}

// located prefixes each finding with the doc's path and its line, if any.
func located(file string, findings []mdtemplate.Finding) []string {
	var out []string
	for _, f := range findings {
		at := file
		if f.Line > 0 {
			at += ":" + strconv.Itoa(f.Line)
		}
		out = append(out, at+": "+f.Text)
	}
	return out
}

// dirOf returns the slash directory of a slash path; the repo root is ".".
func dirOf(path string) string {
	return filepath.ToSlash(filepath.Dir(path))
}
