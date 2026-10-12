package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"strings"
)

// goIndex reads Go declarations from syntax and resolves uses with
// go/types. Every product and generated Go file outside tests is parsed
// into one file set, so a use resolves to a declaration's file and line.
type goIndex struct {
	*declIndex
	folders  *goFolders
	fset     *token.FileSet
	files    map[string][]*ast.File
	info     *types.Info
	packages map[string]*types.Package
	cache    map[string][]use
}

func (m *Model) goIndex(x *declIndex) *goIndex {
	g := &goIndex{declIndex: x, fset: token.NewFileSet(), files: map[string][]*ast.File{}, packages: map[string]*types.Package{}, cache: map[string][]use{}}
	g.folders = &goFolders{m: m, modules: map[string]string{}, folders: map[string]string{}, packages: map[string]string{}}
	g.folders.readModules()
	g.info = &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
	for _, s := range m.Sources {
		if path.Ext(s.Path) == ".go" && s.Category != "test" {
			g.parse(m, s.Path)
		}
	}
	return g
}

func (g *goIndex) parse(m *Model, file string) {
	lines := m.repo.read(file)
	parsed, err := parser.ParseFile(g.fset, file, strings.Join(lines, "\n"), parser.SkipObjectResolution)
	if err != nil {
		g.errors = append(g.errors, Location{file, 1}.String()+": cannot parse "+err.Error())
		return
	}
	folder := path.Dir(file)
	g.folders.folder(folder)
	g.files[folder] = append(g.files[folder], parsed)
	r := goDecls{g: g, file: file, pkg: parsed.Name.Name, condition: buildCondition(lines)}
	for _, d := range parsed.Decls {
		r.decl(d)
	}
}

// goDecls reads one file's top-level declarations.
type goDecls struct {
	g                    *goIndex
	file, pkg, condition string
}

func (r goDecls) new(name, owner, kind string, node ast.Node) *decl {
	return r.g.add(&decl{File: r.file, Name: name, Owner: owner, Kind: kind, Package: r.pkg, Condition: r.condition,
		Line: r.g.fset.Position(node.Pos()).Line, End: r.g.fset.Position(node.End()).Line})
}

func (r goDecls) decl(d ast.Decl) {
	switch d := d.(type) {
	case *ast.FuncDecl:
		r.function(d)
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			r.spec(spec)
		}
	}
}

func (r goDecls) function(f *ast.FuncDecl) {
	kind, owner := kindFunc, ""
	if f.Recv != nil && len(f.Recv.List) > 0 {
		kind, owner = kindMethod, receiverName(f.Recv.List[0].Type)
	}
	d := r.new(f.Name.Name, owner, kind, f)
	d.Params, d.Results = goParams(f.Type.Params), goResults(f.Type.Results)
}

// receiverName is a method receiver's base type, without pointer or type
// parameters.
func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func (r goDecls) spec(spec ast.Spec) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		r.typeSpec(s)
	case *ast.ValueSpec:
		for _, name := range s.Names {
			r.new(name.Name, "", kindValue, s)
		}
	}
}

func (r goDecls) typeSpec(s *ast.TypeSpec) {
	kind := kindType
	switch t := s.Type.(type) {
	case *ast.StructType:
		r.fields(s.Name.Name, t.Fields)
	case *ast.InterfaceType:
		kind = kindInterface
		r.interfaceMethods(s.Name.Name, t.Methods)
	}
	d := r.new(s.Name.Name, "", kind, s)
	d.Exported = ast.IsExported(s.Name.Name)
}

func (r goDecls) fields(owner string, list *ast.FieldList) {
	for _, f := range list.List {
		for _, name := range fieldNames(f) {
			r.new(name, owner, kindField, f).Type = types.ExprString(f.Type)
		}
	}
}

// fieldNames lists a field's names; an embedded field is named by its type.
func fieldNames(f *ast.Field) []string {
	if len(f.Names) == 0 {
		return []string{receiverName(f.Type)}
	}
	names := make([]string, len(f.Names))
	for i, n := range f.Names {
		names[i] = n.Name
	}
	return names
}

func (r goDecls) interfaceMethods(owner string, list *ast.FieldList) {
	for _, f := range list.List {
		ft, ok := f.Type.(*ast.FuncType)
		if !ok || len(f.Names) == 0 {
			continue
		}
		d := r.new(f.Names[0].Name, owner, kindMethod, f)
		d.Params, d.Results = goParams(ft.Params), goResults(ft.Results)
	}
}

func goParams(list *ast.FieldList) []param {
	var params []param
	for _, f := range list.List {
		typ := types.ExprString(f.Type)
		if len(f.Names) == 0 {
			params = append(params, param{typ: typ})
		}
		for _, n := range f.Names {
			params = append(params, param{name: n.Name, typ: typ})
		}
	}
	return params
}

func goResults(list *ast.FieldList) []string {
	var results []string
	for _, p := range goParams(orEmpty(list)) {
		results = append(results, p.typ)
	}
	return results
}

func orEmpty(list *ast.FieldList) *ast.FieldList {
	if list == nil {
		return &ast.FieldList{}
	}
	return list
}
