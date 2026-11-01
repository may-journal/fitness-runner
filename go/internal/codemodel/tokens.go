package codemodel

import "sort"

// LineIndex holds the byte offset each line of a source starts at, so a
// token's offset finds its line.
type LineIndex []int

// NewLineIndex indexes the lines of src.
func NewLineIndex(src string) LineIndex {
	starts := LineIndex{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// Line is the one-based line holding the byte at offset pos.
func (l LineIndex) Line(pos int) int {
	return sort.SearchInts(l, pos+1)
}

// Tokens is one file's tokens as a lexical reader reads them, so relations
// come from names, never from comments or literals.
type Tokens interface {
	// Len is the number of tokens.
	Len() int
	// Word is the text of a name or punctuation token; a literal, or an
	// index outside the file, reads as "".
	Word(i int) string
	// Ident reports a name at token i.
	Ident(i int) bool
	// Line is the line token i starts on.
	Line(i int) int
	// Pos is the byte offset token i starts at.
	Pos(i int) int
}

// RelateTokens relates the node holding each of a file's tokens.
func RelateTokens(g *Graph, file string, t Tokens) {
	for j := 0; j < t.Len(); j++ {
		relateToken(g, t, j, Location{Path: file, Line: t.Line(j)})
	}
}

// relateToken relates the node holding token j. A type header reads the
// types it extends and implements once, at its line's first token, and
// relates nothing else. Tokens after the body's brace relate as names,
// even on a declaration's first line.
func relateToken(g *Graph, t Tokens, j int, at Location) {
	from := g.Owner(at)
	switch d := g.Header(at); {
	case from == nil:
	case d != nil && t.Pos(j) < d.Body:
		relateHeader(g, t, j, from, d, at)
	case t.Ident(j):
		relateName(g, t, j, from, at)
	}
}

func relateHeader(g *Graph, t Tokens, j int, from *Node, d *Decl, at Location) {
	if j == 0 || t.Line(j-1) < at.Line {
		g.Supers(from, d, at)
	}
}

// relateName relates a name to the one node it names, or Type.member to
// that member's node. A name before an opening parenthesis calls;
// anything else references.
func relateName(g *Graph, t Tokens, j int, from *Node, at Location) {
	key, member := t.Word(j), t.Word(j-1) == "."
	if member {
		key = t.Word(j-2) + "." + key
	}
	if to := g.Unique(at.Path, key); to != nil && (!member || t.Ident(j-2)) {
		g.Observe(from, to, tokenKind(t, j), at)
	}
}

func tokenKind(t Tokens, j int) string {
	if t.Word(j+1) == "(" {
		return RelCalls
	}
	return RelReferences
}
