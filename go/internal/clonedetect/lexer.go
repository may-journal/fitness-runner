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

// Lex tokenizes source generically per the package-comment boundaries:
// comments stripped (and scanned for the ignore markers), strings and
// numbers collapsed to $str/$num, identifiers and punctuation verbatim.
func Lex(src string) []Token {
	var out []Token
	ignoring := false
	line := 1
	emit := func(val string, at int) {
		if !ignoring {
			out = append(out, Token{Val: val, Line: at})
		}
	}
	comment := func(text string) {
		if strings.Contains(text, ignoreStart) {
			ignoring = true
		}
		if strings.Contains(text, ignoreEnd) {
			ignoring = false
		}
	}
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f':
			i++
		case c == '/' && strings.HasPrefix(src[i:], "//"):
			end := lineEnd(src, i)
			comment(src[i:end])
			i = end
		case c == '/' && strings.HasPrefix(src[i:], "/*"):
			end := blockEnd(src, i+2, "*/")
			comment(src[i:end])
			line += strings.Count(src[i:end], "\n")
			i = end
		case c == '#':
			end := lineEnd(src, i)
			comment(src[i:end])
			i = end
		case c == '<' && strings.HasPrefix(src[i:], "<!--"):
			end := blockEnd(src, i+4, "-->")
			comment(src[i:end])
			line += strings.Count(src[i:end], "\n")
			i = end
		case c == '"' || c == '\'' || c == '`':
			end := stringEnd(src, i)
			emit("$str:"+stringBody(src, i, end), line)
			line += strings.Count(src[i:end], "\n")
			i = end
		case c >= '0' && c <= '9':
			end := i + 1
			for end < len(src) && isNumPart(src[end]) {
				end++
			}
			emit("$num", line)
			i = end
		case isIdentStart(c):
			end := i + 1
			for end < len(src) && isIdentPart(src[end]) {
				end++
			}
			emit(src[i:end], line)
			i = end
		default:
			_, size := utf8.DecodeRuneInString(src[i:])
			emit(src[i:i+size], line)
			i += size
		}
	}
	return out
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
		switch src[j] {
		case '\\':
			j++
		case '\n':
			if quote != '`' {
				return j
			}
		case quote:
			return j + 1
		}
	}
	return len(src)
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// isNumPart accepts the tail of a numeric literal: digits, letters (hex,
// exponents, suffixes), underscore separators, and the decimal point.
func isNumPart(c byte) bool {
	return isIdentPart(c) || c == '.'
}
