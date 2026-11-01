package comments

import (
	"regexp"
	"strings"
	"unicode"
)

// Part is what one span of C-style source holds.
type Part int

const (
	// Code is source a compiler reads, character literals and spans in
	// backticks included.
	Code Part = iota
	// Literal is a "..." or """...""" string, its quotes included.
	Literal
	// LineComment is a // comment, running to the line's end.
	LineComment
	// BlockComment is a /* */ comment, or the part of one on this line.
	BlockComment
)

// Span is one run of a line, from byte From up to To. Open marks a block
// comment that runs on to the next line.
type Span struct {
	Part     Part
	From, To int
	Open     bool
}

// Lexer splits source in the common C-style syntax into spans, a line at a
// time: // and /* */ comments, "..." and """...""" strings, '...'
// character literals, and spans in backticks. Literals are always read, so
// a // or /* inside one starts no comment. An open block comment, text
// block, or span in backticks carries over to the next line.
type Lexer struct {
	// Nested lets block comments nest, as Swift, Kotlin and Rust do.
	Nested bool
	// Lifetimes reads a ' before a name as a Rust lifetime, which opens no
	// character literal.
	Lifetimes bool
	// Templates lets a backslash escape a backtick inside a span in
	// backticks, as in a JavaScript template literal.
	Templates bool
	// RawStrings reads Rust raw strings, r"..." and r#"..."#, which have
	// no escapes.
	RawStrings bool
	depth      int
	// closer ends the string literal open over lines, if any.
	closer string
	raw    bool
}

// Line returns the spans of one line, in order, covering all of it.
func (l *Lexer) Line(text string) []Span {
	var spans []Span
	for i := 0; i < len(text); {
		sp := l.step(text, i)
		if n := len(spans); n > 0 && sp.Part == Code && spans[n-1].Part == Code {
			spans[n-1].To = sp.To
		} else {
			spans = append(spans, sp)
		}
		i = sp.To
	}
	return spans
}

func (l *Lexer) step(text string, i int) Span {
	switch {
	case l.depth > 0:
		return l.block(text, i, i)
	case l.closer != "":
		return Span{Part: Literal, From: i, To: l.until(text, i)}
	case l.raw:
		return Span{Part: Code, From: i, To: l.rawEnd(text, i)}
	}
	return l.code(text, i)
}

func (l *Lexer) code(text string, i int) Span {
	for _, o := range openers {
		if strings.HasPrefix(text[i:], o.prefix) {
			return o.read(l, text, i)
		}
	}
	return Span{Part: Code, From: i, To: i + 1}
}

// opener reads the token that its prefix opens at byte i.
type opener struct {
	prefix string
	read   func(l *Lexer, text string, i int) Span
}

// openers are tried in order, so """ wins over ".
var openers = []opener{
	{"//", func(_ *Lexer, text string, i int) Span {
		return Span{Part: LineComment, From: i, To: len(text)}
	}},
	{"/*", func(l *Lexer, text string, i int) Span {
		l.depth = 1
		return l.block(text, i, i+2)
	}},
	{`"""`, func(l *Lexer, text string, i int) Span {
		l.closer = `"""`
		return Span{Part: Literal, From: i, To: l.until(text, i+3)}
	}},
	{`"`, func(_ *Lexer, text string, i int) Span {
		return Span{Part: Literal, From: i, To: literalEnd(text, i+1)}
	}},
	{"'", func(l *Lexer, text string, i int) Span {
		if l.lifetime(text, i) {
			return Span{Part: Code, From: i, To: i + 1}
		}
		return Span{Part: Code, From: i, To: runeEnd(text, i)}
	}},
	{"`", func(l *Lexer, text string, i int) Span {
		l.raw = true
		return Span{Part: Code, From: i, To: l.rawEnd(text, i+1)}
	}},
	{"r", (*Lexer).rawString},
}

// until moves past the end of the open string's closer, clearing it when
// found.
func (l *Lexer) until(text string, i int) int {
	end := strings.Index(text[i:], l.closer)
	if end < 0 {
		return len(text)
	}
	to := i + end + len(l.closer)
	l.closer = ""
	return to
}

// rawEnd moves past the backtick closing a span, clearing raw when found.
func (l *Lexer) rawEnd(text string, i int) int {
	for ; i < len(text); i++ {
		if text[i] == '\\' && l.Templates {
			i++
			continue
		}
		if text[i] == '`' {
			l.raw = false
			return i + 1
		}
	}
	return len(text)
}

// rustRaw opens a Rust raw string and captures its hashes.
var rustRaw = regexp.MustCompile(`^r(#*)"`)

// rawString reads a Rust raw string at i, or the r as code when none opens
// there or the r ends a name.
func (l *Lexer) rawString(text string, i int) Span {
	m := rustRaw.FindStringSubmatch(text[i:])
	if !l.RawStrings || m == nil || i > 0 && nameByte(text[i-1]) {
		return Span{Part: Code, From: i, To: i + 1}
	}
	l.closer = `"` + m[1]
	return Span{Part: Literal, From: i, To: l.until(text, i+len(m[0]))}
}

// block moves through a block comment from j, which started at from,
// until it closes or the line ends.
func (l *Lexer) block(text string, from, j int) Span {
	for j < len(text) && l.depth > 0 {
		j += l.mark(text[j:])
	}
	return Span{Part: BlockComment, From: from, To: j, Open: l.depth > 0}
}

// mark moves over the comment mark that opens rest, changing the depth,
// or over one byte, and returns how far it moved.
func (l *Lexer) mark(rest string) int {
	if strings.HasPrefix(rest, "*/") {
		l.depth--
		return 2
	}
	if l.Nested && strings.HasPrefix(rest, "/*") {
		l.depth++
		return 2
	}
	return 1
}

// lifetime reports a Rust lifetime such as 'a, where no quote closes the
// name right after it.
func (l *Lexer) lifetime(text string, i int) bool {
	return l.Lifetimes && i+1 < len(text) && nameByte(text[i+1]) && (i+2 >= len(text) || text[i+2] != '\'')
}

func nameByte(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

// runeEnd moves past a character literal, kept whole so a quote or //
// inside one starts neither a string nor a comment.
func runeEnd(text string, i int) int {
	for j := i + 1; j < len(text); j++ {
		switch text[j] {
		case '\\':
			j++
		case '\'':
			return j + 1
		}
	}
	return len(text)
}

// literalEnd moves past a one-line string literal, escapes included.
func literalEnd(text string, i int) int {
	for ; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return min(i, len(text))
}
