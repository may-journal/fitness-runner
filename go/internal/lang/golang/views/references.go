package views

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
)

// goFolders maps the import path of every folder holding product Go files
// to that folder, through the nearest tracked go.mod.
type goFolders struct {
	repo     codemodel.Repo
	modules  map[string]string
	folders  map[string]string
	packages map[string]string
}

// goSource is one parsed product Go file.
type goSource struct {
	file   string
	lines  []string
	fset   *token.FileSet
	parsed *ast.File
}

func newFolders(repo codemodel.Repo) *goFolders {
	g := &goFolders{repo: repo, modules: map[string]string{}, folders: map[string]string{}, packages: map[string]string{}}
	g.readModules()
	return g
}

// references reads each product Go file's imports of folders in this
// repository and every use of their names, so each line that reaches
// another folder is known. A file that does not parse is an error.
func references(repo codemodel.Repo) ([]codemodel.Reference, []string) {
	g := newFolders(repo)
	files := codemodel.FilesWith(repo.Product, golang.Ext)
	for _, file := range files {
		g.folder(path.Dir(file))
	}
	sources, errs := g.parse(files)
	var refs []codemodel.Reference
	for _, src := range sources {
		r := &goFile{fset: src.fset, file: src.file, condition: buildCondition(src.lines), names: map[string]string{}}
		refs = append(refs, r.references(g, src.parsed)...)
	}
	return refs, errs
}

// parse reads every file and records each folder's package name, which an
// import uses unless it renames the package.
func (g *goFolders) parse(files []string) ([]goSource, []string) {
	var sources []goSource
	var errs []string
	for _, file := range files {
		lines := g.repo.Read(file)
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, strings.Join(lines, "\n"), parser.SkipObjectResolution)
		if err != nil {
			errs = append(errs, codemodel.Location{Path: file, Line: 1}.String()+": cannot parse "+err.Error())
			continue
		}
		g.packageName(path.Dir(file), parsed.Name.Name)
		sources = append(sources, goSource{file, lines, fset, parsed})
	}
	return sources, errs
}

func (g *goFolders) readModules() {
	for _, file := range g.repo.Files {
		if path.Base(file) == golang.Manifest {
			g.module(file)
		}
	}
}

func (g *goFolders) module(file string) {
	for _, line := range g.repo.Read(file) {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			g.modules[path.Dir(file)] = strings.TrimSpace(name)
			return
		}
	}
}

// folder records a folder's import path under the nearest module.
func (g *goFolders) folder(folder string) {
	for dir := folder; ; dir = path.Dir(dir) {
		if module, ok := g.modules[dir]; ok {
			g.folders[importPath(module, dir, folder)] = folder
			return
		}
		if dir == "." {
			return
		}
	}
}

func importPath(module, dir, folder string) string {
	if folder == dir {
		return module
	}
	return module + "/" + strings.TrimPrefix(folder, dir+"/")
}

// buildCondition reads a file's go:build line, which sits before its
// package clause.
func buildCondition(lines []string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			return ""
		}
		if expr, ok := strings.CutPrefix(line, "//go:build "); ok {
			return "go:build " + strings.TrimSpace(expr)
		}
	}
	return ""
}

type goFile struct {
	fset      *token.FileSet
	file      string
	condition string
	names     map[string]string
}

// references lists the import of each repository folder and each use of
// the name it was imported as.
func (f *goFile) references(g *goFolders, parsed *ast.File) []codemodel.Reference {
	var refs []codemodel.Reference
	for _, spec := range parsed.Imports {
		refs = f.importSpec(g, spec, refs)
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		refs = f.selector(n, refs)
		return true
	})
	return refs
}

func (f *goFile) importSpec(g *goFolders, spec *ast.ImportSpec, refs []codemodel.Reference) []codemodel.Reference {
	folder, ok := g.folders[strings.Trim(spec.Path.Value, `"`)]
	if !ok {
		return refs
	}
	f.names[g.importName(spec, folder)] = folder
	return append(refs, f.reference(spec.Pos(), folder))
}

// packageName keeps the first importable package name a folder declares.
// A main package, such as a generator under a build tag, or an external
// test package names nothing an import can use.
func (g *goFolders) packageName(folder, name string) {
	if name == "main" || strings.HasSuffix(name, "_test") || g.packages[folder] != "" {
		return
	}
	g.packages[folder] = name
}

// importName is the name a file uses for an import: its rename, else the
// folder's package name.
func (g *goFolders) importName(spec *ast.ImportSpec, folder string) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	if name := g.packages[folder]; name != "" {
		return name
	}
	return path.Base(folder)
}

func (f *goFile) selector(n ast.Node, refs []codemodel.Reference) []codemodel.Reference {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return refs
	}
	if id, ok := sel.X.(*ast.Ident); ok && f.names[id.Name] != "" {
		return append(refs, f.reference(id.Pos(), f.names[id.Name]))
	}
	return refs
}

func (f *goFile) reference(pos token.Pos, folder string) codemodel.Reference {
	return codemodel.Reference{At: codemodel.Location{Path: f.file, Line: f.fset.Position(pos).Line}, To: folder, Condition: f.condition}
}
