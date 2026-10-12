package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"
)

// goFolders maps the import path of every folder holding product Go files
// to that folder, through the nearest tracked go.mod.
type goFolders struct {
	m       *Model
	modules map[string]string
	folders map[string]string
}

// goReferences reads each product Go file's imports of folders in this
// repository and every use of their names, so each line that reaches
// another folder is known. A file that does not parse is an error.
func (m *Model) goReferences() ([]reference, []string) {
	g := &goFolders{m: m, modules: map[string]string{}, folders: map[string]string{}}
	g.readModules()
	files := m.productSources(".go")
	for _, file := range files {
		g.folder(path.Dir(file))
	}
	var refs []reference
	var errs []string
	for _, file := range files {
		refs, errs = g.file(file, refs, errs)
	}
	return refs, errs
}

func (g *goFolders) readModules() {
	for _, file := range g.m.repo.sorted() {
		if path.Base(file) == "go.mod" {
			g.module(file)
		}
	}
}

func (g *goFolders) module(file string) {
	for _, line := range g.m.repo.read(file) {
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

func (g *goFolders) file(file string, refs []reference, errs []string) ([]reference, []string) {
	lines := g.m.repo.read(file)
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, strings.Join(lines, "\n"), parser.SkipObjectResolution)
	if err != nil {
		return refs, append(errs, Location{file, 1}.String()+": cannot parse "+err.Error())
	}
	r := &goFile{fset: fset, file: file, condition: buildCondition(lines), names: map[string]string{}}
	return append(refs, r.references(g.folders, parsed)...), errs
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
func (f *goFile) references(folders map[string]string, parsed *ast.File) []reference {
	var refs []reference
	for _, spec := range parsed.Imports {
		refs = f.importSpec(folders, spec, refs)
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		refs = f.selector(n, refs)
		return true
	})
	return refs
}

func (f *goFile) importSpec(folders map[string]string, spec *ast.ImportSpec, refs []reference) []reference {
	folder, ok := folders[strings.Trim(spec.Path.Value, `"`)]
	if !ok {
		return refs
	}
	f.names[importName(spec, folder)] = folder
	return append(refs, f.reference(spec.Pos(), folder))
}

func importName(spec *ast.ImportSpec, folder string) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	return path.Base(folder)
}

func (f *goFile) selector(n ast.Node, refs []reference) []reference {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return refs
	}
	if id, ok := sel.X.(*ast.Ident); ok && f.names[id.Name] != "" {
		return append(refs, f.reference(id.Pos(), f.names[id.Name]))
	}
	return refs
}

func (f *goFile) reference(pos token.Pos, folder string) reference {
	return reference{At: Location{f.file, f.fset.Position(pos).Line}, To: folder, Condition: f.condition}
}
