// cspell:ignore fileprivate

package comments

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// placer returns the function that sets a block's InBody and Next from the
// source the block was read from. An inline comment sits after code, so it
// is always in a body.
func placer(lang string, src []byte) func(b *Block) {
	rows := lines(src)
	inBody := bodyTest(lang, src, rows)
	return func(b *Block) {
		if b.Kind == Inline {
			b.InBody = true
			return
		}
		b.Next = nextCode(rows, b.End)
		b.InBody = inBody(b)
	}
}

// bodyTest returns the test for a comment on lines of its own inside a
// function body. Go reads its function bodies from the parser, and YAML has
// none. Other languages judge by the comment's indent, marker and next line.
func bodyTest(lang string, src []byte, rows []string) func(b *Block) bool {
	switch lang {
	case "yaml":
		return func(*Block) bool { return false }
	case "go":
		if spans, ok := goBodies(src); ok {
			return func(b *Block) bool { return spans.holds(b.Line) }
		}
	}
	return func(b *Block) bool { return indentedStatement(rows, b) }
}

// span is the lines of one function body's braces, counted from 1.
type span struct{ from, to int }

// spans are the function bodies of one file.
type spans []span

// holds reports a line between the braces of some function body.
func (s spans) holds(line int) bool {
	for _, sp := range s {
		if sp.from < line && line < sp.to {
			return true
		}
	}
	return false
}

// goBodies returns the bodies of a Go file's functions and function
// literals, or false when the parser rejects the file.
func goBodies(src []byte) (spans, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, false
	}
	var found spans
	ast.Inspect(file, func(n ast.Node) bool {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		}
		if body != nil {
			found = append(found, span{fset.Position(body.Lbrace).Line, fset.Position(body.Rbrace).Line})
		}
		return true
	})
	return found, true
}

// docMarker opens a comment that its language reads as documentation, such
// as Swift's or Rust's `///` and a JSDoc or KDoc `/**`.
var docMarker = regexp.MustCompile(`^(///|/\*\*|//!|/\*!)`)

// declaration opens a line of code that declares a function, a type or a
// member. It starts with a modifier, a keyword, or a shell `name() {`.
var declaration = regexp.MustCompile(`^(?:(?:public|private|fileprivate|internal|open|protected|static|final|override|abstract|export|default|async|declare|readonly|mutating|convenience|required|lazy|weak|indirect|sealed|data|inline|suspend|unsafe|extern|pub(?:\([^)]*\))?)\s+|(?:func|function|class|struct|enum|protocol|extension|interface|type|typealias|init|deinit|subscript|fn|impl|trait|mod|fun|object|namespace|actor)\b|[\w-]+\s*\(\)\s*\{)`)

// indentedStatement reports an indented comment, written without a doc
// marker, whose next line of code declares nothing. A comment like that
// sits among a body's statements.
func indentedStatement(rows []string, b *Block) bool {
	row := rows[b.Line-1]
	if !isIndented(row) || docMarker.MatchString(strings.TrimSpace(row)) {
		return false
	}
	return !declaration.MatchString(b.Next)
}

// codeMarkers open a line that holds no code a comment documents: another
// comment, or an attribute such as `@MainActor` or `#[derive]`.
var codeMarkers = []string{"//", "/*", "*", "#", "@"}

// nextCode returns the first line of code after line end, trimmed, or ""
// when the file ends first.
func nextCode(rows []string, end int) string {
	for _, row := range rows[min(end, len(rows)):] {
		t := strings.TrimSpace(row)
		if t != "" && !hasAnyPrefix(t, codeMarkers) {
			return t
		}
	}
	return ""
}

// hasAnyPrefix reports s starting with one of prefixes.
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
