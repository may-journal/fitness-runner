package codemodel

import (
	"strings"
	"unicode"
)

// Scanner strips comments, and string literals when Literals is set, from
// source in the common C-style syntax: // and /* */ comments, "..." and
// """...""" strings, '...' character literals, and spans in backticks.
// Neither comments nor literals can create a dependency. Literals are
// always read, so a // or /* inside one starts no comment. A span in
// backticks, which may run over lines, is kept whole.
type Scanner struct {
	Literals         bool
	block, text, raw bool
}

// Strip returns the lines with comments removed and literals kept.
func Strip(lines []string) []string {
	sc := &Scanner{}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = sc.Line(line)
	}
	return out
}

// keep writes a span of string literal to out unless literals are stripped.
func (s *Scanner) keep(text string, from, to int, out *strings.Builder) int {
	if !s.Literals {
		out.WriteString(text[from:to])
	}
	return to
}

// Line returns one line's code, carrying an open comment, text block, or
// span in backticks over to the next line.
func (s *Scanner) Line(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		i = s.step(text, i, &out)
	}
	return out.String()
}

func (s *Scanner) step(text string, i int, out *strings.Builder) int {
	switch {
	case s.block:
		return s.skipTo(text, i, "*/", &s.block)
	case s.text:
		return s.keep(text, i, s.skipTo(text, i, `"""`, &s.text), out)
	case s.raw:
		next := s.skipTo(text, i, "`", &s.raw)
		out.WriteString(text[i:next])
		return next
	}
	return s.code(text, i, out)
}

// skipTo moves past the end of close, clearing flag when it is found.
func (s *Scanner) skipTo(text string, i int, close string, flag *bool) int {
	end := strings.Index(text[i:], close)
	if end < 0 {
		return len(text)
	}
	*flag = false
	return i + end + len(close)
}

func (s *Scanner) code(text string, i int, out *strings.Builder) int {
	if next, ok := s.comment(text, i); ok {
		return next
	}
	if next, ok := s.quote(text, i); ok {
		return s.keep(text, i, next, out)
	}
	switch text[i] {
	case '\'':
		next := runeEnd(text, i)
		out.WriteString(text[i:next])
		return next
	case '`':
		s.raw = true
	}
	out.WriteByte(text[i])
	return i + 1
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

func (s *Scanner) comment(text string, i int) (int, bool) {
	switch {
	case strings.HasPrefix(text[i:], "//"):
		return len(text), true
	case strings.HasPrefix(text[i:], "/*"):
		s.block = true
		return i + 2, true
	}
	return 0, false
}

func (s *Scanner) quote(text string, i int) (int, bool) {
	switch {
	case text[i] != '"':
		return 0, false
	case strings.HasPrefix(text[i:], `"""`):
		s.text = true
		return i + 3, true
	}
	return s.literal(text, i+1), true
}

// literal moves past a one-line string literal, escapes included.
func (s *Scanner) literal(text string, i int) int {
	for ; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return i
}

// Matching finds the bracket closing the one at open, or the text's end.
func Matching(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		depth += arrowSafeDepth(text, i)
		if depth == 0 {
			return i
		}
	}
	return len(text) - 1
}

func bracketDepth(c byte) int {
	switch c {
	case '(', '[', '<', '{':
		return 1
	case ')', ']', '>', '}':
		return -1
	}
	return 0
}

// SplitTop splits on commas outside brackets; an empty text has no items.
func SplitTop(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var items []string
	depth, start := 0, 0
	for i := 0; i < len(text); i++ {
		depth += arrowSafeDepth(text, i)
		if text[i] == ',' && depth == 0 {
			items, start = append(items, strings.TrimSpace(text[start:i])), i+1
		}
	}
	return append(items, strings.TrimSpace(text[start:]))
}

// arrowSafeDepth is bracketDepth, except the > of a -> arrow closes
// nothing.
func arrowSafeDepth(text string, i int) int {
	if text[i] == '>' && i > 0 && text[i-1] == '-' {
		return 0
	}
	return bracketDepth(text[i])
}

// TypeText spells a type with a space only between two words and after
// each comma, so Map< K,V > and Map<K, V> compare equal.
func TypeText(text string) string {
	var b strings.Builder
	words := strings.Fields(text)
	for i, word := range words {
		if i > 0 && WordByte(words[i-1][len(words[i-1])-1]) && WordByte(word[0]) {
			b.WriteByte(' ')
		}
		b.WriteString(word)
	}
	return strings.ReplaceAll(b.String(), ",", ", ")
}

// WordByte reports a byte that can be part of a name.
func WordByte(c byte) bool {
	return c == '_' || c == '$' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

// NormalSpace collapses each run of white space to one space.
func NormalSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
