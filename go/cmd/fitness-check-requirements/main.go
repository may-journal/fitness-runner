package check

import (
	"errors"
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"slices"
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

// fixFlag makes the check delete every leftover Source line before it
// judges, so `fitness requirements --fix` converts a repo's docs.
const fixFlag = "--fix"

func run(root string, args []string) (checkkit.Result, error) {
	paths := docPaths(root)
	if err := dropSources(root, paths, args); err != nil {
		return checkkit.Result{}, err
	}
	tests, files, written, err := collectTests(root, langs.All)
	if err != nil || len(written) == 0 {
		return checkkit.Pass(0), err
	}
	checks, proven := reqChecks(root)
	errs, notes, err := docErrors(root, paths, tests, written, proven)
	if err != nil {
		return checkkit.Result{}, err
	}
	res := result(len(paths)+len(files), append(errs, pairErrors(root, paths, append(files, checks...), tests)...))
	res.Info = notes
	return res, nil
}

// docErrors judges each doc's shape, IDs, area, test owners and tools, and
// notes each doc it could not read.
//
// A doc in proven has a fitness/req-NNNN/ check owning acceptances no test
// does, and a check with no doc fails.
func docErrors(root string, paths []string, tests []reqTest, written []lang.Module, proven map[string]bool) (errs, notes []string, err error) {
	docs, errs, skipped := requirement.Load(root, paths)
	notes = skipped
	if len(skipped) > 0 {
		// A doc left out would leave its tests unowned and its IDs unknown.
		return errs, append(notes, "ownership and history were not judged without every doc"), nil
	}
	errs = append(errs, missingDocs(paths)...)
	hist, historyNote := readHistory(root)
	if historyNote != "" {
		notes = append(notes, historyNote)
	}
	errs = append(errs, idErrors(docs, hist)...)
	errs = append(errs, overlapErrors(docs)...)
	errs = append(errs, ownershipErrors(docs, tests, written, proven)...)
	errs = append(errs, orphanChecks(docs, proven)...)
	tooled, err := toolErrors(root, tests, written)
	return append(errs, tooled...), notes, err
}

// dropSources deletes the leftover Source line from each doc at paths when
// args ask for the fix.
func dropSources(root string, paths, args []string) error {
	if !slices.Contains(args, fixFlag) {
		return nil
	}
	docs, _, _ := requirement.Load(root, paths)
	var err error
	for _, d := range docs {
		err = errors.Join(err, requirement.DropSource(root, d))
	}
	return err
}

// result passes with no errors and fails with any.
func result(files int, errs []string) checkkit.Result {
	if len(errs) > 0 {
		return checkkit.Fail(files, errs...)
	}
	return checkkit.Pass(files)
}

// collectTests reads every module's tests and their files in registry order,
// and the modules the repo is written in.
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
