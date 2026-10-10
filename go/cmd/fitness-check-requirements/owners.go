package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// goTest is one test declaration, Go or Swift, and the acceptance ID its
// name carries, if any.
type goTest struct {
	name  string
	id    string
	loc   string
	file  string
	swift bool
}

// langs records which languages a repo is written in, to name tests its way.
type langs struct {
	goCode bool
	swift  bool
}

// testName is the test name that owns id in the repo's languages: Test0001_1
// in Go, test0001_1 in Swift.
func (l langs) testName(id string) string {
	suffix := strings.Replace(id, ".", "_", 1)
	var names []string
	if l.goCode || !l.swift {
		names = append(names, "Test"+suffix)
	}
	if l.swift {
		names = append(names, "test"+suffix)
	}
	return strings.Join(names, " or ")
}

var (
	// testFunc matches the names go test runs: Test, then no lowercase letter.
	testFunc = regexp.MustCompile(`^Test($|[^a-z])`)
	// testID matches a test named for an acceptance: Test0001_1 is 0001.1.
	testID = regexp.MustCompile(`^Test(\d{4})_([1-9]\d*)$`)
)

// testFiles lists every tracked Go test file.
func testFiles(root string) []string {
	return walkfs.FilesByExt(root, "_test.go")
}

// scanTests parses every test file and returns its test declarations.
func scanTests(root string, files []string) ([]goTest, error) {
	fset := token.NewFileSet()
	var tests []goTest
	for _, f := range files {
		parsed, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(f)), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		tests = append(tests, fileTests(fset, f, parsed)...)
	}
	return tests, nil
}

// fileTests returns one file's top-level test functions.
func fileTests(fset *token.FileSet, file string, parsed *ast.File) []goTest {
	var tests []goTest
	for _, decl := range parsed.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && isTest(fn) {
			loc := fmt.Sprintf("%s:%d", file, fset.Position(fn.Pos()).Line)
			tests = append(tests, goTest{name: fn.Name.Name, id: idOf(fn.Name.Name), loc: loc, file: file})
		}
	}
	return tests
}

// isTest reports whether fn is a function go test runs as a test.
func isTest(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	return fn.Recv == nil && name != "TestMain" && testFunc.MatchString(name) && fn.Type.Params.NumFields() == 1
}

// idOf returns the acceptance ID a test name carries, or "".
func idOf(name string) string {
	m := testID.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}

// ownershipErrors flags acceptances that no test or several tests own, tests
// named for an acceptance no doc defines, and tests in scope that name no
// acceptance at all.
func ownershipErrors(docs []reqDoc, tests []goTest, scope map[string]bool, repo langs) []string {
	current := map[string]bool{}
	owners := map[string][]string{}
	for _, t := range tests {
		owners[t.id] = append(owners[t.id], t.loc)
	}
	var errs []string
	for _, d := range docs {
		for _, a := range d.acceptances {
			current[a.id] = true
			errs = append(errs, ownerError(d.path, a, owners[a.id], repo)...)
		}
	}
	for _, t := range tests {
		errs = append(errs, testError(t, current, scope)...)
	}
	return errs
}

// ownerError requires exactly one test to own the acceptance.
func ownerError(path string, a idLine, owners []string, repo langs) []string {
	name := repo.testName(a.id)
	switch len(owners) {
	case 1:
		return nil
	case 0:
		return []string{at(path, a.line, fmt.Sprintf("%s has no test; name exactly one test %s", a.id, name))}
	}
	return []string{at(path, a.line, fmt.Sprintf("%s is owned by %d tests (%s); keep exactly one", a.id, len(owners), strings.Join(owners, ", ")))}
}

// testError flags a test named for an undefined acceptance, and an in-scope
// test that names none.
func testError(t goTest, current, scope map[string]bool) []string {
	switch {
	case t.id != "" && !current[t.id]:
		return []string{fmt.Sprintf("%s: %s names %s, which no requirement defines; delete the test with its acceptance", t.loc, t.name, t.id)}
	case t.id == "" && scope[t.file]:
		return []string{fmt.Sprintf("%s: %s proves no requirement; rename it %s for the acceptance it proves, or delete it if another test covers that", t.loc, t.name, langs{swift: t.swift}.testName("NNNN.N"))}
	}
	return nil
}
