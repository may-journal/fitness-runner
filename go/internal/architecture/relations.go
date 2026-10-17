package architecture

import "github.com/may-journal/fitness-runner/go/internal/codemodel"

// observed is one relation the source shows between two concrete nodes.
type observed struct {
	from, to *codeNode
	kind     string
	at       Location
}

type relations struct {
	c    *code
	seen []observed
}

// relate has each module's reader observe the relations between concrete
// nodes in its own files, in registry order, then compares them with the
// drawn edges both ways.
func (c *code) relate() {
	r := &relations{c: c, seen: c.observe()}
	for _, e := range c.m.Relationships {
		r.drawn(e)
	}
	r.strays()
}

// observe has each reader relate the concrete nodes, and returns the
// relations seen as view nodes.
func (c *code) observe() []observed {
	g, nodes := c.graph()
	for _, r := range viewReaders {
		if read := c.reads[r.Language]; read.Relate != nil {
			read.Relate(g)
		}
	}
	seen := make([]observed, len(g.Seen))
	for i, o := range g.Seen {
		seen[i] = observed{from: nodes[o.From], to: nodes[o.To], kind: o.Kind, at: Location(o.At)}
	}
	return seen
}

// graph adds every concrete node to a graph, and maps each graph node back
// to its view node.
func (c *code) graph() (*codemodel.Graph, map[*codemodel.Node]*codeNode) {
	g := codemodel.NewGraph(c.x, language)
	nodes := map[*codemodel.Node]*codeNode{}
	for _, n := range c.order {
		if n.d != nil {
			nodes[g.Add(n.d)] = n
		}
	}
	return g, nodes
}
