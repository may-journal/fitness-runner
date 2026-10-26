package release

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// topicDirective marks a package whose doc comments `fitness docs` serves
// as a topic: a `//fitness:topic <name>` line in its package doc comment.
// Go leaves directive lines out of the doc text.
const topicDirective = "//fitness:topic "

// topicDoc is one `fitness docs` topic, read from a package's code.
type topicDoc struct {
	// Name is the topic name its directive gives.
	Name string
	// Package is the package path under go/.
	Package string
	// Summary is the first sentence of the package doc comment.
	Summary string
	// Body is the package doc comment and its exported declarations' doc
	// comments, as Markdown.
	Body string
}

// topics reads every package under go/internal, outside the folders the go
// tool ignores, whose doc comment carries a topic directive, in topic name
// order. Two packages naming one topic, or a topic package without a doc
// comment, is an error.
func (c Config) topics() ([]topicDoc, error) {
	var topics []topicDoc
	err := filepath.WalkDir(filepath.Join(c.Root, "go", "internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if ignoredDir(entry.Name()) {
			return fs.SkipDir
		}
		t, ok, err := readTopic(c.Root, path)
		if ok {
			topics = append(topics, t)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(topics, func(a, b topicDoc) int { return strings.Compare(a.Name, b.Name) })
	for i := 1; i < len(topics); i++ {
		if topics[i].Name == topics[i-1].Name {
			return nil, fmt.Errorf("go/%s and go/%s both name topic %q", topics[i-1].Package, topics[i].Package, topics[i].Name)
		}
	}
	return topics, nil
}

// ignoredDir reports whether the go tool leaves a folder of this name out of
// ./...: testdata, and any name starting with _ or a dot.
func ignoredDir(name string) bool {
	return name == "testdata" || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".")
}

// readTopic reads the package in dir as a topic, reporting whether it is
// one. Only a topic's files are parsed whole; the rest stop at the package
// clause, where the directive sits.
func readTopic(root, dir string) (topicDoc, bool, error) {
	clauses, err := parseSources(token.NewFileSet(), dir, parser.PackageClauseOnly|parser.ParseComments)
	name := topicName(clauses)
	if err != nil || name == "" {
		return topicDoc{}, false, err
	}
	fset := token.NewFileSet()
	files, err := parseSources(fset, dir, parser.ParseComments)
	if err != nil {
		return topicDoc{}, true, err
	}
	t, err := buildTopic(fset, files, root, dir, name)
	return t, true, err
}

// buildTopic reads the documentation of the topic package in dir.
func buildTopic(fset *token.FileSet, files []*ast.File, root, dir, name string) (topicDoc, error) {
	rel, err := filepath.Rel(filepath.Join(root, "go"), dir)
	if err != nil {
		return topicDoc{}, err
	}
	rel = filepath.ToSlash(rel)
	pkg, err := doc.NewFromFiles(fset, files, "github.com/may-journal/fitness-runner/go/"+rel)
	if err == nil {
		err = documented(pkg, rel, name)
	}
	if err != nil {
		return topicDoc{}, err
	}
	return topicDoc{Name: name, Package: rel, Summary: pkg.Synopsis(pkg.Doc), Body: topicMarkdown(pkg)}, nil
}

// documented fails a topic package without a doc comment, or with a type
// whose exported field has none.
func documented(pkg *doc.Package, rel, name string) error {
	if strings.TrimSpace(pkg.Doc) == "" {
		return fmt.Errorf("go/%s names topic %q but has no package doc comment", rel, name)
	}
	for _, t := range pkg.Types {
		if strings.Contains(fieldDocs(t.Decl), undocumented) {
			return fmt.Errorf("go/%s names topic %q, but type %s has an exported field without a doc comment", rel, name, t.Name)
		}
	}
	return nil
}

// undocumented stands in for a missing field doc comment, which documented
// then refuses.
const undocumented = "(no doc comment)"

// topicName is the name a topic directive in a package doc comment gives,
// or "".
func topicName(files []*ast.File) string {
	for _, file := range files {
		if file.Doc == nil {
			continue
		}
		if name := directive(file.Doc); name != "" {
			return name
		}
	}
	return ""
}

// directive is the topic name a comment's directive line gives, or "".
func directive(group *ast.CommentGroup) string {
	for _, line := range group.List {
		if name, ok := strings.CutPrefix(line.Text, topicDirective); ok {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

// topicMarkdown renders a package's doc comment, then each exported
// constant and variable group, type with its methods, and function with its
// doc comment, as Markdown.
func topicMarkdown(pkg *doc.Package) string {
	var b strings.Builder
	b.WriteString(markdown(pkg, pkg.Doc))
	writeValues(&b, pkg, pkg.Consts)
	writeValues(&b, pkg, pkg.Vars)
	for _, t := range pkg.Types {
		writeType(&b, pkg, t)
	}
	writeFuncs(&b, pkg, pkg.Funcs, "")
	return b.String()
}

// writeType adds a type with its fields, then its constants, variables,
// constructors and methods.
func writeType(b *strings.Builder, pkg *doc.Package, t *doc.Type) {
	writeDecl(b, pkg, t.Name, t.Doc+fieldDocs(t.Decl))
	writeValues(b, pkg, t.Consts)
	writeValues(b, pkg, t.Vars)
	writeFuncs(b, pkg, t.Funcs, "")
	writeFuncs(b, pkg, t.Methods, t.Name+".")
}

// writeValues adds each constant group with its exported names.
func writeValues(b *strings.Builder, pkg *doc.Package, values []*doc.Value) {
	for _, value := range values {
		writeDecl(b, pkg, strings.Join(exported(value.Names), ", "), value.Doc)
	}
}

// writeFuncs adds each function, its name led by prefix, as a method by
// its type.
func writeFuncs(b *strings.Builder, pkg *doc.Package, funcs []*doc.Func, prefix string) {
	for _, fn := range funcs {
		writeDecl(b, pkg, prefix+fn.Name, fn.Doc)
	}
}

// writeDecl adds a declaration's heading and doc comment, unless it has
// no exported name or no doc comment.
func writeDecl(b *strings.Builder, pkg *doc.Package, name, text string) {
	if name == "" || strings.TrimSpace(text) == "" {
		return
	}
	fmt.Fprintf(b, "\n## %s\n\n%s", name, markdown(pkg, text))
}

// exported keeps the exported names.
func exported(names []string) []string {
	var out []string
	for _, name := range names {
		if token.IsExported(name) {
			out = append(out, name)
		}
	}
	return out
}

// fieldDocs lists a struct type's exported fields, nested ones by their path
// such as proseBudget.maxWords, with their JSON keys and doc comments, as a
// doc comment list, or "" for any other type.
func fieldDocs(decl *ast.GenDecl) string {
	var b strings.Builder
	for _, spec := range decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			writeFields(&b, ts.Type, "")
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\nFields:\n\n" + b.String()
}

// writeFields adds each exported field of the struct t names, and the
// fields nested in it, named from prefix.
func writeFields(b *strings.Builder, t ast.Expr, prefix string) {
	st, ok := structOf(t)
	if !ok {
		return
	}
	for _, field := range st.Fields.List {
		for _, name := range fieldNames(field) {
			fmt.Fprintf(b, "  - %s\n", keyLine(configKey{Key: prefix + name, Doc: orUndocumented(field, prefix+name)}))
			writeFields(b, field.Type, prefix+name+".")
		}
	}
}

// fieldNames are the exported names a field declares: its JSON key when it
// has one, else each Go name, or for an embedded field its type's name.
func fieldNames(field *ast.Field) []string {
	if key := tag(field, "json"); key != "" && key != "-" {
		return []string{key}
	}
	var names []string
	for _, ident := range field.Names {
		names = append(names, ident.Name)
	}
	if len(field.Names) == 0 {
		names = append(names, embeddedName(field.Type))
	}
	return exported(names)
}

// embeddedName is the type name an embedded field takes its name from.
func embeddedName(t ast.Expr) string {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return embeddedName(t.X)
	}
	return ""
}

// orUndocumented is a field's doc comment on one line, as explain shows
// config keys, opening with name in place of its key, or undocumented.
func orUndocumented(field *ast.Field, name string) string {
	text := fieldDoc(field)
	if text == "" {
		return undocumented
	}
	return renamed(configKey{Key: tag(field, "json"), Doc: text}, name).Doc
}
