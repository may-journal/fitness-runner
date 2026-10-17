package views

import (
	"strings"
)

// Swift token kinds.
const (
	swiftName   = 'i'
	swiftString = 's'
	swiftNumber = 'n'
	swiftPunct  = 'p'
	swiftQuoted = 'q'
)

// swiftToken is one Swift token: a name, a literal, or punctuation, with
// its line and the #if conditions it sits under. Keywords are names; the
// parser tells them apart.
type swiftToken struct {
	kind  byte
	text  string
	line  int
	under string
}

// swiftLexer splits Swift source into tokens. Comments, which may nest,
// make none. A string literal makes one token after the tokens of its
// interpolations, so a name inside \( ) is still read. Groups holds the
// token index each branch of the open #if groups starts at.
type swiftLexer struct {
	src    string
	i      int
	line   int
	cond   conditions
	out    []swiftToken
	groups [][]int
}

// swiftRule lexes one kind of text where it matches.
type swiftRule struct {
	match func(l *swiftLexer) bool
	lex   func(l *swiftLexer)
}

var swiftRules []swiftRule

func init() {
	swiftRules = []swiftRule{
		{func(l *swiftLexer) bool { return l.at("\n") }, func(l *swiftLexer) { l.line++; l.i++ }},
		{func(l *swiftLexer) bool { return strings.IndexByte(" \t\r", l.src[l.i]) >= 0 }, func(l *swiftLexer) { l.i++ }},
		{func(l *swiftLexer) bool { return l.at("//") }, (*swiftLexer).lineComment},
		{func(l *swiftLexer) bool { return l.at("/*") }, (*swiftLexer).blockComment},
		{(*swiftLexer).atDirective, (*swiftLexer).directive},
		{(*swiftLexer).atString, (*swiftLexer).str},
		{func(l *swiftLexer) bool { return l.at("`") }, (*swiftLexer).quotedName},
		{func(l *swiftLexer) bool { return nameByte(l.src[l.i]) && !digit(l.src[l.i]) }, (*swiftLexer).name},
		{func(l *swiftLexer) bool { return digit(l.src[l.i]) }, (*swiftLexer).number},
	}
}

// lexSwift tokenizes one Swift file.
func lexSwift(src string) []swiftToken {
	l := &swiftLexer{src: src, line: 1}
	l.code(false)
	return l.out
}

func (l *swiftLexer) at(prefix string) bool { return strings.HasPrefix(l.src[l.i:], prefix) }

// code lexes to the end of the source or, inside an interpolation, past
// the parenthesis closing it.
func (l *swiftLexer) code(interpolation bool) {
	depth := 0
	for l.i < len(l.src) {
		if interpolation && l.closes(&depth) {
			return
		}
		l.step()
	}
}

func (l *swiftLexer) closes(depth *int) bool {
	switch l.src[l.i] {
	case '(':
		*depth++
	case ')':
		if *depth == 0 {
			l.i++
			return true
		}
		*depth--
	}
	return false
}

func (l *swiftLexer) step() {
	for _, r := range swiftRules {
		if r.match(l) {
			r.lex(l)
			return
		}
	}
	l.punct()
}

func (l *swiftLexer) emit(kind byte, text string, line int) {
	l.out = append(l.out, swiftToken{kind: kind, text: text, line: line, under: l.cond.String()})
}

func (l *swiftLexer) lineComment() {
	if end := strings.IndexByte(l.src[l.i:], '\n'); end >= 0 {
		l.i += end
		return
	}
	l.i = len(l.src)
}

// blockComment skips a comment, counting the comments nested inside it.
func (l *swiftLexer) blockComment() {
	depth := 0
	for l.i < len(l.src) {
		depth += l.commentDelta()
		if depth == 0 {
			return
		}
	}
}

// commentDelta moves past one character, or a comment's opening or closing
// pair, reporting how the nesting changes.
func (l *swiftLexer) commentDelta() int {
	switch {
	case l.at("/*"):
		l.i += 2
		return 1
	case l.at("*/"):
		l.i += 2
		return -1
	case l.at("\n"):
		l.line++
	}
	l.i++
	return 0
}

// atDirective reports a conditional compilation directive.
func (l *swiftLexer) atDirective() bool {
	for _, d := range []string{"#if", "#elseif", "#else", "#endif"} {
		if l.at(d) && !l.nameAt(l.i+len(d)) {
			return true
		}
	}
	return false
}

func (l *swiftLexer) nameAt(i int) bool { return i < len(l.src) && nameByte(l.src[i]) }

// directive reads a directive line, without its comment, into the
// conditions the following tokens sit under.
func (l *swiftLexer) directive() {
	start := l.i
	l.lineComment()
	text, _, _ := strings.Cut(l.src[start:l.i], "//")
	text, _, _ = strings.Cut(text, "/*")
	l.cond.directive(text)
	l.branch(strings.TrimSpace(text))
}

// branch tracks where each branch of an #if group starts.
func (l *swiftLexer) branch(text string) {
	last := len(l.groups) - 1
	switch {
	case strings.HasPrefix(text, "#if"):
		l.groups = append(l.groups, []int{len(l.out)})
	case strings.HasPrefix(text, "#endif"):
		l.endGroup()
	case last >= 0:
		l.groups[last] = append(l.groups[last], len(l.out))
	}
}

// endGroup closes an #if group. When a branch leaves a brace open, as
// alternative declaration headers do, only the first branch is kept, so
// the braces still pair up as the compiler sees them.
func (l *swiftLexer) endGroup() {
	if len(l.groups) == 0 {
		return
	}
	starts := l.groups[len(l.groups)-1]
	l.groups = l.groups[:len(l.groups)-1]
	ends := append(append([]int{}, starts[1:]...), len(l.out))
	if !l.balanced(starts, ends) {
		l.out = l.out[:ends[0]]
	}
}

func (l *swiftLexer) balanced(starts, ends []int) bool {
	for i := range starts {
		if l.braces(starts[i], ends[i]) != 0 {
			return false
		}
	}
	return true
}

// braces counts the braces a token range opens, less those it closes.
func (l *swiftLexer) braces(from, to int) int {
	n := 0
	for _, t := range l.out[from:to] {
		if t.kind == swiftPunct {
			n += swiftBrace[t.text]
		}
	}
	return n
}

var swiftBrace = map[string]int{"{": 1, "}": -1}

// atString reports a string literal, raw ones included: #"..."#.
func (l *swiftLexer) atString() bool {
	return l.src[l.i+l.hashes()] == '"'
}

func (l *swiftLexer) hashes() int {
	n := 0
	for l.i+n < len(l.src)-1 && l.src[l.i+n] == '#' {
		n++
	}
	return n
}

// str lexes a string literal and the code of its interpolations.
func (l *swiftLexer) str() {
	line, pounds := l.line, strings.Repeat("#", l.hashes())
	l.i += len(pounds)
	quote := `"`
	if l.at(`"""`) {
		quote = `"""`
	}
	l.i += len(quote)
	l.literal(quote+pounds, `\`+pounds)
	l.emit(swiftString, "", line)
}

// literal moves past the literal's closing delimiter. A one-line literal
// left open ends at its line.
func (l *swiftLexer) literal(close, escape string) {
	for l.i < len(l.src) && !l.at(close) && !(len(close) < 3 && l.at("\n")) {
		l.literalStep(escape)
	}
	l.i = min(l.i+len(close), len(l.src))
}

func (l *swiftLexer) literalStep(escape string) {
	switch {
	case l.at(escape + "("):
		l.i += len(escape) + 1
		l.code(true)
	case l.at(escape):
		l.i += len(escape)
		l.escaped()
	case l.at("\n"):
		l.line++
		l.i++
	default:
		l.i++
	}
}

// escaped moves past an escaped character; a line break, which continues
// a multiline literal's line, is left for the line count.
func (l *swiftLexer) escaped() {
	if l.i < len(l.src) && !l.at("\n") {
		l.i++
	}
}

// quotedName lexes `name`, a name that may spell a keyword.
func (l *swiftLexer) quotedName() {
	end := strings.IndexAny(l.src[l.i+1:], "`\n")
	if end < 0 {
		end = len(l.src) - l.i - 1
	}
	l.emit(swiftQuoted, l.src[l.i+1:l.i+1+end], l.line)
	l.i = min(l.i+end+2, len(l.src))
}

func (l *swiftLexer) name() {
	start := l.i
	for l.i < len(l.src) && nameByte(l.src[l.i]) {
		l.i++
	}
	l.emit(swiftName, l.src[start:l.i], l.line)
}

// number lexes a numeric literal; a dot joins it only before a digit.
func (l *swiftLexer) number() {
	start := l.i
	for l.i < len(l.src) && (nameByte(l.src[l.i]) || l.decimalPoint()) {
		l.i++
	}
	l.emit(swiftNumber, l.src[start:l.i], l.line)
}

// decimalPoint reports a dot before a digit.
func (l *swiftLexer) decimalPoint() bool {
	return l.at(".") && l.i+1 < len(l.src) && digit(l.src[l.i+1])
}

// punct lexes one punctuation token; an arrow and the range operators
// stay whole, so the dots of a range are no member access.
func (l *swiftLexer) punct() {
	for _, p := range []string{"->", "...", "..<"} {
		if l.at(p) {
			l.i += len(p)
			l.emit(swiftPunct, p, l.line)
			return
		}
	}
	l.emit(swiftPunct, l.src[l.i:l.i+1], l.line)
	l.i++
}

// nameByte reports a byte of a name: ASCII letters, digits, _, $, and
// every byte of a non-ASCII character.
func nameByte(c byte) bool {
	return strings.IndexByte("_$", c) >= 0 || c >= 0x80 || digit(c) || letter(c)
}

func letter(c byte) bool { return (c|0x20) >= 'a' && (c|0x20) <= 'z' }

func digit(c byte) bool { return c >= '0' && c <= '9' }
