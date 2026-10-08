// Command fitness-check-requirements enforces docs/requirements in Go repos:
// each file is one requirement with a unique four-digit ID, a Why, a
// Measurement ratio with one cited source, and acceptances written as a
// single Given, When, and Then chain. Every acceptance is owned by exactly
// one Go test named for it (Test0001_1 owns 0001.1), and every test must own
// one, so a test that proves no requirement is either a missing requirement
// or a duplicate. Git history supplies removed IDs, so an ID is never reused.
package main

import (
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "requirements"},
		Run:      run,
	})
}

// docsDir holds the requirement docs, one per file.
const docsDir = "docs/requirements/"

func run(root string, _ []string) (checkkit.Result, error) {
	paths := docPaths(root)
	if !available(root, paths) {
		return checkkit.Pass(0), nil
	}
	docs, errs, err := loadDocs(root, paths)
	if err != nil {
		return checkkit.Result{}, err
	}
	files := testFiles(root)
	tests, err := scanTests(root, files)
	if err != nil {
		return checkkit.Result{}, err
	}
	errs = append(errs, missingDocs(paths)...)
	errs = append(errs, idErrors(docs, readHistory(root))...)
	errs = append(errs, ownershipErrors(docs, tests, inScope(files))...)
	if len(errs) > 0 {
		return checkkit.Fail(len(paths)+len(files), errs...), nil
	}
	return checkkit.Pass(len(paths) + len(files)), nil
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

// available reports whether the check applies. A repo without Go has no
// tests this check can read. A Go repo owes requirement docs, so one without
// any is always judged and fails; one with docs is judged when a scoped run
// changed a doc or a test file.
func available(root string, paths []string) bool {
	if len(walkfs.FilesByExt(root, ".go")) == 0 {
		return false
	}
	changed := checkkit.ChangedFiles()
	return len(paths) == 0 || changed == nil || touchesRequirements(changed)
}

// touchesRequirements reports whether any changed path is a requirement doc
// or a Go test file.
func touchesRequirements(changed []string) bool {
	for _, f := range changed {
		if strings.HasPrefix(f, docsDir) || strings.HasSuffix(f, "_test.go") {
			return true
		}
	}
	return false
}

// inScope marks the test files whose every test must own an acceptance: all
// of them in a full run, only the changed ones in a scoped run. Each test
// file is judged on its own, so the audit can land one package at a time.
func inScope(files []string) map[string]bool {
	scope := map[string]bool{}
	for _, f := range walkfs.InScope(files) {
		scope[f] = true
	}
	return scope
}

// missingDocs fails a Go repo that has no requirement docs.
func missingDocs(paths []string) []string {
	if len(paths) > 0 {
		return nil
	}
	return []string{docsDir + " has no requirement docs; add NNNN-kebab-title.md files from the template in the fitness-check-requirements README"}
}

// idErrors flags a requirement ID that two docs share, and any doc or
// acceptance ID that history shows was removed before.
func idErrors(docs []reqDoc, hist history) []string {
	var errs []string
	owners := map[string][]string{}
	for _, d := range docs {
		owners[d.id] = append(owners[d.id], d.path)
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
func reusedErrors(d reqDoc, hist history) []string {
	var errs []string
	for _, id := range d.ids() {
		if hist.removed[id.id] {
			errs = append(errs, at(d.path, id.line, id.id+" was deleted before; IDs are never reused, so take the next free number"))
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
