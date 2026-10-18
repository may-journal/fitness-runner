package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
)

// testScan is what one parse of every test file finds: named-helper
// failures and, per package, which tests the audit may run side by side.
type testScan struct {
	failures []string
	parallel map[string]map[string]bool
}

func scanTests(packages []Package) (testScan, error) {
	scan := testScan{parallel: map[string]map[string]bool{}}
	for _, pkg := range packages {
		if err := scan.add(pkg); err != nil {
			return scan, err
		}
	}
	sort.Strings(scan.failures)
	return scan, nil
}

// add scans pkg's test files. A package with TestMain gets no parallel
// tests, because each audit process would run its TestMain at once.
func (s *testScan) add(pkg Package) error {
	parallel := map[string]bool{}
	hasMain := false
	for _, name := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
		main, err := s.scanFile(filepath.Join(pkg.Dir, name), parallel)
		if err != nil {
			return err
		}
		hasMain = hasMain || main
	}
	if hasMain {
		parallel = map[string]bool{}
	}
	s.parallel[pkg.ImportPath] = parallel
	return nil
}

// scanFile records path's named-helper failures and parallel tests, and
// reports whether it declares TestMain.
func (s *testScan) scanFile(path string, parallel map[string]bool) (bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return false, err
	}
	hasMain := false
	for _, fn := range testFunctions(file) {
		s.failures = append(s.failures, namedCalls(fset, path, fn)...)
		parallel[fn.Name.Name] = callsParallel(fn)
		hasMain = hasMain || fn.Name.Name == "TestMain"
	}
	return hasMain, nil
}

// testFunctions lists file's top-level test functions with bodies.
func testFunctions(file *ast.File) []*ast.FuncDecl {
	var tests []*ast.FuncDecl
	for _, decl := range file.Decls {
		if fn := testFunction(decl); fn != nil {
			tests = append(tests, fn)
		}
	}
	return tests
}

func testFunction(decl ast.Decl) *ast.FuncDecl {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || !testName.MatchString(fn.Name.Name) || fn.Body == nil {
		return nil
	}
	return fn
}

func namedCalls(fset *token.FileSet, path string, fn *ast.FuncDecl) []string {
	var failures []string
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, helper, ok := namedWrapperCall(node)
		if ok {
			line := fset.Position(call.Pos()).Line
			failures = append(failures, fmt.Sprintf("test wrapper: %s:%d: %s passes named helper %s to t.Run; keep independently measured tests or remove the duplicated body", filepath.Base(path), line, fn.Name.Name, helper))
		}
		return true
	})
	return failures
}

func namedWrapperCall(node ast.Node) (*ast.CallExpr, string, bool) {
	call, ok := node.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return nil, "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Run" {
		return nil, "", false
	}
	helper, ok := call.Args[1].(*ast.Ident)
	return call, helperName(helper), ok
}

func helperName(helper *ast.Ident) string {
	if helper == nil {
		return ""
	}
	return helper.Name
}
