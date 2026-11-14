package views

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// kind is what a token is: punctuation, a name, or a literal.
type kind int

const (
	punct kind = iota
	ident
	literal
	// skipped is white space or a comment, which lex drops.
	skipped
)

// token is one token of a Java source: its kind, its text, and the byte
// offset it starts at. A literal keeps its text, but no name reads it.
type token struct {
	kind kind
	text string
	pos  int
}

// lex splits a Java source into tokens per JLS chapter 3, dropping white space
// and comments and keeping literals whole.
func lex(src string) []token {
	var out []token
	for i := 0; i < len(src); {
		next, k := scan(src, i)
		if k != skipped {
			out = append(out, token{kind: k, text: src[i:next], pos: i})
		}
		i = next
	}
	return out
}

// rule reads one kind of token, where match tests its first character and read
// returns the offset after it.
type rule struct {
	match func(rest string, r rune) bool
	read  func(src string, i, size int) (int, kind)
}

// rules are tried in order; a character no rule matches is punctuation.
var rules = []rule{
	{func(_ string, r rune) bool { return unicode.IsSpace(r) }, func(_ string, i, size int) (int, kind) { return i + size, skipped }},
	{prefix("//"), func(src string, i, _ int) (int, kind) { return through(src, i, "\n"), skipped }},
	{prefix("/*"), func(src string, i, _ int) (int, kind) { return through(src, i+2, "*/"), skipped }},
	{prefix(`"""`), func(src string, i, _ int) (int, kind) { return quoted(src, i+3, `"""`), literal }},
	{func(_ string, r rune) bool { return r == '"' || r == '\'' }, func(src string, i, _ int) (int, kind) { return quoted(src, i+1, src[i:i+1]), literal }},
	{numberStart, func(src string, i, _ int) (int, kind) { return number(src, i), literal }},
	{func(_ string, r rune) bool { return nameStart(r) }, func(src string, i, size int) (int, kind) { return name(src, i+size), ident }},
}

// scan reads the token or gap at offset i and returns the offset after
// it, with its kind.
func scan(src string, i int) (int, kind) {
	r, size := utf8.DecodeRuneInString(src[i:])
	for _, rl := range rules {
		if rl.match(src[i:], r) {
			return rl.read(src, i, size)
		}
	}
	return i + size, punct
}

func prefix(p string) func(rest string, _ rune) bool {
	return func(rest string, _ rune) bool { return strings.HasPrefix(rest, p) }
}

// through returns the offset after the first close at or past offset i,
// or the source's end.
func through(src string, i int, close string) int {
	if end := strings.Index(src[i:], close); end >= 0 {
		return i + end + len(close)
	}
	return len(src)
}

// quoted returns the offset after the close of a literal whose body
// starts at offset i. A backslash escapes the next character, and a
// one-line literal ends at its line's end even when left open.
func quoted(src string, i int, close string) int {
	for i < len(src) && !strings.HasPrefix(src[i:], close) {
		if src[i] == '\n' && len(close) == 1 {
			return i
		}
		i += escape(src[i])
	}
	return min(i+len(close), len(src))
}

// escape is how many bytes a literal's character at c takes: two for a
// backslash and the character it escapes.
func escape(c byte) int {
	if c == '\\' {
		return 2
	}
	return 1
}

// numberStart reports a digit, or a dot before one.
func numberStart(rest string, r rune) bool {
	return unicode.IsDigit(r) || (r == '.' && len(rest) > 1 && unicode.IsDigit(rune(rest[1])))
}

// number returns the offset after a number literal at offset i, a hex literal's
// exponent letter being p because e is a digit.
func number(src string, i int) int {
	exponents := "eEpP"
	if strings.EqualFold(src[i:min(i+2, len(src))], "0x") {
		exponents = "pP"
	}
	j := i + 1
	for j < len(src) && numberByte(src, j, exponents) {
		j++
	}
	return j
}

func numberByte(src string, j int, exponents string) bool {
	return src[j] == '.' || nameByte(src[j]) || exponentSign(src, j, exponents)
}

func exponentSign(src string, j int, exponents string) bool {
	return strings.ContainsRune("+-", rune(src[j])) && strings.ContainsRune(exponents, rune(src[j-1]))
}

// name returns the offset after a name whose first character ends before
// offset i.
func name(src string, i int) int {
	for i < len(src) {
		r, size := utf8.DecodeRuneInString(src[i:])
		if !nameStart(r) && !unicode.IsDigit(r) {
			return i
		}
		i += size
	}
	return i
}

// nameStart reports a character that can start a Java name.
func nameStart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r)
}

func nameByte(c byte) bool {
	return c >= utf8.RuneSelf || nameStart(rune(c)) || unicode.IsDigit(rune(c))
}
