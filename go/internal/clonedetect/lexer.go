package clonedetect

import (
	"strings"
	"unicode/utf8"
)

// Token is one normalized lexical unit and the 1-based line it starts on.
type Token struct {
	Val  string
	Line int
}

// jscpd's escape-hatch markers; recognized inside any comment.
const (
	ignoreStart = "jscpd:ignore-start"
	ignoreEnd   = "jscpd:ignore-end"
)

// lexer is the scan state for Lex: the input, a byte cursor, the current
// 1-based line, the tokens emitted so far, and whether a
// jscpd:ignore-start region is suppressing emission.
type lexer struct {
	src      string
	i        int
	line     int
	out      []Token
	ignoring bool
}

// Lex tokenizes source generically per the package-comment boundaries:
// comments stripped (and scanned for the ignore markers), strings and
// numbers collapsed to $str/$num, identifiers and punctuation verbatim.
func Lex(src string) []Token {
	lx := &lexer{src: src, line: 1}
	for lx.i < len(src) {
		lx.step()
	}
	return lx.out
}

// step consumes one lexical element at the cursor, dispatching on its
// leading byte to the branch family that owns it; comment openers and
// verbatim runes share the default branch (scanOther).
func (lx *lexer) step() {
	c := lx.src[lx.i]
	switch {
	case isSpace(c):
		lx.skipSpace()
	case isQuote(c):
		lx.scanString()
	case isDigit(c):
		lx.scanNumber()
	case isIdentStart(c):
		lx.scanIdent()
	default:
		lx.scanOther()
	}
}

// emit appends a token starting on the current line unless an ignore
// region is active.
func (lx *lexer) emit(val string) {
	if !lx.ignoring {
		lx.out = append(lx.out, Token{Val: val, Line: lx.line})
	}
}

// skipSpace consumes one whitespace byte, counting the line on a newline.
func (lx *lexer) skipSpace() {
	if lx.src[lx.i] == '\n' {
		lx.line++
	}
	lx.i++
}

// scanString consumes the string literal opening at the cursor, emitting
// one $str token (quote style dropped, content kept) and counting any
// lines a backtick literal spans.
func (lx *lexer) scanString() {
	end := stringEnd(lx.src, lx.i)
	lx.emit("$str:" + stringBody(lx.src, lx.i, end))
	lx.line += strings.Count(lx.src[lx.i:end], "\n")
	lx.i = end
}

// scanNumber consumes a numeric literal (a digit then any run of number
// bytes) and emits the collapsed $num token.
func (lx *lexer) scanNumber() {
	lx.i = scanRun(lx.src, lx.i, isNumPart)
	lx.emit("$num")
}

// scanIdent consumes an identifier and emits it verbatim.
func (lx *lexer) scanIdent() {
	end := scanRun(lx.src, lx.i, isIdentPart)
	lx.emit(lx.src[lx.i:end])
	lx.i = end
}

// scanOther handles bytes no other branch family owns: a comment opener
// consumes the whole comment, any other byte emits its rune verbatim.
func (lx *lexer) scanOther() {
	if end, ok := commentSpan(lx.src, lx.i); ok {
		lx.skipComment(end)
		return
	}
	_, size := utf8.DecodeRuneInString(lx.src[lx.i:])
	lx.emit(lx.src[lx.i : lx.i+size])
	lx.i += size
}

// skipComment consumes the comment ending at end, updating the ignore
// state from its text and the line count from any newlines it spans.
func (lx *lexer) skipComment(end int) {
	text := lx.src[lx.i:end]
	if strings.Contains(text, ignoreStart) {
		lx.ignoring = true
	}
	if strings.Contains(text, ignoreEnd) {
		lx.ignoring = false
	}
	lx.line += strings.Count(text, "\n")
	lx.i = end
}

// commentSpan returns the index just past the comment opening at i, or
// ok=false when no comment starts there. Line comments (// and #) end
// before their newline; block comments include their closer and may span
// lines.
func commentSpan(src string, i int) (end int, ok bool) {
	switch {
	case src[i] == '#' || strings.HasPrefix(src[i:], "//"):
		return lineEnd(src, i), true
	case strings.HasPrefix(src[i:], "/*"):
		return blockEnd(src, i+2, "*/"), true
	case strings.HasPrefix(src[i:], "<!--"):
		return blockEnd(src, i+4, "-->"), true
	}
	return 0, false
}

// scanRun returns the index just past the run of bytes after i that
// satisfy part (the byte at i itself is already accepted).
func scanRun(src string, i int, part func(byte) bool) int {
	end := i + 1
	for end < len(src) && part(src[end]) {
		end++
	}
	return end
}

// CountLines returns the number of source lines in content: a trailing
// newline does not start a new line, and empty content has zero.
func CountLines(content string) int {
	if content == "" {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}

// lineEnd returns the index of the newline ending the line containing i
// (the newline itself is not consumed), or len(src).
func lineEnd(src string, i int) int {
	if idx := strings.IndexByte(src[i:], '\n'); idx >= 0 {
		return i + idx
	}
	return len(src)
}

// blockEnd returns the index just past the closing delimiter for a block
// construct whose body starts at from, or len(src) when unterminated.
func blockEnd(src string, from int, close string) int {
	if idx := strings.Index(src[from:], close); idx >= 0 {
		return from + idx + len(close)
	}
	return len(src)
}

// stringBody extracts the literal's content between the quotes (escape
// sequences stay raw): quote style normalizes away, content does not —
// fully collapsing strings would manufacture clones real jscpd never
// reports (e.g. Go import blocks differing only in import paths).
func stringBody(src string, i, end int) string {
	body := src[i+1 : end]
	if len(body) > 0 && body[len(body)-1] == src[i] {
		body = body[:len(body)-1]
	}
	return body
}

// stringEnd returns the index just past the string literal opening at i.
// Backslash escapes the next byte; single- and double-quoted strings close
// implicitly at end of line (the newline is left for the caller), backtick
// strings span lines; unterminated strings run to len(src).
func stringEnd(src string, i int) int {
	quote := src[i]
	for j := i + 1; j < len(src); j++ {
		if src[j] == '\\' {
			j++
			continue
		}
		if end, ok := stringClose(quote, src[j], j); ok {
			return end
		}
	}
	return len(src)
}

// stringClose reports whether byte c at index j terminates a string opened
// with quote, returning the index just past the literal — for the implicit
// newline close of non-backtick strings, the newline stays unconsumed.
func stringClose(quote, c byte, j int) (int, bool) {
	if c == quote {
		return j + 1, true
	}
	if c == '\n' && quote != '`' {
		return j, true
	}
	return 0, false
}

// isSpace reports whether c is whitespace the lexer discards, newline
// included (skipSpace tracks the line count).
func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// isQuote reports whether c opens a string literal.
func isQuote(c byte) bool {
	return c == '"' || c == '\'' || c == '`'
}

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isLetter reports whether c is an ASCII letter.
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || isLetter(c)
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

// isNumPart accepts the tail of a numeric literal: digits, letters (hex,
// exponents, suffixes), underscore separators, and the decimal point.
func isNumPart(c byte) bool {
	return isIdentPart(c) || c == '.'
}
