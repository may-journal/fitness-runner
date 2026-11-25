// cspell:ignore Elts
package golang

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/lang"
)

// tables maps each type a test folder declares to its definition, so a
// case table's type resolves through named types.
type tables map[string]ast.Expr

// add records the types one parsed file declares.
func (named tables) add(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		if spec, ok := n.(*ast.TypeSpec); ok {
			named[spec.Name.Name] = spec.Type
		}
		return true
	})
}

// resolve follows a type name to its definition, once, since a case table
// names its row type or its own slice type directly.
func (named tables) resolve(typ ast.Expr) ast.Expr {
	if id, ok := typ.(*ast.Ident); ok && named[id.Name] != nil {
		return named[id.Name]
	}
	return typ
}

// namesFirst reports whether a struct's first field is a string called
// name, as a case table's row type declares.
func namesFirst(st *ast.StructType) bool {
	if len(st.Fields.List) == 0 || len(st.Fields.List[0].Names) == 0 {
		return false
	}
	first := st.Fields.List[0]
	typ, ok := first.Type.(*ast.Ident)
	return ok && typ.Name == "string" && isName(first.Names[0])
}

// isName reports an identifier spelled name, in any case.
func isName(id *ast.Ident) bool {
	return strings.EqualFold(id.Name, "name")
}

// cases lists the named cases inside a test's body in source order. A case
// is a t.Run given a literal name, or a case table's row.
func cases(fset *token.FileSet, file string, body *ast.BlockStmt, named tables) []lang.Case {
	var out []lang.Case
	add := func(name string, at token.Pos) {
		out = append(out, lang.NewCase(name, file, fset.Position(at).Line))
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			runName(n, add)
		case *ast.CompositeLit:
			named.rows(n, add)
		}
		return true
	})
	return out
}

// runName adds the literal name a call such as t.Run("name", f) gives.
func runName(call *ast.CallExpr, add func(string, token.Pos)) {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Run" && len(call.Args) == 2 {
		literal(call.Args[0], add)
	}
}

// rows adds the name of each row of a case table. A case table is a slice
// or array of structs whose first field is the case's name.
func (named tables) rows(lit *ast.CompositeLit, add func(string, token.Pos)) {
	if !named.isTable(lit.Type) {
		return
	}
	for _, row := range lit.Elts {
		if row, ok := row.(*ast.CompositeLit); ok {
			literal(rowName(row), add)
		}
	}
}

// isTable reports whether a type is a case table's.
func (named tables) isTable(typ ast.Expr) bool {
	slice, ok := named.resolve(typ).(*ast.ArrayType)
	if !ok {
		return false
	}
	row, ok := named.resolve(slice.Elt).(*ast.StructType)
	return ok && namesFirst(row)
}

// rowName is the value a row gives its name: the value keyed name, or its
// first value when its values are not keyed.
func rowName(row *ast.CompositeLit) ast.Expr {
	for _, elt := range row.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return row.Elts[0]
		}
		if key, ok := kv.Key.(*ast.Ident); ok && isName(key) {
			return kv.Value
		}
	}
	return nil
}

// literal adds a string literal's value where it starts.
func literal(e ast.Expr, add func(string, token.Pos)) {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		value, _ := strconv.Unquote(lit.Value)
		add(value, lit.Pos())
	}
}
