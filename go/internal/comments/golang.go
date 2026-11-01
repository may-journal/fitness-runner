package comments

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// goNotes reads the comments of Go source through go/parser. A file the
// parser rejects, such as a template, is read by the C-style lexer.
func goNotes(src []byte) []note {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return cNotes(lines(src), Lexer{})
	}
	text := string(src)
	offs := newOffsets(text)
	kept := goCodeGroups(file)
	var notes []note
	for _, group := range file.Comments {
		for _, c := range group.List {
			start := fset.Position(c.Pos()).Offset
			n := offs.note(start, start+goCommentLen(text[start:], c.Text))
			n.code = kept[group]
			notes = append(notes, n)
		}
	}
	return notes
}

// exampleOutput opens the comment that states what a Go example prints.
var exampleOutput = regexp.MustCompile(`(?im)^\s*(unordered )?output:`)

// goCodeGroups lists the comment groups the Go toolchain reads as code. One
// is a cgo preamble above `import "C"`. Another is an example's output
// comment, which `go test` checks.
func goCodeGroups(file *ast.File) map[*ast.CommentGroup]bool {
	kept := map[*ast.CommentGroup]bool{}
	for _, group := range file.Comments {
		kept[group] = exampleOutput.MatchString(group.Text())
	}
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			keepPreamble(kept, gen)
		}
	}
	return kept
}

// keepPreamble marks the comments above an `import "C"` as code.
func keepPreamble(kept map[*ast.CommentGroup]bool, gen *ast.GenDecl) {
	for _, s := range gen.Specs {
		if spec := s.(*ast.ImportSpec); spec.Path.Value == `"C"` {
			kept[gen.Doc], kept[spec.Doc] = true, true
		}
	}
}

// goCommentLen is the length in the source of a comment the parser holds
// as text. The parser drops carriage returns, so a block comment's length
// is read up to its */ instead.
func goCommentLen(rest, text string) int {
	if end := strings.Index(rest, "*/"); strings.HasPrefix(text, "/*") && end >= 0 {
		return end + 2
	}
	return len(text)
}
