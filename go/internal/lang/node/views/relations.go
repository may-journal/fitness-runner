package views

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
)

// relate reads each TypeScript node file's tokens, so comments and strings
// relate nothing: heritage clauses, then names of other TypeScript nodes.
func relate(g *codemodel.Graph, repo codemodel.Repo) {
	for _, file := range g.Files(typeScript) {
		codemodel.RelateTokens(g, file, newTSReader(file, strings.Join(repo.Read(file), "\n")))
	}
}

// Len is the number of tokens, for relating names.
func (r *tsReader) Len() int { return len(r.tokens) }

// Word is the text of a name or punctuation token, for relating names.
func (r *tsReader) Word(i int) string { return r.word(i) }

// Ident reports a name at token i, for relating names.
func (r *tsReader) Ident(i int) bool { return r.ident(i) }

// Line is the line token i starts on, for relating names.
func (r *tsReader) Line(i int) int { return r.line(i) }

// Pos is the byte offset token i starts at, for relating names.
func (r *tsReader) Pos(i int) int { return r.at(i).Pos }
