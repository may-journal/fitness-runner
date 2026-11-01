package views

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
)

// at returns token i, or an empty punctuation token past the file's end.
func (r *reader) at(i int) token {
	if i < 0 || i >= len(r.tokens) {
		return token{kind: punct, pos: r.size}
	}
	return r.tokens[i]
}

// Len is the number of tokens.
func (r *reader) Len() int { return len(r.tokens) }

// Word is the text of a name or punctuation token; a literal reads as
// nothing.
func (r *reader) Word(i int) string {
	if t := r.at(i); t.kind != literal {
		return t.text
	}
	return ""
}

// Ident reports a name at token i.
func (r *reader) Ident(i int) bool { return r.at(i).kind == ident }

// Line is the line token i starts on.
func (r *reader) Line(i int) int { return r.lines.Line(r.at(i).pos) }

// Pos is the byte offset token i starts at.
func (r *reader) Pos(i int) int { return r.at(i).pos }

func (r *reader) is(i int, text string) bool { return r.Word(i) == text }

// seq reports the texts in order from token k.
func (r *reader) seq(k int, texts ...string) bool {
	for i, text := range texts {
		if !r.is(k+i, text) {
			return false
		}
	}
	return true
}

// qualified returns the last name of a dotted name starting at token j.
func (r *reader) qualified(j int) int {
	for r.is(j+1, ".") && r.Ident(j+2) {
		j += 2
	}
	return j
}

// brackets is how each bracket token changes the depth of bracket groups.
var brackets = map[string]int{"(": 1, "[": 1, "{": 1, ")": -1, "]": -1, "}": -1}

// angles is how each angle bracket changes the depth of type arguments.
var angles = map[string]int{"<": 1, ">": -1}

// bracket is +1 for an opening parenthesis, bracket, or brace, -1 for a
// closing one, else 0.
func (r *reader) bracket(j int) int {
	if r.at(j).kind != punct {
		return 0
	}
	return brackets[r.Word(j)]
}

// close returns the token closing the bracket group token i opens, else the
// last token or i.
func (r *reader) close(i int) int {
	return r.matching(i, r.bracket)
}

// angle returns the > closing the type arguments token i opens.
func (r *reader) angle(i int) int {
	return r.matching(i, func(j int) int { return angles[r.Word(j)] })
}

// matching returns the token that brings depth back to zero from token i,
// or the last token.
func (r *reader) matching(i int, step func(j int) int) int {
	depth := 0
	for j := i; j < len(r.tokens); j++ {
		if depth += step(j); depth == 0 {
			return j
		}
	}
	return len(r.tokens) - 1
}

// group returns the last token of the bracket group token j opens, or j.
func (r *reader) group(j int) int {
	if r.bracket(j) > 0 {
		return r.close(j)
	}
	return j
}

// generic reports whether the < at token i opens type arguments, holding only
// names and type punctuation up to its matching >.
func (r *reader) generic(i int) bool {
	depth := 0
	for j := i; j < len(r.tokens); j++ {
		if depth += angles[r.Word(j)]; depth == 0 {
			return true
		}
		if !r.typeArgument(j) {
			return false
		}
	}
	return false
}

// typeArgument reports a token type arguments may hold.
func (r *reader) typeArgument(j int) bool {
	return r.Ident(j) || (r.at(j).kind == punct && strings.Contains(".,?&[]@<>", r.Word(j)))
}

// typeStep returns the last token of the group token j opens: type
// arguments or a bracket group. Any other token is its own.
func (r *reader) typeStep(j int) int {
	if r.is(j, "<") && r.generic(j) {
		return r.angle(j)
	}
	return r.group(j)
}

// skipAngle returns the token after the type parameters token k opens,
// or k when it opens none.
func (r *reader) skipAngle(k int) int {
	if r.is(k, "<") {
		return r.angle(k) + 1
	}
	return k
}

// typeEnd returns the token after the type starting at token j, or j
// when none starts there.
func (r *reader) typeEnd(j int) int {
	if !r.Ident(j) {
		return j
	}
	k := j + 1
	for next := r.typePart(k); next > k; next = r.typePart(k) {
		k = next
	}
	return k
}

// typePart returns the token after one part of a type that goes on at
// token k, or k when the type ends there.
func (r *reader) typePart(k int) int {
	for _, part := range typeParts {
		if next := part(r, k); next > k {
			return next
		}
	}
	return k
}

// typeParts read what may follow a type's first name, each returning the token
// after its part, or k.
var typeParts = []func(r *reader, k int) int{
	func(r *reader, k int) int {
		if r.is(k, "<") && r.generic(k) {
			return r.angle(k) + 1
		}
		return k
	},
	func(r *reader, k int) int {
		if r.is(k, ".") && r.Ident(k+1) {
			return k + 2
		}
		return k
	},
	func(r *reader, k int) int { return r.after(k, "[", "]") },
	func(r *reader, k int) int { return r.after(k, ".", ".", ".") },
}

// after returns the token after the texts in order from token k, or k.
func (r *reader) after(k int, texts ...string) int {
	if r.seq(k, texts...) {
		return k + len(texts)
	}
	return k
}

// statementEnd returns the last token of the statement starting at token i,
// before any brace closing the body around it.
func (r *reader) statementEnd(i int) int {
	for j := i; j < len(r.tokens); j = r.group(j) + 1 {
		switch {
		case r.is(j, ";"):
			return j
		case r.is(j, "{"):
			return r.close(j)
		case r.bracket(j) < 0:
			return j - 1
		}
	}
	return len(r.tokens) - 1
}

// initializerEnd returns the comma or semicolon ending a field's declarator from
// token k, else the token before the body's closing brace.
func (r *reader) initializerEnd(k int) int {
	for j := k; j < len(r.tokens); j = r.typeStep(j) + 1 {
		switch {
		case r.is(j, ",") || r.is(j, ";"):
			return j
		case r.bracket(j) < 0:
			return j - 1
		}
	}
	return len(r.tokens) - 1
}

// render spells tokens from up to to as a type, spaced as TypeText does.
func (r *reader) render(from, to int) string {
	var parts []string
	for j := from; j < to && j < len(r.tokens); j++ {
		parts = append(parts, r.tokens[j].text)
	}
	return codemodel.TypeText(strings.Join(parts, " "))
}
