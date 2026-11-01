package comments

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Place sets InBody and Next on each block, reading each block's file under
// root once. The blocks of one file sit next to each other, as Files
// returns them.
func Place(root string, blocks []Block) error {
	var errs []error
	for i := 0; i < len(blocks); {
		j := i + 1
		for j < len(blocks) && blocks[j].Path == blocks[i].Path {
			j++
		}
		errs = append(errs, placeFile(root, blocks[i:j]))
		i = j
	}
	return errors.Join(errs...)
}

// placeFile places the blocks of one file.
func placeFile(root string, blocks []Block) error {
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(blocks[0].Path)))
	if err != nil {
		return err
	}
	place := placer(blocks[0].Lang, src)
	for i := range blocks {
		place(&blocks[i])
	}
	return nil
}

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
// function body. A language with braces finds its bodies through its code,
// a shell script goes by indent, and YAML has no bodies.
func bodyTest(lang string, src []byte, rows []string) func(b *Block) bool {
	if found, ok := bodies(lang, src, rows); ok {
		return func(b *Block) bool { return found.holds(b.Line) }
	}
	return func(b *Block) bool { return lang == "shell" && isIndented(rows[b.Line-1]) }
}

// bodies returns the function bodies of a file in a language with braces.
// Go reads them through its parser, falling back to its braces when the
// parser rejects the file.
func bodies(lang string, src []byte, rows []string) (spans, bool) {
	if lang == "go" {
		if found, ok := goBodies(src); ok {
			return found, true
		}
		return braceBodies(lexedCode(rows, Lexer{})), true
	}
	if lang == "swift" {
		return braceBodies(swiftCode(src)), true
	}
	lex, ok := lexers[lang]
	return braceBodies(lexedCode(rows, lex)), ok
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
		if body := funcBody(n); body != nil {
			found = append(found, span{fset.Position(body.Lbrace).Line, fset.Position(body.Rbrace).Line})
		}
		return true
	})
	return found, true
}

// funcBody returns the body of a function declaration or literal, or nil.
func funcBody(n ast.Node) *ast.BlockStmt {
	switch fn := n.(type) {
	case *ast.FuncDecl:
		return fn.Body
	case *ast.FuncLit:
		return fn.Body
	}
	return nil
}

// lexedCode returns rows with every comment and literal blanked, as lex
// reads them, joined by newlines.
func lexedCode(rows []string, lex Lexer) string {
	out := make([]string, len(rows))
	for i, row := range rows {
		code := []byte(strings.Repeat(" ", len(row)))
		for _, sp := range lex.Line(row) {
			if sp.Part == Code {
				copy(code[sp.From:sp.To], row[sp.From:sp.To])
			}
		}
		out[i] = string(code)
	}
	return strings.Join(out, "\n")
}

// swiftCode returns Swift source with every comment and string blanked,
// keeping its newlines.
func swiftCode(src []byte) string {
	code := []byte(strings.ReplaceAll(string(src), "\r\n", "\n"))
	blank := func(start, end int) {
		for i := start; i < end; i++ {
			if code[i] != '\n' {
				code[i] = ' '
			}
		}
	}
	WalkSwift(string(code), SwiftVisitor{Comment: blank, String: func(start, end int, _ string) { blank(start, end) }})
	return string(code)
}

// braceBodies returns the function bodies of code whose comments and
// literals are blanked. A brace opens a function body when the header
// before it declares a function and no body holds it yet.
func braceBodies(code string) spans {
	s := &braceScan{line: 1}
	for i := 0; i < len(code); i++ {
		s.char(code, i)
	}
	return s.found
}

// braceScan tracks the open braces, the line, and where the header of the
// next brace starts.
type braceScan struct {
	open  []brace
	found spans
	line  int
	start int
}

// brace is one open brace: its line, whether a function body holds it, and
// whether it opened that body itself.
type brace struct {
	line          int
	inBody, opens bool
}

// char reads the byte at i of code.
func (s *braceScan) char(code string, i int) {
	switch code[i] {
	case '\n':
		s.line++
	case ';':
		s.start = i + 1
	case '{':
		s.push(code[s.start:i])
		s.start = i + 1
	case '}':
		s.pop()
		s.start = i + 1
	}
}

// push opens a brace after header.
func (s *braceScan) push(header string) {
	inBody := len(s.open) > 0 && s.open[len(s.open)-1].inBody
	opens := !inBody && opensFunction(header)
	s.open = append(s.open, brace{line: s.line, inBody: inBody || opens, opens: opens})
}

// pop closes the innermost brace, keeping the body it opened.
func (s *braceScan) pop() {
	if len(s.open) == 0 {
		return
	}
	top := s.open[len(s.open)-1]
	s.open = s.open[:len(s.open)-1]
	if top.opens {
		s.found = append(s.found, span{top.line, s.line})
	}
}

// control opens a header whose brace holds a branch or a loop, not a
// function.
var control = regexp.MustCompile(`^(else|if|for|while|switch|catch|guard|do|try|with|repeat|when|match|loop|finally|synchronized)\b`)

// typeHeader declares a type, whose brace holds members, not statements.
var typeHeader = regexp.MustCompile(`\b(class|struct|enum|interface|object|protocol|extension|impl|trait|record|namespace|module)\b`)

// function ends a header that declares a function: a keyword, an arrow, or
// a closing parenthesis with an optional return type.
var function = regexp.MustCompile(`\b(func|function|fn|fun|init|deinit|constructor|get|set|willSet|didSet)\b|=>$|\)\s*(async\s*)?(throws\s*|rethrows\s*)?((->|:)[^{}]*)?$`)

// opensFunction reports a header, read from its last line of code, that
// declares a function.
func opensFunction(header string) bool {
	h := lastCodeLine(header)
	return !control.MatchString(h) && !typeHeader.MatchString(h) && function.MatchString(h)
}

// lastCodeLine returns the last line of text that holds code, trimmed.
func lastCodeLine(text string) string {
	rows := strings.Split(text, "\n")
	for i := len(rows) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(rows[i]); t != "" {
			return t
		}
	}
	return ""
}

// codeMarkers open a line that holds no code a comment documents: another
// comment, or an attribute such as `@MainActor`, `#[derive]` or `#if`.
var codeMarkers = []string{"//", "/*", "#", "@"}

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
