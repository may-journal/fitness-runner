// cspell:ignore Elts
package golang

import (
	"go/ast"
	"go/token"
	"maps"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/lang"
)

// tables maps each type a test folder declares at its top level, or a test
// declares in its body, to its definition. A case table's type resolves
// through it.
type tables map[string]ast.Expr

// add records the top-level types one parsed file declares.
func (named tables) add(file *ast.File) {
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			named.specs(gen.Specs)
		}
	}
}

// specs records the types among specs.
func (named tables) specs(specs []ast.Spec) {
	for _, s := range specs {
		if spec, ok := s.(*ast.TypeSpec); ok {
			named[spec.Name.Name] = spec.Type
		}
	}
}

// with is named plus the types a test's body declares, which shadow the
// folder's own.
func (named tables) with(body *ast.BlockStmt) tables {
	out := maps.Clone(named)
	ast.Inspect(body, func(n ast.Node) bool {
		if decl, ok := n.(*ast.DeclStmt); ok {
			out.specs(decl.Decl.(*ast.GenDecl).Specs)
		}
		return true
	})
	return out
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

// anyStruct accepts every struct, since a map table's keys name its rows.
func anyStruct(*ast.StructType) bool { return true }

// isName reports an identifier spelled name, in any case.
func isName(id *ast.Ident) bool {
	return strings.EqualFold(id.Name, "name")
}

// cases lists the named cases inside a test's body in source order. A case
// is a t.Run given a literal name, or a case table's row.
func cases(fset *token.FileSet, file string, body *ast.BlockStmt, named tables) []lang.Case {
	named = named.with(body)
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

// rows adds the name of each row of a case table.
func (named tables) rows(lit *ast.CompositeLit, add func(string, token.Pos)) {
	byKey, ok := named.table(lit.Type)
	if !ok {
		return
	}
	for _, row := range lit.Elts {
		literal(rowName(row, byKey), add)
	}
}

// table reads a case table's type. Its rows are structs or pointers to
// them in a slice, named by the first field, or in a map keyed by name.
func (named tables) table(typ ast.Expr) (byKey, ok bool) {
	switch t := named.resolve(typ).(type) {
	case *ast.ArrayType:
		return false, named.isRow(t.Elt, namesFirst)
	case *ast.MapType:
		key, isIdent := t.Key.(*ast.Ident)
		return true, isIdent && key.Name == "string" && named.isRow(t.Value, anyStruct)
	}
	return false, false
}

// isRow reports whether a row type, or the type it points to, is a struct
// that accept takes.
func (named tables) isRow(typ ast.Expr, accept func(*ast.StructType) bool) bool {
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	row, ok := named.resolve(typ).(*ast.StructType)
	return ok && accept(row)
}

// rowName is the value naming a row. That is a map row's key, else the
// value keyed name, or the first value when none is keyed.
func rowName(row ast.Expr, byKey bool) ast.Expr {
	if kv, ok := row.(*ast.KeyValueExpr); ok && byKey {
		return kv.Key
	}
	if lit, ok := ast.Unparen(deref(row)).(*ast.CompositeLit); ok && !byKey {
		return fieldName(lit)
	}
	return nil
}

// deref is the struct literal a row such as &tc{...} points to.
func deref(row ast.Expr) ast.Expr {
	if ref, ok := row.(*ast.UnaryExpr); ok {
		return ref.X
	}
	return row
}

// fieldName is the value a struct literal gives its name field.
func fieldName(row *ast.CompositeLit) ast.Expr {
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
