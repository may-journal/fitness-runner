// Package comments reads the comment blocks of source files, so every check
// reads them the same way.
package comments

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Types maps each type in src, keyed Type, and each struct field, keyed
// Type.Field.Inner, to its doc comment as one line.
func Types(src []byte) (map[string]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	docs := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		addTypes(docs, gen)
	}
	return docs, nil
}

// addTypes records each type in a type declaration, with its fields. A
// lone type's comment may sit on the declaration rather than the spec.
func addTypes(docs map[string]string, gen *ast.GenDecl) {
	for _, s := range gen.Specs {
		spec := s.(*ast.TypeSpec)
		doc := spec.Doc
		if doc == nil {
			doc = gen.Doc
		}
		docs[spec.Name.Name] = oneLine(doc)
		if st, ok := spec.Type.(*ast.StructType); ok {
			addFields(docs, spec.Name.Name, st)
		}
	}
}

// addFields records each named field of st under prefix, then the fields
// of each struct nested in one.
func addFields(docs map[string]string, prefix string, st *ast.StructType) {
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			key := prefix + "." + name.Name
			docs[key] = text(field)
			if inner, ok := field.Type.(*ast.StructType); ok {
				addFields(docs, key, inner)
			}
		}
	}
}

// text is a field's doc comment, else its line comment, as one line.
func text(field *ast.Field) string {
	group := field.Doc
	if group == nil {
		group = field.Comment
	}
	return oneLine(group)
}

// oneLine is a comment's text with its lines joined.
func oneLine(group *ast.CommentGroup) string {
	return strings.Join(strings.Fields(group.Text()), " ")
}
