package comments

import "strings"

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
	depth     int
	text, raw bool
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
	case l.text:
		return Span{Part: Literal, From: i, To: skipTo(text, i, `"""`, &l.text)}
	case l.raw:
		return Span{Part: Code, From: i, To: skipTo(text, i, "`", &l.raw)}
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
		l.text = true
		return Span{Part: Literal, From: i, To: skipTo(text, i+3, `"""`, &l.text)}
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
		return Span{Part: Code, From: i, To: skipTo(text, i+1, "`", &l.raw)}
	}},
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
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// skipTo moves past the end of close, clearing flag when it is found.
func skipTo(text string, i int, close string, flag *bool) int {
	end := strings.Index(text[i:], close)
	if end < 0 {
		return len(text)
	}
	*flag = false
	return i + end + len(close)
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
