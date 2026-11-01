package comments

import (
	"go/parser"
	"go/token"
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
	var notes []note
	for _, group := range file.Comments {
		for _, c := range group.List {
			start := fset.Position(c.Pos()).Offset
			notes = append(notes, offs.note(start, start+goCommentLen(text[start:], c.Text)))
		}
	}
	return notes
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
