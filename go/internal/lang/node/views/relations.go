package views

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
)

// relate reads each TypeScript node file's tokens, so comments and strings
// relate nothing: heritage clauses, then names of other TypeScript nodes.
func relate(g *codemodel.Graph, repo codemodel.Repo) {
	for _, file := range g.Files(typeScript) {
		t := newTSReader(file, strings.Join(repo.Read(file), "\n"))
		for j := range t.tokens {
			relateToken(g, t, j, codemodel.Location{Path: file, Line: t.line(j)})
		}
	}
}

// relateToken relates the node holding token j. A class or interface
// header reads its heritage clause once, at the line's first token, and
// relates nothing else. Tokens after the body's brace, or in a type alias
// or enum, relate as names, even on a declaration's first line.
func relateToken(g *codemodel.Graph, t *tsReader, j int, at codemodel.Location) {
	from := g.Owner(at)
	switch d := g.Header(at); {
	case from == nil:
	case d != nil && t.at(j).Pos < d.Body:
		relateHeader(g, t, j, from, d, at)
	case t.ident(j):
		relateName(g, t, j, from, at)
	}
}

func relateHeader(g *codemodel.Graph, t *tsReader, j int, from *codemodel.Node, d *codemodel.Decl, at codemodel.Location) {
	if j == 0 || t.line(j-1) < at.Line {
		g.Supers(from, d, at)
	}
}

// relateName relates a name to the one node it names, or Type.member to
// that member's node. A call or new expression calls; anything else
// references.
func relateName(g *codemodel.Graph, t *tsReader, j int, from *codemodel.Node, at codemodel.Location) {
	key := t.word(j)
	if t.is(j-1, ".") {
		key = t.word(j-2) + "." + key
	}
	if to := g.Unique(at.Path, key); to != nil && (!t.is(j-1, ".") || t.ident(j-2)) {
		g.Observe(from, to, scriptKind(t, j), at)
	}
}

func scriptKind(t *tsReader, j int) string {
	if t.is(j+1, "(") {
		return codemodel.RelCalls
	}
	return codemodel.RelReferences
}
