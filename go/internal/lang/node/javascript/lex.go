// Package javascript tokenizes JavaScript and TypeScript source without running
// it, so checks can read config literals and test calls.
package javascript

import (
	"strconv"
	"strings"
)

// Kind classifies a lexed token.
type Kind int

// The token kinds: punctuation, identifier, string or template literal,
// number, and regex literal.
const (
	Punct Kind = iota
	Ident
	String
	Number
	Regex
)

// Token is one lexed unit of JS/TS source, and comments never become tokens.
// For strings, Text is the decoded value and Lit is false when a template held
// a ${…} expression.
type Token struct {
	Kind Kind
	Text string
	Lit  bool
	Pos  int
}

// regexKeywords are identifiers after which a slash starts a regex literal,
// not division.
var regexKeywords = map[string]bool{
	"await": true, "case": true, "delete": true, "do": true, "else": true,
	"in": true, "instanceof": true, "new": true, "of": true, "return": true,
	"throw": true, "typeof": true, "void": true, "yield": true,
}

// Lex tokenizes JS/TS source, dropping whitespace and comments while
// keeping string contents intact.
func Lex(src string) []Token {
	return lex(src, false)
}

// LexJSX tokenizes TSX or JSX source as Lex does, but a slash after a < closes
// a tag and never starts a regex.
func LexJSX(src string) []Token {
	return lex(src, true)
}

func lex(src string, jsx bool) []Token {
	var tokens []Token
	i := 0
	for i < len(src) {
		if isSpace(src[i]) {
			i++
			continue
		}
		start, n := i, len(tokens)
		tokens, i = lexAt(src, i, tokens, jsx)
		if len(tokens) > n {
			tokens[n].Pos = start
		}
	}
	return tokens
}

// isSpace reports whether c is whitespace the lexer drops.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

// lexAt lexes the comment or token at the non-whitespace byte i, returning the
// extended tokens and the next index.
func lexAt(src string, i int, tokens []Token, jsx bool) ([]Token, int) {
	switch c := src[i]; {
	case c == '/':
		return lexSlash(src, i, tokens, jsx)
	case c == '\'' || c == '"':
		value, next := scanQuoted(src, i)
		return append(tokens, Token{Kind: String, Text: value, Lit: true}), next
	case c == '`':
		value, hasExpr, next := scanTemplate(src, i)
		return append(tokens, Token{Kind: String, Text: value, Lit: !hasExpr}), next
	default:
		return lexWord(src, i, tokens)
	}
}

// lexSlash lexes a slash as a comment, a regex literal, or a punctuation token.
func lexSlash(src string, i int, tokens []Token, jsx bool) ([]Token, int) {
	switch {
	case i+1 < len(src) && (src[i+1] == '/' || src[i+1] == '*'):
		return tokens, skipComment(src, i)
	case regexStarts(i, tokens, jsx):
		return append(tokens, Token{Kind: Regex}), skipRegex(src, i)
	default:
		return append(tokens, Token{Kind: Punct, Text: src[i : i+1]}), i + 1
	}
}

// regexStarts reports whether the slash at i starts a regex rather than
// closing a JSX tag.
func regexStarts(i int, tokens []Token, jsx bool) bool {
	return regexCanFollow(tokens) && !(jsx && closingTag(i, tokens))
}

// closingTag reports the slash at i written right after a < token, as a
// JSX closing tag such as </b> starts.
func closingTag(i int, tokens []Token) bool {
	if len(tokens) == 0 {
		return false
	}
	last := tokens[len(tokens)-1]
	return last.Kind == Punct && last.Text == "<" && last.Pos == i-1
}

// skipComment advances past the comment starting at i (which points at the
// slash; src[i+1] selects line or block). An unterminated block comment
// swallows the rest of the input.
func skipComment(src string, i int) int {
	if src[i+1] == '*' {
		end := strings.Index(src[i+2:], "*/")
		if end < 0 {
			return len(src)
		}
		return i + 2 + end + 2
	}
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

// lexWord lexes an identifier, number, or single-character punctuation token.
func lexWord(src string, i int, tokens []Token) ([]Token, int) {
	c := src[i]
	switch {
	case isIdentStart(c):
		end := scanIdent(src, i)
		return append(tokens, Token{Kind: Ident, Text: src[i:end]}), end
	case c >= '0' && c <= '9':
		return append(tokens, Token{Kind: Number}), scanNumber(src, i)
	default:
		return append(tokens, Token{Kind: Punct, Text: src[i : i+1]}), i + 1
	}
}

// scanIdent advances past the identifier characters at i, returning the
// index just past the last one.
func scanIdent(src string, i int) int {
	for i < len(src) && isIdentPart(src[i]) {
		i++
	}
	return i
}

// scanNumber advances past a numeric literal: the same loose sweep of ident
// characters and dots the lexer always used.
func scanNumber(src string, i int) int {
	for i < len(src) && (isIdentPart(src[i]) || src[i] == '.') {
		i++
	}
	return i
}

// regexCanFollow reports whether a slash starts a regex: yes at the input start
// and after operators, no after a value.
func regexCanFollow(tokens []Token) bool {
	if len(tokens) == 0 {
		return true
	}
	last := tokens[len(tokens)-1]
	switch last.Kind {
	case Ident:
		return regexKeywords[last.Text]
	case String, Number, Regex:
		return false
	}
	return !closesValue(last.Text)
}

// closesValue reports whether punctuation text ends a value expression — a
// closing paren, bracket, or brace, after which a slash means division.
func closesValue(text string) bool {
	return text == ")" || text == "]" || text == "}"
}

// skipRegex advances past a regex literal (body, character classes, flags).
func skipRegex(src string, i int) int {
	end, closed := skipRegexBody(src, i+1)
	if closed {
		return scanIdent(src, end+1) // skip the slash, then the flags
	}
	return end
}

// skipRegexBody advances past a regex body starting just after the opening
// slash. closed reports whether end is the closing slash; otherwise end is
// the newline of a malformed regex, or the end of input.
func skipRegexBody(src string, i int) (end int, closed bool) {
	for i < len(src) {
		switch src[i] {
		case '\\':
			i += 2
		case '[':
			i = skipRegexClass(src, i+1)
		case '/', '\n':
			return i, src[i] == '/'
		default:
			i++
		}
	}
	return i, false
}

// skipRegexClass advances past a regex character class that starts after
// its bracket, honoring escapes.
func skipRegexClass(src string, i int) int {
	for i < len(src) {
		switch src[i] {
		case '\\':
			i += 2
		case ']':
			return i + 1
		default:
			i++
		}
	}
	return i
}

// scanQuoted decodes a single- or double-quoted string starting at i,
// returning its value and the index past the closing quote. An unterminated
// string ends at the newline (a JS syntax error; tolerated here).
func scanQuoted(src string, i int) (string, int) {
	quote := src[i]
	i++
	var b strings.Builder
	for i < len(src) {
		c := src[i]
		switch {
		case c == quote:
			return b.String(), i + 1
		case c == '\\':
			s, next := decodeEscape(src, i)
			b.WriteString(s)
			i = next
		case c == '\n':
			return b.String(), i
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), i
}

// scanTemplate decodes a template literal starting at i. hasExpr is true
// when the template held a ${…} expression, making its value unresolvable.
func scanTemplate(src string, i int) (value string, hasExpr bool, next int) {
	i++ // skip the opening backtick
	var b strings.Builder
	for i < len(src) {
		c := src[i]
		switch {
		case c == '`':
			return b.String(), hasExpr, i + 1
		case c == '\\':
			s, n := decodeEscape(src, i)
			b.WriteString(s)
			i = n
		case startsTemplateExpr(src, i):
			hasExpr = true
			i = skipBraced(src, i+1)
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), hasExpr, i
}

// startsTemplateExpr reports whether the ${ opener of a template expression
// sits at i.
func startsTemplateExpr(src string, i int) bool {
	return src[i] == '$' && i+1 < len(src) && src[i+1] == '{'
}

// skipBraced advances past a balanced-brace span starting at an opening
// brace, skipping nested strings and templates so their braces don't count.
func skipBraced(src string, i int) int {
	depth := 0
	for i < len(src) {
		switch src[i] {
		case '{':
			depth++
			i++
		case '}':
			depth--
			i++
			if depth == 0 {
				return i
			}
		default:
			i = skipInert(src, i)
		}
	}
	return i
}

// skipInert advances past content that cannot change brace depth: a quoted
// string or template literal, a backslash escape, or one ordinary character.
func skipInert(src string, i int) int {
	switch src[i] {
	case '\'', '"':
		_, next := scanQuoted(src, i)
		return next
	case '`':
		_, _, next := scanTemplate(src, i)
		return next
	case '\\':
		return i + 2
	default:
		return i + 1
	}
}

// namedEscapes maps single-character escape names to their decoded values,
// including the backslash–LF line continuation, which decodes to nothing.
var namedEscapes = map[byte]string{
	'n': "\n", 't': "\t", 'r': "\r", 'b': "\b",
	'f': "\f", 'v': "\v", '0': "\x00", '\n': "",
}

// decodeEscape decodes the escape sequence at the backslash i with the
// semantics that JavaScript gives it.
func decodeEscape(src string, i int) (string, int) {
	if i+1 >= len(src) {
		return "", i + 1
	}
	if s, ok := namedEscapes[src[i+1]]; ok {
		return s, i + 2
	}
	if src[i+1] == '\r' {
		return "", skipEscapedCRLF(src, i+2)
	}
	if s, next, ok := decodeHexEscape(src, i); ok {
		return s, next
	}
	return src[i+1 : i+2], i + 2
}

// skipEscapedCRLF returns the index past a backslash–CR line continuation
// whose CR ends just before i, consuming a following LF when present.
func skipEscapedCRLF(src string, i int) int {
	if i < len(src) && src[i] == '\n' {
		return i + 1
	}
	return i
}

// decodeHexEscape decodes the \xHH, \uHHHH, or \u{…} escape at i (pointing
// at the backslash). ok is false for any other escape name or for malformed
// digits — the caller then treats the escaped character as itself.
func decodeHexEscape(src string, i int) (string, int, bool) {
	switch src[i+1] {
	case 'x':
		if r, ok := hexRune(src, i+2, 2); ok {
			return string(r), i + 4, true
		}
	case 'u':
		return decodeUnicodeEscape(src, i)
	}
	return "", 0, false
}

// decodeUnicodeEscape decodes the \uHHHH or \u{…} escape at i (pointing at
// the backslash); ok is false when the digits are malformed.
func decodeUnicodeEscape(src string, i int) (string, int, bool) {
	if i+2 < len(src) && src[i+2] == '{' {
		return decodeBracedUnicode(src, i)
	}
	if r, ok := hexRune(src, i+2, 4); ok {
		return string(r), i + 6, true
	}
	return "", 0, false
}

// decodeBracedUnicode decodes the \u{…} escape at the backslash i, with ok false
// when the braces are unclosed, empty or malformed.
func decodeBracedUnicode(src string, i int) (string, int, bool) {
	end := strings.IndexByte(src[i+3:], '}')
	if end <= 0 {
		return "", 0, false
	}
	r, ok := hexRune(src, i+3, end)
	if !ok {
		return "", 0, false
	}
	return string(r), i + 3 + end + 1, true
}

// hexRune parses n hex digits at i as a rune.
func hexRune(src string, i, n int) (rune, bool) {
	if i+n > len(src) {
		return 0, false
	}
	v, err := strconv.ParseUint(src[i:i+n], 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(v), true
}

// isASCIILetter reports whether c is an ASCII letter.
func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdentStart reports whether c can start an identifier; any byte >= 0x80
// (part of a multibyte rune) is accepted.
func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || isASCIILetter(c) || c >= 0x80
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
