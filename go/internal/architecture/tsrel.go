package architecture

import "strings"

// scriptUses reads each TypeScript node file's tokens, so comments and
// strings relate nothing: heritage clauses, then names of other
// TypeScript nodes.
func (r *relations) scriptUses() {
	for _, file := range r.files(typeScript) {
		t := newTSReader(strings.Join(r.c.m.repo.read(file), "\n"))
		for j := range t.tokens {
			r.scriptToken(t, j, Location{file, t.line(j)})
		}
	}
}

// scriptToken relates the node holding token j. A header line reads its
// heritage clause once, at its first token, and relates nothing else.
func (r *relations) scriptToken(t *tsReader, j int, at Location) {
	from := r.owner(at)
	switch d := r.headers[at]; {
	case from == nil:
	case d != nil:
		r.scriptHeader(t, j, from, d, at)
	case t.ident(j):
		r.scriptName(t, j, from, at)
	}
}

func (r *relations) scriptHeader(t *tsReader, j int, from *codeNode, d *decl, at Location) {
	if j == 0 || t.line(j-1) < at.Line {
		r.supers(from, d, at)
	}
}

// scriptName relates a name to the one node it names, or Type.member to
// that member's node. A call or new expression calls; anything else
// references.
func (r *relations) scriptName(t *tsReader, j int, from *codeNode, at Location) {
	key := t.word(j)
	if t.is(j-1, ".") {
		key = t.word(j-2) + "." + key
	}
	if to := r.unique(at.Path, key); to != nil && (!t.is(j-1, ".") || t.ident(j-2)) {
		r.observe(from, to, scriptKind(t, j), at)
	}
}

func scriptKind(t *tsReader, j int) string {
	if t.is(j+1, "(") {
		return relCalls
	}
	return relReferences
}
