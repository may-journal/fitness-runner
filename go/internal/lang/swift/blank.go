package swift

import "strings"

// blanker copies Swift source with every comment and string literal turned
// to spaces. Newlines stay, so offsets and line numbers still match.
type blanker struct {
	src string
	out []byte
	i   int
	// keepStrings leaves string literals as written, so only comments blank.
	keepStrings bool
	// literals collects each string literal the blanker passes, inner
	// ones in an interpolation before the literal holding them.
	literals []Literal
}

// blankSwift returns src with its comments and string literals blanked.
func blankSwift(src string) string { return blanked(src, false) }

// blankComments returns src with only its comments blanked, so a word a
// string literal holds, such as a command name, still reads.
func blankComments(src string) string { return blanked(src, true) }

// blanked runs the blanker over src, keeping string literals if asked.
func blanked(src string, keepStrings bool) string {
	b := &blanker{src: src, out: []byte(src), keepStrings: keepStrings}
	for b.i < len(b.src) {
		b.step()
	}
	return string(b.out)
}

// step consumes one token: a comment, a string literal, or one byte of code.
func (b *blanker) step() {
	switch {
	case b.at("//"):
		b.lineComment()
	case b.at("/*"):
		b.blockComment()
	case b.src[b.i] == '#' || b.src[b.i] == '"':
		b.stringLiteral()
	default:
		b.i++
	}
}

// at reports whether the source continues with s.
func (b *blanker) at(s string) bool {
	return strings.HasPrefix(b.src[b.i:], s)
}

// lineComment blanks up to the end of the line.
func (b *blanker) lineComment() {
	start := b.i
	if end := strings.IndexByte(b.src[b.i:], '\n'); end >= 0 {
		b.i += end
	} else {
		b.i = len(b.src)
	}
	b.blank(start)
}

// blockComment blanks a /* */ comment, which Swift lets nest.
func (b *blanker) blockComment() {
	start, depth := b.i, 0
	for b.i < len(b.src) {
		depth += b.commentDelta()
		if depth == 0 {
			break
		}
	}
	b.blank(start)
}

// commentDelta consumes one byte or comment mark and returns how it changes
// the nesting depth.
func (b *blanker) commentDelta() int {
	for mark, delta := range map[string]int{"/*": 1, "*/": -1} {
		if b.at(mark) {
			b.i += len(mark)
			return delta
		}
	}
	b.i++
	return 0
}

// stringLiteral blanks a string, multi-line string, or raw #"…"# string. A
// '#' that opens no string, such as #expect, is code.
func (b *blanker) stringLiteral() {
	start := b.i
	hashes := strings.Repeat("#", len(b.src[b.i:])-len(strings.TrimLeft(b.src[b.i:], "#")))
	b.i += len(hashes)
	if !b.at(`"`) {
		return
	}
	quote := `"`
	if b.at(`"""`) {
		quote = `"""`
	}
	b.i += len(quote)
	var text strings.Builder
	for b.i < len(b.src) && !b.at(quote+hashes) {
		text.WriteString(b.stringChar(`\` + hashes))
	}
	b.i = min(b.i+len(quote+hashes), len(b.src))
	b.literals = append(b.literals, Literal{Start: start, End: b.i, Text: text.String()})
	b.blankString(start)
}

// stringChar consumes one character of string content, an escape, or a
// whole \( … ) interpolation, and returns the text it adds to the
// literal's words: the character, an escaped quote or backslash, and
// nothing for an interpolation or another escape.
func (b *blanker) stringChar(escape string) string {
	if !b.at(escape) {
		b.i++
		return b.src[b.i-1 : b.i]
	}
	b.i += len(escape)
	switch {
	case b.at("("):
		b.interpolation()
		return ""
	case b.at("u{"):
		b.i += strings.IndexByte(b.src[b.i:]+"}", '}') + 1
		return ""
	}
	b.i = min(b.i+1, len(b.src))
	return escapedWords[b.src[b.i-1]]
}

// escapedWords is the text an escaped character adds to a literal's words:
// a quote or backslash stands for itself, and a control character for none.
var escapedWords = map[byte]string{'"': `"`, '\'': "'", '\\': `\`}

// interpolation consumes balanced parentheses, skipping nested strings and
// comments.
func (b *blanker) interpolation() {
	depth := 0
	for b.i < len(b.src) {
		depth += parenDelta[b.src[b.i]]
		if depth == 0 {
			b.i++
			return
		}
		b.step()
	}
}

// parenDelta is how a byte changes parenthesis depth.
var parenDelta = map[byte]int{'(': 1, ')': -1}

// blankString blanks a string literal from start, unless strings are kept.
func (b *blanker) blankString(start int) {
	if !b.keepStrings {
		b.blank(start)
	}
}

// blank turns everything from start up to the cursor into spaces, keeping
// newlines.
func (b *blanker) blank(start int) {
	for i := start; i < b.i; i++ {
		if b.out[i] != '\n' {
			b.out[i] = ' '
		}
	}
}
