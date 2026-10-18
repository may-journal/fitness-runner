package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// parallelTests names pkg's top-level tests that call t.Parallel() in their
// own body. go test runs only those alongside other tests, so the audit does
// the same.
func parallelTests(pkg Package) (map[string]bool, error) {
	found := map[string]bool{}
	for _, name := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkg.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		addParallel(found, file)
	}
	return found, nil
}

// addParallel records file's top-level tests that call t.Parallel().
func addParallel(found map[string]bool, file *ast.File) {
	for _, decl := range file.Decls {
		if fn := testFunction(decl); fn != nil && callsParallel(fn) {
			found[fn.Name.Name] = true
		}
	}
}

// callsParallel reports whether a statement of fn's body is a Parallel()
// call on its parameter.
func callsParallel(fn *ast.FuncDecl) bool {
	for _, stmt := range fn.Body.List {
		if parallelCall(stmt) {
			return true
		}
	}
	return false
}

func parallelCall(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Parallel"
}
