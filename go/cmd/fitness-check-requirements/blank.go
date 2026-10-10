package main

import "strings"

// blanker copies Swift source with every comment and string literal turned
// to spaces. Newlines stay, so offsets and line numbers still match.
type blanker struct {
	src string
	out []byte
	i   int
}

// blankSwift returns src with its comments and string literals blanked.
func blankSwift(src string) string {
	b := &blanker{src: src, out: []byte(src)}
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
	for b.i < len(b.src) && !b.at(quote+hashes) {
		b.stringChar(`\` + hashes)
	}
	b.i = min(b.i+len(quote+hashes), len(b.src))
	b.blank(start)
}

// stringChar consumes one character of string content, an escape, or a
// whole \( … ) interpolation.
func (b *blanker) stringChar(escape string) {
	if b.at(escape) {
		b.i += len(escape)
		if b.at("(") {
			b.interpolation()
			return
		}
	}
	b.i++
}

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

// blank turns everything from start up to the cursor into spaces, keeping
// newlines.
func (b *blanker) blank(start int) {
	for i := start; i < b.i; i++ {
		if b.out[i] != '\n' {
			b.out[i] = ' '
		}
	}
}
