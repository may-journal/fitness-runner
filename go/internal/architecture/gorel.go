package architecture

import (
	"go/ast"
	"go/types"
	"path"
)

// Kinds of relation a source shows between two declarations.
const (
	relCalls      = "calls"
	relReferences = "references"
	relEmbeds     = "embeds"
	relInherits   = "inherits from"
	relConforms   = "conforms to"
)

// use is one place in source that reaches a declaration at To.
type use struct {
	At, To Location
	Kind   string
}

// Import satisfies types.Importer. A folder of this repository is checked
// from source; any other import is an empty package, so its uses resolve
// to nothing and never to a declaration here.
func (g *goIndex) Import(p string) (*types.Package, error) {
	if pkg, ok := g.packages[p]; ok {
		return pkg, nil
	}
	folder, ok := g.folders.folders[p]
	if !ok || len(g.files[folder]) == 0 {
		return g.empty(p), nil
	}
	g.packages[p] = g.empty(p)
	conf := types.Config{Importer: g, Error: func(error) {}}
	pkg, _ := conf.Check(p, g.fset, g.files[folder], g.info)
	g.packages[p] = pkg
	return pkg, nil
}

func (g *goIndex) empty(p string) *types.Package {
	pkg := types.NewPackage(p, path.Base(p))
	pkg.MarkComplete()
	return pkg
}

// check type-checks one folder by its import path, or by the folder when
// no go.mod covers it.
func (g *goIndex) check(folder string) *types.Package {
	for p, f := range g.folders.folders {
		if f == folder {
			pkg, _ := g.Import(p)
			return pkg
		}
	}
	g.folders.folders[folder] = folder
	pkg, _ := g.Import(folder)
	return pkg
}

// uses lists every identifier in a folder's files that go/types resolves
// to a declaration in this repository.
func (g *goIndex) uses(folder string) []use {
	g.check(folder)
	var uses []use
	for _, f := range g.files[folder] {
		roles := goRoles(f)
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				uses = g.use(id, roles, uses)
			}
			return true
		})
	}
	return uses
}

func (g *goIndex) use(id *ast.Ident, roles map[*ast.Ident]string, uses []use) []use {
	obj := g.info.Uses[id]
	if obj == nil || !obj.Pos().IsValid() {
		return uses
	}
	to, at := g.fset.Position(obj.Pos()), g.fset.Position(id.Pos())
	kind := roles[id]
	if kind == "" {
		kind = relReferences
	}
	return append(uses, use{At: Location{at.Filename, at.Line}, To: Location{to.Filename, to.Line}, Kind: kind})
}

// goRoles marks the identifiers a call names and those an embedded field
// or interface names.
func goRoles(f *ast.File) map[*ast.Ident]string {
	roles := map[*ast.Ident]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			roles[callee(n.Fun)] = relCalls
		case *ast.StructType:
			embeddedIn(n.Fields, roles)
		case *ast.InterfaceType:
			embeddedIn(n.Methods, roles)
		}
		return true
	})
	delete(roles, nil)
	return roles
}

// callee is the identifier a call expression names.
func callee(fun ast.Expr) *ast.Ident {
	switch f := fun.(type) {
	case *ast.Ident:
		return f
	case *ast.SelectorExpr:
		return f.Sel
	case *ast.IndexExpr:
		return callee(f.X)
	case *ast.ParenExpr:
		return callee(f.X)
	}
	return nil
}

func embeddedIn(list *ast.FieldList, roles map[*ast.Ident]string) {
	for _, f := range orEmpty(list).List {
		embedded(f, roles)
	}
}

func embedded(f *ast.Field, roles map[*ast.Ident]string) {
	if _, method := f.Type.(*ast.FuncType); len(f.Names) > 0 || method {
		return
	}
	ast.Inspect(f.Type, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			roles[id] = relEmbeds
		}
		return true
	})
}

// implements reports whether a Go type, or a pointer to it, satisfies an
// interface, by go/types.
func (g *goIndex) implements(t, i *decl) bool {
	typ, iface := g.object(t), g.object(i)
	if typ == nil || iface == nil {
		return false
	}
	under, ok := iface.Type().Underlying().(*types.Interface)
	return ok && (types.Implements(typ.Type(), under) || types.Implements(types.NewPointer(typ.Type()), under))
}

func (g *goIndex) object(d *decl) types.Object {
	if pkg := g.check(d.folder()); pkg != nil {
		return pkg.Scope().Lookup(d.Name)
	}
	return nil
}

// usesIn lists the uses on lines from through to of a Go file, reading its
// folder once.
func (g *goIndex) usesIn(file string, from, to int) []use {
	folder := path.Dir(file)
	if _, ok := g.cache[folder]; !ok {
		g.cache[folder] = g.uses(folder)
	}
	var found []use
	for _, u := range g.cache[folder] {
		if inLines(u.At, file, from, to) {
			found = append(found, u)
		}
	}
	return found
}

// callee is the Go function or method a call use reaches.
func (g *goIndex) callee(u use) *decl {
	if u.Kind != relCalls {
		return nil
	}
	for _, d := range g.decls {
		if declaredAt(d, u.To) {
			return d
		}
	}
	return nil
}

func inLines(at Location, file string, from, to int) bool {
	return at.Path == file && from <= at.Line && at.Line <= to
}

// declaredAt reports a Go function or method declared at a line.
func declaredAt(d *decl, at Location) bool {
	return d.File == at.Path && d.Line == at.Line && d.callable() && d.Package != ""
}
