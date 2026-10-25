package check

import (
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "requirements", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// docsDir holds the requirement docs, one per file.
const docsDir = "docs/requirements/"

func run(root string, _ []string) (checkkit.Result, error) {
	paths := docPaths(root)
	tests, files, written, err := collectTests(root, langs.All)
	if err != nil || len(written) == 0 {
		return checkkit.Pass(0), err
	}
	errs, err := docErrors(root, paths, tests, written)
	if err != nil {
		return checkkit.Result{}, err
	}
	return result(len(paths)+len(files), append(errs, pairErrors(root, paths, files, tests)...)), nil
}

// docErrors judges every doc's shape, its IDs against history, docs that
// share an area, which tests own its acceptances, and the tools
// the tests use.
func docErrors(root string, paths []string, tests []reqTest, written []lang.Module) ([]string, error) {
	docs, errs, err := requirement.Load(root, paths)
	if err != nil {
		return nil, err
	}
	errs = append(errs, missingDocs(paths)...)
	errs = append(errs, idErrors(docs, readHistory(root))...)
	errs = append(errs, overlapErrors(docs)...)
	errs = append(errs, ownershipErrors(docs, tests, written)...)
	tooled, err := toolErrors(root, tests, written)
	return append(errs, tooled...), err
}

// result passes with no errors and fails with any.
func result(files int, errs []string) checkkit.Result {
	if len(errs) > 0 {
		return checkkit.Fail(files, errs...)
	}
	return checkkit.Pass(files)
}

// collectTests reads every module's tests and the files that hold them, in
// registry order, and lists the modules whose language the repo is written
// in.
func collectTests(root string, modules []lang.Module) ([]reqTest, []string, []lang.Module, error) {
	var tests []reqTest
	var files []string
	var written []lang.Module
	for _, m := range lang.Testing(modules) {
		found, held, writes, err := m.Tests.Scan(root)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, t := range found {
			tests = append(tests, reqTest{Test: t, module: m})
		}
		files = append(files, held...)
		if writes {
			written = append(written, m)
		}
	}
	return tests, files, written, nil
}

// docPaths lists every requirement doc. A template.md beside them is the
// doc-template check's concern, not a requirement.
func docPaths(root string) []string {
	var out []string
	for _, f := range walkfs.FilesByExt(root, ".md") {
		if strings.HasPrefix(f, docsDir) && !strings.HasSuffix(f, "/template.md") {
			out = append(out, f)
		}
	}
	return out
}

// missingDocs fails a repo with code that has no requirement docs.
func missingDocs(paths []string) []string {
	if len(paths) > 0 {
		return nil
	}
	return []string{docsDir + " has no requirement docs; add NNNN-kebab-title.md files from the template in the fitness-check-requirements README"}
}

// idErrors flags a requirement ID that two docs share, and any doc or
// acceptance ID that history shows was removed before.
func idErrors(docs []requirement.Document, hist history) []string {
	var errs []string
	owners := map[string][]string{}
	for _, d := range docs {
		owners[d.ID] = append(owners[d.ID], d.Path)
		errs = append(errs, reusedErrors(d, hist)...)
	}
	for _, id := range sortedKeys(owners) {
		if len(owners[id]) > 1 {
			errs = append(errs, "requirement "+id+" is used by "+strings.Join(owners[id], ", ")+"; give each requirement its own ID")
		}
	}
	return errs
}

// reusedErrors flags the doc's IDs that a past commit deleted.
func reusedErrors(d requirement.Document, hist history) []string {
	var errs []string
	for _, id := range d.IDs() {
		if hist.removed[id.ID] {
			errs = append(errs, requirement.At(d.Path, id.Line, id.ID+" was deleted before; IDs are never reused, so take the next free number"))
		}
	}
	return errs
}

// sortedKeys returns m's keys in order, for stable output.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
