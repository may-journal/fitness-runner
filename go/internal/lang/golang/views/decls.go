// Package views reads Go sources for the architecture views: declarations
// from syntax, the uses between them and interface satisfaction through
// go/types, and each line that imports another folder of the repository.
//
// The code view reads declarations with go/parser and relations with
// go/types: calls, references, embedding, and interface satisfaction.
// Every folder of the repository is checked from source, and other
// imports are left empty. A label's qualifier may name a Go package. The
// view covers types with an exported name. A Go type states no interface
// it satisfies, so go/types decides a realization, and its link must hold
// the type or one of its methods. A call into a function the view does
// not show is followed into that function, so a diagram may leave helpers
// out. A declaration redeclared under two build conditions resolves its
// uses to the first one go/types sees.
//
// The components view's references are each import of a folder in the
// repository, resolved through the nearest tracked go.mod, and each pkg.X
// use of it. A reference under //go:build keeps that condition, so an
// error says under go:build linux. A file that does not parse fails.
//
//fitness:container cli
//fitness:component kit Its language's checks share it.
package views

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
)

// Reader is the Go module's reader for the architecture views.
var Reader = codemodel.Reader{
	Language:   golang.Technology,
	Exts:       []string{golang.Ext},
	Read:       read,
	References: references,
	Strip:      codemodel.Strip,
}

// goIndex reads Go declarations from syntax and resolves uses with
// go/types. Every Go file outside tests is parsed into one file set, so a
// use resolves to a declaration's file and line.
type goIndex struct {
	x        *codemodel.Index
	own      []*codemodel.Decl
	repo     codemodel.Repo
	folders  *goFolders
	fset     *token.FileSet
	files    map[string][]*ast.File
	info     *types.Info
	packages map[string]*types.Package
	cache    map[string][]use
}

func read(repo codemodel.Repo, x *codemodel.Index) codemodel.Code {
	g := &goIndex{x: x, repo: repo, fset: token.NewFileSet(), files: map[string][]*ast.File{}, packages: map[string]*types.Package{}, cache: map[string][]use{}}
	g.folders = newFolders(repo)
	g.info = &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
	for _, file := range codemodel.FilesWith(repo.Code, golang.Ext) {
		g.parse(file)
	}
	return codemodel.Code{Relate: g.relate, Realizes: g.implements}
}

func (g *goIndex) parse(file string) {
	lines := g.repo.Read(file)
	condition := buildCondition(lines)
	if condition == "go:build ignore" {
		return // a generator run by hand is no part of its folder's package
	}
	parsed, err := parser.ParseFile(g.fset, file, strings.Join(lines, "\n"), parser.SkipObjectResolution)
	if err != nil {
		g.x.Errors = append(g.x.Errors, codemodel.Location{Path: file, Line: 1}.String()+": cannot parse "+err.Error())
		return
	}
	folder := path.Dir(file)
	g.folders.folder(folder)
	g.files[folder] = append(g.files[folder], parsed)
	r := goDecls{g: g, file: file, pkg: parsed.Name.Name, condition: condition}
	for _, d := range parsed.Decls {
		r.decl(d)
	}
}

// goDecls reads one file's top-level declarations.
type goDecls struct {
	g                    *goIndex
	file, pkg, condition string
}

// new records a declaration. Members join the type of their owner's name
// declared in the same package folder.
func (r goDecls) new(name, owner, kind string, node ast.Node) *codemodel.Decl {
	d := &codemodel.Decl{File: r.file, Name: name, Owner: owner, Kind: kind, Package: r.pkg, Scope: path.Dir(r.file) + " " + r.pkg, Condition: r.condition,
		Line: r.g.fset.Position(node.Pos()).Line, End: r.g.fset.Position(node.End()).Line}
	r.g.own = append(r.g.own, d)
	return r.g.x.Add(d)
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
	kind, owner := codemodel.KindFunc, ""
	if f.Recv != nil && len(f.Recv.List) > 0 {
		kind, owner = codemodel.KindMethod, receiverName(f.Recv.List[0].Type)
	}
	d := r.new(f.Name.Name, owner, kind, f)
	d.Params, d.Results = goParams(f.Type.Params), goResults(f.Type.Results)
}

// receiverName is a method receiver's or embedded field's base type,
// without pointer, package qualifier, or type parameters.
func receiverName(expr ast.Expr) string {
	switch t := baseType(expr).(type) {
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// baseType strips pointers and type arguments from a type expression.
func baseType(expr ast.Expr) ast.Expr {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseType(t.X)
	case *ast.IndexExpr:
		return baseType(t.X)
	case *ast.IndexListExpr:
		return baseType(t.X)
	}
	return expr
}

func (r goDecls) spec(spec ast.Spec) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		r.typeSpec(s)
	case *ast.ValueSpec:
		for _, name := range s.Names {
			r.new(name.Name, "", codemodel.KindValue, s)
		}
	}
}

func (r goDecls) typeSpec(s *ast.TypeSpec) {
	kind := codemodel.KindType
	switch t := s.Type.(type) {
	case *ast.StructType:
		r.fields(s.Name.Name, t.Fields)
	case *ast.InterfaceType:
		kind = codemodel.KindInterface
		r.interfaceMethods(s.Name.Name, t.Methods)
	}
	d := r.new(s.Name.Name, "", kind, s)
	d.Exported = ast.IsExported(s.Name.Name)
}

func (r goDecls) fields(owner string, list *ast.FieldList) {
	for _, f := range list.List {
		for _, name := range fieldNames(f) {
			r.new(name, owner, codemodel.KindField, f).Type = types.ExprString(f.Type)
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
		d := r.new(f.Names[0].Name, owner, codemodel.KindMethod, f)
		d.Params, d.Results = goParams(ft.Params), goResults(ft.Results)
	}
}

func goParams(list *ast.FieldList) []codemodel.Param {
	var params []codemodel.Param
	for _, f := range list.List {
		typ := types.ExprString(f.Type)
		if len(f.Names) == 0 {
			params = append(params, codemodel.Param{Type: typ})
		}
		for _, n := range f.Names {
			params = append(params, codemodel.Param{Name: n.Name, Type: typ})
		}
	}
	return params
}

func goResults(list *ast.FieldList) []string {
	var results []string
	for _, p := range goParams(orEmpty(list)) {
		results = append(results, p.Type)
	}
	return results
}

func orEmpty(list *ast.FieldList) *ast.FieldList {
	if list == nil {
		return &ast.FieldList{}
	}
	return list
}
