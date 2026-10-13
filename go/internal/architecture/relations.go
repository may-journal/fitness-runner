package architecture

import (
	"path"
	"regexp"
	"strings"
)

// identifier matches a name in a line of Swift code.
var identifier = regexp.MustCompile(`[A-Za-z_]\w*`)

func afterDot(code string, start int) bool {
	return start > 0 && code[start-1] == '.'
}

// span is a line range whose references belong to a node. The smallest
// weight wins, and a declaration's own node beats the type that owns it.
type span struct {
	n                *codeNode
	file             string
	from, to, weight int
}

// observed is one relation the source shows between two concrete nodes.
type observed struct {
	from, to *codeNode
	kind     string
	at       Location
}

type relations struct {
	c       *code
	spans   []span
	seen    []observed
	swift   map[string][]*codeNode
	headers map[Location]*decl
}

// relate reads the relations between concrete nodes from Go type data and
// Swift declarations, then compares them with the drawn edges both ways.
func (c *code) relate() {
	r := &relations{c: c, swift: map[string][]*codeNode{}, headers: map[Location]*decl{}}
	for _, n := range c.order {
		r.node(n)
	}
	r.goUses()
	r.swiftUses()
	for _, e := range c.m.Relationships {
		r.drawn(e)
	}
	r.strays()
}

// node adds a concrete node's own lines and, for a type, its methods and
// extensions, then indexes a Swift node by name.
func (r *relations) node(n *codeNode) {
	if n.d == nil {
		return
	}
	r.add(n, n.d, 0)
	if n.d.typeLike() {
		r.typeSpans(n)
	}
	if n.d.Package == "" {
		r.swift[swiftKey(n.d)] = append(r.swift[swiftKey(n.d)], n)
	}
}

func (r *relations) typeSpans(n *codeNode) {
	r.headers[n.d.at()] = n.d
	for _, d := range filter(r.c.x.members(n.d), (*decl).callable) {
		r.add(n, d, 1)
	}
	for _, d := range r.c.x.named(n.d.Name, "", nil) {
		r.extension(n, d)
	}
}

// extension adds a Swift extension of the type: its lines and its
// inheritance clause.
func (r *relations) extension(n *codeNode, d *decl) {
	if d.Kind == kindExtension && n.d.Package == "" {
		r.add(n, d, 1)
		r.headers[d.at()] = n.d
	}
}

func (r *relations) add(n *codeNode, d *decl, inherited int) {
	r.spans = append(r.spans, span{n: n, file: d.File, from: d.Line, to: d.End, weight: 2*(d.End-d.Line) + inherited})
}

func swiftKey(d *decl) string {
	if d.Owner == "" {
		return d.Name
	}
	return d.Owner + "." + d.Name
}

// owner is the node whose smallest span holds a line.
func (r *relations) owner(at Location) *codeNode {
	var best *span
	for i, s := range r.spans {
		if s.holds(at) && s.beats(best) {
			best = &r.spans[i]
		}
	}
	return best.node()
}

func (s span) holds(at Location) bool {
	return s.file == at.Path && s.from <= at.Line && at.Line <= s.to
}

func (s span) beats(best *span) bool {
	return best == nil || s.weight < best.weight
}

func (s *span) node() *codeNode {
	if s == nil {
		return nil
	}
	return s.n
}

func (r *relations) observe(from, to *codeNode, kind string, at Location) {
	if from != nil && to != nil && from != to {
		r.seen = append(r.seen, observed{from: from, to: to, kind: kind, at: at})
	}
}

// goUses reads the uses in each Go node's lines through go/types, so
// comments, strings, and a name that merely matches never relate two nodes.
func (r *relations) goUses() {
	for _, s := range r.spans {
		if path.Ext(s.file) == ".go" {
			r.goSpan(s)
		}
	}
}

func (r *relations) goSpan(s span) {
	visited := map[*decl]bool{}
	for _, u := range r.c.g.usesIn(s.file, s.from, s.to) {
		if r.owner(u.At) == s.n {
			r.follow(s.n, u, u.At, visited)
		}
	}
}

// follow relates a node to the node a use reaches. A call to a function
// the view does not show is followed into that function, so a diagram may
// leave helpers out; the relation is reported at the node's own line.
func (r *relations) follow(n *codeNode, u use, at Location, visited map[*decl]bool) {
	if to := r.owner(u.To); to != nil {
		r.observe(n, to, u.Kind, at)
		return
	}
	helper := r.c.g.callee(u)
	if helper == nil || visited[helper] {
		return
	}
	visited[helper] = true
	for _, next := range r.c.g.usesIn(helper.File, helper.Line, helper.End) {
		r.follow(n, next, at, visited)
	}
}

// swiftUses reads each Swift node file with comments and literals
// stripped: inheritance clauses, then names of other Swift nodes.
func (r *relations) swiftUses() {
	files := map[string]bool{}
	for _, s := range r.spans {
		if path.Ext(s.file) == ".swift" {
			files[s.file] = true
		}
	}
	for _, file := range sortedKeys(files) {
		r.swiftFile(file)
	}
}

func (r *relations) swiftFile(file string) {
	sc := &scanner{literals: true}
	for i, line := range r.c.m.repo.read(file) {
		at := Location{file, i + 1}
		code := sc.line(line)
		if from := r.owner(at); from != nil {
			r.swiftLine(from, code, at)
		}
	}
}

func (r *relations) swiftLine(from *codeNode, code string, at Location) {
	if d := r.headers[at]; d != nil {
		r.supers(from, d, at)
		return
	}
	for _, span := range identifier.FindAllStringIndex(code, -1) {
		if to := r.swiftTarget(code, span); to != nil {
			r.observe(from, to, swiftKind(code, span[1]), at)
		}
	}
}

// supers relates a type to each node its inheritance clause names: a
// protocol a type adopts is conformance; anything else is inheritance.
func (r *relations) supers(from *codeNode, d *decl, at Location) {
	for _, name := range r.headerSupers(d, at) {
		if to := r.unique(name); to != nil {
			r.observe(from, to, superKind(from.d, to.d), at)
		}
	}
}

func (r *relations) headerSupers(d *decl, at Location) []string {
	for _, x := range r.c.x.named(d.Name, "", []string{at.Path}) {
		if x.Line == at.Line {
			return x.Supers
		}
	}
	return nil
}

func superKind(from, to *decl) string {
	if to.Kind == kindInterface && from.Kind != kindInterface {
		return relConforms
	}
	return relInherits
}

// swiftTarget resolves an identifier to the one node of that name, or
// Type.member to that member's node.
func (r *relations) swiftTarget(code string, span []int) *codeNode {
	name := code[span[0]:span[1]]
	if !afterDot(code, span[0]) {
		return r.unique(name)
	}
	owner := identifier.FindAllString(code[:span[0]-1], -1)
	if len(owner) == 0 || !strings.HasSuffix(code[:span[0]-1], owner[len(owner)-1]) {
		return nil
	}
	return r.unique(owner[len(owner)-1] + "." + name)
}

func (r *relations) unique(key string) *codeNode {
	if nodes := r.swift[key]; len(nodes) == 1 {
		return nodes[0]
	}
	return nil
}

func swiftKind(code string, end int) string {
	if strings.HasPrefix(strings.TrimSpace(code[end:]), "(") {
		return relCalls
	}
	return relReferences
}
