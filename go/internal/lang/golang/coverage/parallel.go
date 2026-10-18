package coverage

import "go/ast"

// callsParallel reports whether a statement of fn's own body calls Parallel()
// on fn's testing parameter. go test runs only such tests alongside others;
// a call nested in a branch or a helper is not counted, so that test stays
// serial.
func callsParallel(fn *ast.FuncDecl) bool {
	param := testParam(fn)
	if param == "" {
		return false
	}
	for _, stmt := range fn.Body.List {
		if parallelCall(stmt, param) {
			return true
		}
	}
	return false
}

// testParam names fn's single parameter, or returns "".
func testParam(fn *ast.FuncDecl) string {
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) != 1 {
		return ""
	}
	return params[0].Names[0].Name
}

// parallelCall reports whether stmt is exactly param.Parallel().
func parallelCall(stmt ast.Stmt, param string) bool {
	selector, ok := methodCall(stmt)
	if !ok || selector.Sel.Name != "Parallel" {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == param
}

// methodCall returns the selector of a statement that is a call with no
// arguments.
func methodCall(stmt ast.Stmt) (*ast.SelectorExpr, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return selector, ok
}
