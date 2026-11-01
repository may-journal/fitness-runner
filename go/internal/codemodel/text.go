package codemodel

import (
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/comments"
)

// Scanner strips comments, and string literals when Literals is set, from
// source in the common C-style syntax that comments.Lexer reads. Neither
// comments nor literals can create a dependency. A span in backticks, which
// may run over lines, is kept whole.
type Scanner struct {
	Literals bool
	lex      comments.Lexer
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

// Line returns one line's code, carrying an open comment, text block, or
// span in backticks over to the next line.
func (s *Scanner) Line(text string) string {
	var out strings.Builder
	for _, sp := range s.lex.Line(text) {
		if sp.Part == comments.Code || sp.Part == comments.Literal && !s.Literals {
			out.WriteString(text[sp.From:sp.To])
		}
	}
	return out.String()
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
