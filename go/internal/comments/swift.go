package comments

import "strings"

// SwiftVisitor receives what WalkSwift finds, by byte offsets into the
// source. String gets each string literal's span, delimiters and
// raw-string hashes included, with its words: the characters between the
// delimiters, without interpolations and with escapes read. A literal
// inside another's interpolation arrives before the literal holding it.
// Both funcs must be set.
type SwiftVisitor struct {
	Comment func(start, end int)
	String  func(start, end int, text string)
}

// WalkSwift walks Swift source token by token, reporting every comment,
// nested block comments whole, and every string, multi-line or raw.
func WalkSwift(src string, v SwiftVisitor) {
	w := &swiftWalker{src: src, v: v}
	for w.i < len(w.src) {
		w.step()
	}
}

type swiftWalker struct {
	src string
	i   int
	v   SwiftVisitor
}

// step consumes one token: a comment, a string literal, or one byte of code.
func (w *swiftWalker) step() {
	switch {
	case w.at("//"):
		w.lineComment()
	case w.at("/*"):
		w.blockComment()
	case w.src[w.i] == '#' || w.src[w.i] == '"':
		w.stringLiteral()
	default:
		w.i++
	}
}

// at reports whether the source continues with s.
func (w *swiftWalker) at(s string) bool {
	return strings.HasPrefix(w.src[w.i:], s)
}

// lineComment runs up to the end of the line.
func (w *swiftWalker) lineComment() {
	start := w.i
	if end := strings.IndexByte(w.src[w.i:], '\n'); end >= 0 {
		w.i += end
	} else {
		w.i = len(w.src)
	}
	w.comment(start)
}

// blockComment runs over a /* */ comment, which Swift lets nest.
func (w *swiftWalker) blockComment() {
	start, depth := w.i, 0
	for w.i < len(w.src) {
		depth += w.commentDelta()
		if depth == 0 {
			break
		}
	}
	w.comment(start)
}

func (w *swiftWalker) comment(start int) {
	w.v.Comment(start, w.i)
}

// commentDelta consumes one byte or comment mark and returns how it changes
// the nesting depth.
func (w *swiftWalker) commentDelta() int {
	for mark, delta := range map[string]int{"/*": 1, "*/": -1} {
		if w.at(mark) {
			w.i += len(mark)
			return delta
		}
	}
	w.i++
	return 0
}

// stringLiteral reads a string, multi-line string, or raw #"…"# string. A
// '#' that opens no string, such as #expect, is code.
func (w *swiftWalker) stringLiteral() {
	start := w.i
	hashes := strings.Repeat("#", len(w.src[w.i:])-len(strings.TrimLeft(w.src[w.i:], "#")))
	w.i += len(hashes)
	if !w.at(`"`) {
		return
	}
	text := w.body(w.quote()+hashes, `\`+hashes)
	w.v.String(start, w.i, text)
}

// quote moves past a string's opening quote, three for a multi-line
// string, and returns it.
func (w *swiftWalker) quote() string {
	quote := `"`
	if w.at(`"""`) {
		quote = `"""`
	}
	w.i += len(quote)
	return quote
}

// body reads a string's content up to and past close, and returns its
// words.
func (w *swiftWalker) body(close, escape string) string {
	var text strings.Builder
	for w.i < len(w.src) && !w.at(close) {
		text.WriteString(w.stringChar(escape))
	}
	w.i = min(w.i+len(close), len(w.src))
	return text.String()
}

// stringChar consumes one character of string content, an escape, or a
// whole \( … ) interpolation, and returns the text it adds to the
// literal's words: the character, an escaped quote or backslash, and
// nothing for an interpolation or another escape.
func (w *swiftWalker) stringChar(escape string) string {
	if !w.at(escape) {
		w.i++
		return w.src[w.i-1 : w.i]
	}
	w.i += len(escape)
	switch {
	case w.at("("):
		w.interpolation()
		return ""
	case w.at("u{"):
		if end := strings.IndexByte(w.src[w.i:], '}'); end >= 0 {
			w.i += end + 1
		} else {
			w.i = len(w.src)
		}
		return ""
	}
	w.i = min(w.i+1, len(w.src))
	return escapedWords[w.src[w.i-1]]
}

// escapedWords is the text an escaped character adds to a literal's words:
// a quote or backslash stands for itself, and a control character for none.
var escapedWords = map[byte]string{'"': `"`, '\'': "'", '\\': `\`}

// interpolation consumes balanced parentheses, skipping nested strings and
// comments.
func (w *swiftWalker) interpolation() {
	depth := 0
	for w.i < len(w.src) {
		depth += parenDelta[w.src[w.i]]
		if depth == 0 {
			w.i++
			return
		}
		w.step()
	}
}

// parenDelta is how a byte changes parenthesis depth.
var parenDelta = map[byte]int{'(': 1, ')': -1}
