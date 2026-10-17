package codemodel

import (
	"slices"
	"sort"
	"strings"
)

// Node is one class of a code view that names a declaration.
type Node struct {
	Decl *Decl
}

// Span is a line range whose uses belong to a node. The smallest weight
// wins, and a declaration's own node beats the type that owns it.
type Span struct {
	Node     *Node
	File     string
	From, To int
	weight   int
}

func (s Span) holds(at Location) bool {
	return s.File == at.Path && s.From <= at.Line && at.Line <= s.To
}

func (s Span) beats(best *Span) bool {
	return best == nil || s.weight < best.weight
}

// Observed is one relation the source shows between two nodes.
type Observed struct {
	From, To *Node
	Kind     string
	At       Location
}

// Graph holds the nodes of a code view by the lines they own, and the
// relations the readers observe between them. A name reaches only the
// nodes of its own file's language.
type Graph struct {
	x        *Index
	language func(file string) string
	spans    []Span
	named    map[string][]*Node
	headers  map[Location]*Decl
	// Seen lists every relation observed, in order.
	Seen []Observed
}

// NewGraph starts a graph over the declarations of x. Language names the
// language a file is written in, which scopes the names it can reach.
func NewGraph(x *Index, language func(file string) string) *Graph {
	return &Graph{x: x, language: language, named: map[string][]*Node{}, headers: map[Location]*Decl{}}
}

// Add makes d a node: its own lines and, for a type, its methods and
// extensions belong to it, and its name reaches it.
func (g *Graph) Add(d *Decl) *Node {
	n := &Node{Decl: d}
	g.add(n, d, 0)
	if d.TypeLike() {
		g.typeSpans(n)
	}
	key := g.key(d.File, memberKey(d))
	g.named[key] = append(g.named[key], n)
	return n
}

// typeSpans adds a type's header lines, its methods, and its extensions.
func (g *Graph) typeSpans(n *Node) {
	d := n.Decl
	for line := d.Line; line <= max(d.Line, d.Open); line++ {
		g.headers[Location{d.File, line}] = d
	}
	for _, m := range g.x.Members(d) {
		if m.Callable() {
			g.add(n, m, 1)
		}
	}
	g.extensions(n)
}

// extensions adds each extension of a node's type in its scope: its lines
// and its header.
func (g *Graph) extensions(n *Node) {
	for _, e := range g.x.Named(n.Decl.Name, "", nil) {
		if e.Kind == KindExtension && e.Scope == n.Decl.Scope {
			g.add(n, e, 1)
			g.headers[e.At()] = n.Decl
		}
	}
}

func (g *Graph) add(n *Node, d *Decl, inherited int) {
	g.spans = append(g.spans, Span{Node: n, File: d.File, From: d.Line, To: d.End, weight: 2*(d.End-d.Line) + inherited})
}

// memberKey is the name source reaches d by: Type.member, by the last
// name of a nested owner, as code names it, or the name alone.
func memberKey(d *Decl) string {
	if d.Owner == "" {
		return d.Name
	}
	return d.Owner[strings.LastIndex(d.Owner, ".")+1:] + "." + d.Name
}

// key scopes a name to its file's language, so a name in one language
// never reaches a node in another.
func (g *Graph) key(file, name string) string {
	return g.language(file) + ":" + name
}

// Spans lists every node's line ranges, in the order the nodes were added.
func (g *Graph) Spans() []Span { return g.spans }

// Files lists the files of node spans that keep accepts, sorted.
func (g *Graph) Files(keep func(file string) bool) []string {
	var files []string
	for _, s := range g.spans {
		if keep(s.File) && !slices.Contains(files, s.File) {
			files = append(files, s.File)
		}
	}
	sort.Strings(files)
	return files
}

// Owner is the node whose smallest span holds a line, or nil.
func (g *Graph) Owner(at Location) *Node {
	var best *Span
	for i, s := range g.spans {
		if s.holds(at) && s.beats(best) {
			best = &g.spans[i]
		}
	}
	if best == nil {
		return nil
	}
	return best.Node
}

// Header is the node type whose header, or one of whose extensions'
// header, holds the line, or nil.
func (g *Graph) Header(at Location) *Decl { return g.headers[at] }

// Unique is the one node a name in file reaches among the nodes of its
// language, or nil. A member is named as Type.member.
func (g *Graph) Unique(file, name string) *Node {
	if nodes := g.named[g.key(file, name)]; len(nodes) == 1 {
		return nodes[0]
	}
	return nil
}

// Observe records a relation between two different nodes.
func (g *Graph) Observe(from, to *Node, kind string, at Location) {
	if from != nil && to != nil && from != to {
		g.Seen = append(g.Seen, Observed{From: from, To: to, Kind: kind, At: at})
	}
}

// Supers relates a type to each node the header on the line names: an
// interface a type implements is conformance, and a supertype is
// inheritance, or conformance when the type adopts an interface.
func (g *Graph) Supers(from *Node, d *Decl, at Location) {
	x := g.headerDecl(d, at)
	if x == nil {
		return
	}
	for _, name := range x.Supers {
		g.super(from, name, at, "")
	}
	for _, name := range x.Implements {
		g.super(from, name, at, RelConforms)
	}
}

// super relates a type to the node a header names, by kind, or by the
// kinds of both types when kind is empty.
func (g *Graph) super(from *Node, name string, at Location, kind string) {
	to := g.Unique(at.Path, name)
	if to == nil {
		return
	}
	if kind == "" {
		kind = superKind(from.Decl, to.Decl)
	}
	g.Observe(from, to, kind, at)
}

// headerDecl is the type or extension whose header starts on the line.
func (g *Graph) headerDecl(d *Decl, at Location) *Decl {
	for _, x := range g.x.Named(d.Name, "", []string{at.Path}) {
		if x.Line == at.Line {
			return x
		}
	}
	return nil
}

func superKind(from, to *Decl) string {
	if to.Kind == KindInterface && from.Kind != KindInterface {
		return RelConforms
	}
	return RelInherits
}
