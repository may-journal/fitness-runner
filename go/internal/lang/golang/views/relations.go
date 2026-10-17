package views

import (
	"go/ast"
	"go/types"
	"path"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
)

// use is one place in source that reaches a declaration at To.
type use struct {
	At, To codemodel.Location
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

// relate reads the uses in each Go node's lines through go/types, so
// comments, strings, and a name that merely matches never relate two
// nodes.
func (g *goIndex) relate(gr *codemodel.Graph) {
	for _, s := range gr.Spans() {
		if path.Ext(s.File) == golang.Ext {
			g.span(gr, s)
		}
	}
}

func (g *goIndex) span(gr *codemodel.Graph, s codemodel.Span) {
	visited := map[*codemodel.Decl]bool{}
	for _, u := range g.usesIn(s.File, s.From, s.To) {
		if gr.Owner(u.At) == s.Node {
			g.follow(gr, s.Node, u, u.At, visited)
		}
	}
}

// follow relates a node to the node a use reaches. A call to a function
// the view does not show is followed into that function, so a diagram may
// leave helpers out; the relation is reported at the node's own line.
func (g *goIndex) follow(gr *codemodel.Graph, n *codemodel.Node, u use, at codemodel.Location, visited map[*codemodel.Decl]bool) {
	if to := gr.Owner(u.To); to != nil {
		gr.Observe(n, to, u.Kind, at)
		return
	}
	helper := g.callee(u)
	if helper == nil || visited[helper] {
		return
	}
	visited[helper] = true
	for _, next := range g.usesIn(helper.File, helper.Line, helper.End) {
		g.follow(gr, n, next, at, visited)
	}
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
		kind = codemodel.RelReferences
	}
	return append(uses, use{At: codemodel.Location{Path: at.Filename, Line: at.Line}, To: codemodel.Location{Path: to.Filename, Line: to.Line}, Kind: kind})
}

// goRoles marks the identifiers a call names and those an embedded field
// or interface names.
func goRoles(f *ast.File) map[*ast.Ident]string {
	roles := map[*ast.Ident]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			roles[callee(n.Fun)] = codemodel.RelCalls
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
			roles[id] = codemodel.RelEmbeds
		}
		return true
	})
}

// implements reports whether a Go type, or a pointer to it, satisfies an
// interface, by go/types. A Go type states no interface it satisfies, so
// go/types decides a realization rather than the source's relations.
func (g *goIndex) implements(t, i *codemodel.Decl) bool {
	typ, iface := g.object(t), g.object(i)
	if typ == nil || iface == nil {
		return false
	}
	under, ok := iface.Type().Underlying().(*types.Interface)
	return ok && (types.Implements(typ.Type(), under) || types.Implements(types.NewPointer(typ.Type()), under))
}

func (g *goIndex) object(d *codemodel.Decl) types.Object {
	if pkg := g.check(d.Folder()); pkg != nil {
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
func (g *goIndex) callee(u use) *codemodel.Decl {
	if u.Kind != codemodel.RelCalls {
		return nil
	}
	for _, d := range g.own {
		if declaredAt(d, u.To) {
			return d
		}
	}
	return nil
}

// declaredAt reports a function or method declared at a line.
func declaredAt(d *codemodel.Decl, at codemodel.Location) bool {
	return d.File == at.Path && d.Line == at.Line && d.Callable()
}

func inLines(at codemodel.Location, file string, from, to int) bool {
	return at.Path == file && from <= at.Line && at.Line <= to
}
