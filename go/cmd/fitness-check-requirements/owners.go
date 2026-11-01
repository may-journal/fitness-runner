package check

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/requirement"
)

// reqTest is one test declaration a language module found, and the module
// whose naming it follows.
type reqTest struct {
	lang.Test
	module lang.Module
}

// testName is the test name that owns id in the given languages, such as
// Test0001_1 or test0001_1, joined with "or".
func testName(written []lang.Module, id string) string {
	names := make([]string, 0, len(written))
	for _, m := range written {
		names = append(names, m.Tests.Name(id))
	}
	return strings.Join(names, " or ")
}

// ownershipErrors flags acceptances owned by no test or several, and tests
// naming no defined acceptance, skipping docs in proven.
func ownershipErrors(docs []requirement.Document, tests []reqTest, written []lang.Module, proven map[string]bool) []string {
	current := map[string]bool{}
	owners := map[string][]string{}
	for _, t := range tests {
		owners[t.ID] = append(owners[t.ID], t.Loc)
	}
	var errs []string
	for _, d := range docs {
		errs = append(errs, docOwners(d, owners, written, proven[d.ID], current)...)
	}
	for _, t := range tests {
		errs = append(errs, testError(t, current)...)
	}
	return errs
}

// docOwners judges who owns each of the doc's acceptances, marking each
// current. One a check proves needs no test, but two tests still fail.
func docOwners(d requirement.Document, owners map[string][]string, written []lang.Module, proven bool, current map[string]bool) []string {
	var errs []string
	for _, a := range d.Acceptances {
		current[a.ID] = true
		if len(owners[a.ID]) > 0 || !proven {
			errs = append(errs, ownerError(d.Path, a, owners[a.ID], written)...)
		}
	}
	return errs
}

// ownerError requires exactly one test to own the acceptance.
func ownerError(path string, a requirement.IDLine, owners []string, written []lang.Module) []string {
	name := testName(written, a.ID)
	switch len(owners) {
	case 1:
		return nil
	case 0:
		return []string{requirement.At(path, a.Line, fmt.Sprintf("%s has no test; name exactly one test %s, or prove the doc with a check in %s%s/", a.ID, name, reqPrefix, a.ID[:4]))}
	}
	return []string{requirement.At(path, a.Line, fmt.Sprintf("%s is owned by %d tests (%s); keep exactly one", a.ID, len(owners), strings.Join(owners, ", ")))}
}

// testError flags a test named for an undefined acceptance, and a
// test that names none.
func testError(t reqTest, current map[string]bool) []string {
	switch {
	case t.ID != "" && !current[t.ID]:
		return []string{fmt.Sprintf("%s: %s names %s, which no requirement defines; delete the test with its acceptance", t.Loc, t.Name, t.ID)}
	case t.ID == "":
		return []string{fmt.Sprintf("%s: %s proves no requirement; rename it %s for the acceptance it proves, or delete it if another test covers that", t.Loc, t.Name, t.module.Tests.Name("NNNN.N"))}
	}
	return nil
}

// toolErrors asks each written module that judges test tools about its test
// files, such as a Swift test importing XCTest.
func toolErrors(root string, tests []reqTest, written []lang.Module) ([]string, error) {
	var errs []string
	for _, m := range written {
		if m.Tests.Tools == nil {
			continue
		}
		found, err := m.Tests.Tools(root, moduleFiles(tests, m.Name))
		if err != nil {
			return nil, err
		}
		errs = append(errs, found...)
	}
	return errs, nil
}

// moduleFiles lists, once each in order, the files holding the named
// module's tests.
func moduleFiles(tests []reqTest, module string) []string {
	var files []string
	seen := map[string]bool{}
	for _, t := range tests {
		if t.module.Name == module && !seen[t.File] {
			seen[t.File] = true
			files = append(files, t.File)
		}
	}
	return files
}
