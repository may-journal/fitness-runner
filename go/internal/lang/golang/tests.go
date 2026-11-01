package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Tests finds go test requirement tests: func Test0001_1(t *testing.T) owns
// 0001.1, read with go/ast so only real top-level test functions count. A file
// such as 0003_0004_test.go is named for both requirements, one such as
// tools_test.go for none.
var Tests = lang.Tests{
	Scan:    scan,
	Name:    naming.Name,
	Source:  func(file string) bool { return strings.HasSuffix(strings.ToLower(file), testSuffix) },
	Numbers: numbers,
	Reads:   func(file string) bool { return strings.HasSuffix(file, testSuffix) },
}

// naming names go test requirement tests: Test0001_1 owns 0001.1.
const naming = lang.TestPrefix("Test")

// testSuffix ends every file go test builds only for tests.
const testSuffix = "_test" + Ext

var (
	// numberedTest matches a test file named for its requirements:
	// 0014_test.go, or 0003_0004_test.go for two.
	numberedTest = regexp.MustCompile(`^(\d{4}(?:_\d{4})*)` + regexp.QuoteMeta(testSuffix) + `$`)
	// testFunc matches the names go test runs: Test, then no lowercase letter.
	testFunc = regexp.MustCompile(`^Test($|[^a-z])`)
)

// scan reads every tracked Go test file's tests; a repo with any Go file
// is written in Go.
func scan(root string) ([]lang.Test, []string, bool, error) {
	files := walkfs.FilesByExt(root, testSuffix)
	tests, err := scanTests(root, files)
	if err != nil {
		return nil, nil, false, err
	}
	return tests, files, len(walkfs.FilesByExt(root, Ext)) > 0, nil
}

// scanTests parses every test file and returns its test declarations.
func scanTests(root string, files []string) ([]lang.Test, error) {
	fset := token.NewFileSet()
	var tests []lang.Test
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
func fileTests(fset *token.FileSet, file string, parsed *ast.File) []lang.Test {
	var tests []lang.Test
	for _, decl := range parsed.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && isTest(fn) {
			loc := fmt.Sprintf("%s:%d", file, fset.Position(fn.Pos()).Line)
			tests = append(tests, lang.Test{Name: fn.Name.Name, ID: naming.ID(fn.Name.Name), Loc: loc, File: file})
		}
	}
	return tests
}

// numbers returns the requirement numbers a test file's name carries.
func numbers(file string) []string {
	m := numberedTest.FindStringSubmatch(path.Base(file))
	if m == nil {
		return nil
	}
	return strings.Split(m[1], "_")
}

// isTest reports whether fn is a function go test runs as a test.
func isTest(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	return fn.Recv == nil && name != "TestMain" && testFunc.MatchString(name) && fn.Type.Params.NumFields() == 1
}
