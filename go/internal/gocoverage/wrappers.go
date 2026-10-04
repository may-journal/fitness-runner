package gocoverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
)

func namedWrapperFailures(packages []Package) ([]string, error) {
	var failures []string
	for _, pkg := range packages {
		for _, name := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
			found, err := scanNamedWrappers(filepath.Join(pkg.Dir, name))
			if err != nil {
				return nil, err
			}
			failures = append(failures, found...)
		}
	}
	sort.Strings(failures)
	return failures, nil
}

func scanNamedWrappers(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	var failures []string
	for _, decl := range file.Decls {
		fn := testFunction(decl)
		if fn == nil {
			continue
		}
		failures = append(failures, namedCalls(fset, path, fn)...)
	}
	return failures, nil
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
