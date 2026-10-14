package architecture

import (
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/javascript"
)

// at returns token i, or an empty punctuation token outside the file.
func (r *tsReader) at(i int) javascript.Token {
	if i < 0 || i >= len(r.tokens) {
		return javascript.Token{Kind: javascript.Punct, Pos: len(r.src)}
	}
	return r.tokens[i]
}

// word is the text of an identifier or punctuation token; a string or
// number reads as nothing.
func (r *tsReader) word(i int) string {
	if t := r.at(i); t.Kind == javascript.Ident || t.Kind == javascript.Punct {
		return t.Text
	}
	return ""
}

func (r *tsReader) is(i int, text string) bool { return r.word(i) == text }

func (r *tsReader) ident(i int) bool { return r.at(i).Kind == javascript.Ident }

// line is the line token i starts on.
func (r *tsReader) line(i int) int {
	return sort.SearchInts(r.starts, r.at(i).Pos+1)
}

func lineStarts(src string) []int {
	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// newline reports token i starting a later line than the token before.
func (r *tsReader) newline(i int) bool {
	return i > 0 && r.line(i) > r.line(i-1)
}

// arrow reports the > of a => arrow, which closes no type parameters.
func (r *tsReader) arrow(i int) bool {
	return r.is(i, ">") && r.is(i-1, "=") && r.at(i-1).Pos+1 == r.at(i).Pos
}

// qualified returns the last name of a dotted name starting at token j.
func (r *tsReader) qualified(j int) int {
	for r.is(j+1, ".") && r.ident(j+2) {
		j += 2
	}
	return j
}

// bracket is +1 for an opening parenthesis, bracket, or brace, -1 for a
// closing one, else 0.
func (r *tsReader) bracket(j int) int {
	t := r.at(j)
	if t.Kind != javascript.Punct || len(t.Text) != 1 {
		return 0
	}
	return strings.Count("([{", t.Text) - strings.Count(")]}", t.Text)
}

// close returns the token closing the bracket group token i opens.
func (r *tsReader) close(i int) int {
	depth := 0
	for j := i; j < len(r.tokens); j++ {
		if depth += r.bracket(j); depth == 0 {
			return j
		}
	}
	return len(r.tokens) - 1
}

// angle returns the > closing the type parameters token i opens.
func (r *tsReader) angle(i int) int {
	depth := 0
	for j := i; j < len(r.tokens); j++ {
		if depth += r.angleDepth(j); depth == 0 {
			return j
		}
	}
	return len(r.tokens) - 1
}

func (r *tsReader) angleDepth(j int) int {
	switch {
	case r.is(j, "<"):
		return 1
	case r.is(j, ">") && !r.arrow(j):
		return -1
	}
	return 0
}

// group returns the last token of the bracket group token j opens, or j.
func (r *tsReader) group(j int) int {
	if r.bracket(j) > 0 {
		return r.close(j)
	}
	return j
}

// typeGroup is group, with type parameters as a group too.
func (r *tsReader) typeGroup(j int) int {
	if r.is(j, "<") {
		return r.angle(j)
	}
	return r.group(j)
}

// end returns the last token of the statement starting at token i.
func (r *tsReader) end(i int) int {
	j := r.group(i)
	for j+1 < len(r.tokens) && !r.ends(j) {
		j = r.group(j + 1)
	}
	return j
}

// ends reports a statement ending at token j: on a semicolon, before the
// enclosing body's closing brace, or at a line break that automatic
// semicolon insertion ends it on.
func (r *tsReader) ends(j int) bool {
	return r.is(j, ";") || r.bracket(j+1) < 0 || r.breaks(j+1)
}

// continues reports a line ending at token j that the next line goes on.
func (r *tsReader) continues(j int) bool {
	joins := r.at(j).Kind == javascript.Punct && contains(tsJoiners, r.word(j)) && !r.postfix(j)
	return joins || contains(tsLeaders, r.word(j+1))
}

// typeEnd returns the token after the type starting at token i.
func (r *tsReader) typeEnd(i int) int {
	j := i
	for j < len(r.tokens) && !r.typeStops(i, j) {
		j = r.typeGroup(j) + 1
	}
	return j
}

// typeStops reports token j ending the type that starts at token i: a
// body's brace after a complete type, an = that is no arrow, a line the
// type does not continue on, or a closing token. A type starts after a
// colon, so a brace that starts it opens an object type.
func (r *tsReader) typeStops(i, j int) bool {
	switch {
	case r.is(j, "{"):
		return r.closesType(j - 1)
	case r.is(j, "="):
		return !r.arrow(j + 1)
	}
	return (j > i && r.breaks(j)) || contains(tsTypeEnds, r.word(j))
}

// breaks reports token j on a line the line before does not continue.
func (r *tsReader) breaks(j int) bool {
	return r.newline(j) && !r.continues(j-1)
}

// closesType reports a token that can end a type, so a brace after it
// opens a body rather than an object type.
func (r *tsReader) closesType(j int) bool {
	switch t := r.at(j); {
	case t.Kind != javascript.Punct:
		return true
	case r.is(j, ">"):
		return !r.arrow(j)
	}
	return r.bracket(j) < 0
}

// render spells tokens from up to to as a type, spaced as typeText does.
func (r *tsReader) render(from, to int) string {
	var parts []string
	for j := from; j < to && j < len(r.tokens); j++ {
		parts = append(parts, r.spell(j))
	}
	return typeText(strings.Join(parts, " "))
}

func (r *tsReader) spell(j int) string {
	switch t := r.tokens[j]; t.Kind {
	case javascript.String:
		return "'" + t.Text + "'"
	case javascript.Number, javascript.Regex:
		return r.literal(t.Pos)
	default:
		return t.Text
	}
}

// literal reads a number or regex as written, up to a space or a comma.
func (r *tsReader) literal(pos int) string {
	end := pos + 1
	for end < len(r.src) && !strings.ContainsRune(" \t\n,;)]}", rune(r.src[end])) {
		end++
	}
	return r.src[pos:end]
}

// tsPostfix read a sign that can close a value, by its text.
var tsPostfix = map[string]func(r *tsReader, j int) bool{
	"!": (*tsReader).assertion,
	"+": (*tsReader).step,
	"-": (*tsReader).step,
}

// postfix reports token j closing a value, so a line ending on it ends.
func (r *tsReader) postfix(j int) bool {
	read := tsPostfix[r.word(j)]
	return read != nil && read(r, j)
}

// assertion reports the ! of a non-null assertion such as el!.
func (r *tsReader) assertion(j int) bool {
	return r.attached(j) && r.value(j-1)
}

// step reports the second sign of i++ or i--.
func (r *tsReader) step(j int) bool {
	return r.is(j-1, r.word(j)) && r.attached(j) && r.value(j-2)
}

// attached reports token j written with no space before it.
func (r *tsReader) attached(j int) bool {
	pos := r.at(j).Pos
	return pos > 0 && !strings.ContainsRune(" \t\r\n", rune(r.src[pos-1]))
}

// value reports a token that ends a value: a name, a literal, or a closing
// parenthesis or bracket.
func (r *tsReader) value(j int) bool {
	if r.at(j).Kind != javascript.Punct {
		return true
	}
	return r.is(j, ")") || r.is(j, "]")
}
