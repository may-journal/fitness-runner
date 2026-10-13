// Command fitness-check-requirements enforces docs/requirements in Go and
// Swift repos: each file is one requirement with a unique four-digit ID, a
// Why, a Measurement ratio with one cited source, and acceptances written as
// a single Given, When, and Then chain. Every acceptance is owned by exactly
// one test named for it (Go Test0001_1 or Swift test0001_1 owns 0001.1), and
// every test must own one, so a test that proves no requirement is either a
// missing requirement or a duplicate. Git history supplies removed IDs, so an
// ID is never reused. Two docs that cover one area fail, per ADR 0004. A staged or pull request change to a requirement test
// must change its doc too, and the reverse, unless a commit trailer exempts it.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "requirements", ContextInlineArg: "--message"},
		Run:      run,
	})
}

// docsDir holds the requirement docs, one per file.
const docsDir = "docs/requirements/"

func run(root string, _ []string) (checkkit.Result, error) {
	paths := docPaths(root)
	tests, files, repo, err := collectTests(root)
	if err != nil || repo == (langs{}) {
		return checkkit.Pass(0), err
	}
	if _, ok := checkkit.CtxMessage(); ok {
		return result(len(paths)+len(files), pairErrors(root, paths, files, tests, commitChanges(root))), nil
	}
	errs, err := docErrors(root, paths, tests, repo)
	if err != nil {
		return checkkit.Result{}, err
	}
	return result(len(paths)+len(files), append(errs, scopedPairs(root, paths, files, tests)...)), nil
}

// docErrors judges every doc's shape, its IDs against history, docs that
// share an area, and which tests own its acceptances.
func docErrors(root string, paths []string, tests []goTest, repo langs) ([]string, error) {
	docs, errs, err := requirement.Load(root, paths)
	if err != nil {
		return nil, err
	}
	errs = append(errs, missingDocs(paths)...)
	errs = append(errs, idErrors(docs, readHistory(root))...)
	errs = append(errs, overlapErrors(docs)...)
	return append(errs, ownershipErrors(docs, tests, repo)...), nil
}

// result passes with no errors and fails with any.
func result(files int, errs []string) checkkit.Result {
	if len(errs) > 0 {
		return checkkit.Fail(files, errs...)
	}
	return checkkit.Pass(files)
}

// collectTests reads every Go and Swift test, the files that hold them, and
// which languages the repo tests in.
func collectTests(root string) ([]goTest, []string, langs, error) {
	goFiles := testFiles(root)
	tests, err := scanTests(root, goFiles)
	if err != nil {
		return nil, nil, langs{}, err
	}
	swiftFiles := walkfs.FilesByExt(root, ".swift")
	swiftTests, err := scanSwift(root, swiftFiles)
	repo := langs{goCode: len(walkfs.FilesByExt(root, ".go")) > 0, swift: len(swiftFiles) > 0}
	return append(tests, swiftTests...), append(goFiles, swiftTestFiles(swiftTests)...), repo, err
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

// missingDocs fails a Go repo that has no requirement docs.
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
