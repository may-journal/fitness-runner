package vitestconf

import (
	"strconv"
	"strings"
)

// tokKind classifies a lexed token.
type tokKind int

const (
	tokPunct tokKind = iota
	tokIdent
	tokString
	tokNumber
	tokRegex
)

// token is one lexed unit of JS/TS source. For strings, text is the decoded
// value and lit is false when a template literal held a ${…} expression —
// present but unresolvable. Comments never become tokens.
type token struct {
	kind tokKind
	text string
	lit  bool
}

// regexKeywords are identifiers after which a slash starts a regex literal,
// not division.
var regexKeywords = map[string]bool{
	"await": true, "case": true, "delete": true, "do": true, "else": true,
	"in": true, "instanceof": true, "new": true, "of": true, "return": true,
	"throw": true, "typeof": true, "void": true, "yield": true,
}

// lex tokenizes JS/TS source, dropping whitespace and comments while keeping
// string contents intact — the string-aware comment stripping the TS
// evaluation got for free from the JavaScript engine.
func lex(src string) []token {
	var tokens []token
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				i = len(src)
			} else {
				i += 2 + end + 2
			}
		case c == '/' && regexCanFollow(tokens):
			i = skipRegex(src, i)
			tokens = append(tokens, token{kind: tokRegex})
		case c == '\'' || c == '"':
			value, next := scanQuoted(src, i)
			tokens = append(tokens, token{kind: tokString, text: value, lit: true})
			i = next
		case c == '`':
			value, hasExpr, next := scanTemplate(src, i)
			tokens = append(tokens, token{kind: tokString, text: value, lit: !hasExpr})
			i = next
		case isIdentStart(c):
			start := i
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokIdent, text: src[start:i]})
		case c >= '0' && c <= '9':
			for i < len(src) && (isIdentPart(src[i]) || src[i] == '.') {
				i++
			}
			tokens = append(tokens, token{kind: tokNumber})
		default:
			tokens = append(tokens, token{kind: tokPunct, text: src[i : i+1]})
			i++
		}
	}
	return tokens
}

// regexCanFollow reports whether a slash at the current position starts a
// regex literal: yes at the start of input and after operators or opening
// punctuation; no after a value (identifier, string, number, regex, or
// closing punctuation) — the standard division heuristic.
func regexCanFollow(tokens []token) bool {
	if len(tokens) == 0 {
		return true
	}
	last := tokens[len(tokens)-1]
	switch last.kind {
	case tokIdent:
		return regexKeywords[last.text]
	case tokString, tokNumber, tokRegex:
		return false
	}
	return last.text != ")" && last.text != "]" && last.text != "}"
}

// skipRegex advances past a regex literal (body, character classes, flags).
func skipRegex(src string, i int) int {
	i++ // opening slash
	inClass := false
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\\':
			i += 2
		case inClass:
			if c == ']' {
				inClass = false
			}
			i++
		case c == '[':
			inClass = true
			i++
		case c == '/':
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			return i
		case c == '\n':
			return i // malformed regex; bail at end of line
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
	i++ // opening backtick
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
		case c == '$' && i+1 < len(src) && src[i+1] == '{':
			hasExpr = true
			i = skipBraced(src, i+1)
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), hasExpr, i
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
		case '\'', '"':
			_, i = scanQuoted(src, i)
		case '`':
			_, _, i = scanTemplate(src, i)
		case '\\':
			i += 2
		default:
			i++
		}
	}
	return i
}

// decodeEscape decodes the escape sequence at i (which points at the
// backslash) with JavaScript semantics: named escapes, \xHH, \uHHHH,
// \u{…}, line continuations; any other escaped character is itself.
func decodeEscape(src string, i int) (string, int) {
	if i+1 >= len(src) {
		return "", i + 1
	}
	c := src[i+1]
	switch c {
	case 'n':
		return "\n", i + 2
	case 't':
		return "\t", i + 2
	case 'r':
		return "\r", i + 2
	case 'b':
		return "\b", i + 2
	case 'f':
		return "\f", i + 2
	case 'v':
		return "\v", i + 2
	case '0':
		return "\x00", i + 2
	case '\n':
		return "", i + 2
	case '\r':
		if i+2 < len(src) && src[i+2] == '\n' {
			return "", i + 3
		}
		return "", i + 2
	case 'x':
		if r, ok := hexRune(src, i+2, 2); ok {
			return string(r), i + 4
		}
	case 'u':
		if i+2 < len(src) && src[i+2] == '{' {
			if end := strings.IndexByte(src[i+3:], '}'); end > 0 {
				if r, ok := hexRune(src, i+3, end); ok {
					return string(r), i + 3 + end + 1
				}
			}
		} else if r, ok := hexRune(src, i+2, 4); ok {
			return string(r), i + 6
		}
	}
	return src[i+1 : i+2], i + 2
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

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
