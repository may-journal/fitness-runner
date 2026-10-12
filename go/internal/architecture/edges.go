package architecture

import "strings"

// Edge classes a class diagram arrow draws.
const (
	edgeAssociation = "association"
	edgeLink        = "link"
	edgeInheritance = "inheritance"
	edgeRealization = "realization"
)

// arrows maps each supported Mermaid arrow to its class. An arrow that
// starts with < was already read from its other end.
var arrows = map[string]string{
	"-->": edgeAssociation, "<--": edgeAssociation, "..>": edgeAssociation, "<..": edgeAssociation,
	"*--": edgeAssociation, "o--": edgeAssociation, "--": edgeLink, "..": edgeLink,
	"<|--": edgeInheritance, "--|>": edgeInheritance, "<|..": edgeRealization, "..|>": edgeRealization,
}

// accepted lists the source relations each edge class allows.
var accepted = map[string][]string{
	edgeAssociation: {relCalls, relReferences, relEmbeds},
	edgeLink:        {relCalls, relReferences, relEmbeds},
	edgeInheritance: {relEmbeds, relInherits},
	edgeRealization: {relConforms},
}

// drawn checks one code view edge. An edge that touches an abstract node
// is allowed and needs no use; an edge between two concrete nodes is
// required and must be shown by the source.
func (r *relations) drawn(e Relationship) {
	from, to := r.c.nodes[e.From], r.c.nodes[e.To]
	if e.View == codeView && from != nil && to != nil {
		r.edge(e, arrows[e.Arrow], from, to)
	}
}

func (r *relations) edge(e Relationship, class string, from, to *codeNode) {
	switch {
	case class == "":
		r.c.fail(e.At, "unsupported relation %s from %s to %s; draw -->, ..>, <|--, or ..|>", e.Arrow, e.From, e.To)
	case abstractEdge(from, to):
		r.abstractEvidence(e)
	case from.d != nil && to.d != nil:
		r.required(e, class, from, to)
	}
}

func abstractEdge(from, to *codeNode) bool {
	return from.abstract || to.abstract
}

// abstractEvidence fails a link on an edge no source can prove.
func (r *relations) abstractEvidence(e Relationship) {
	for _, ev := range r.edgeEvidence(e) {
		r.c.fail(ev.At, "unsupported evidence %s; %s -> %s touches an abstract node, which no source declares", ev, e.From, e.To)
	}
}

func (r *relations) edgeEvidence(e Relationship) []Evidence {
	var found []Evidence
	for _, ev := range r.c.m.Evidence {
		if ev.Subject == e.From+"->"+e.To && ev.At.Path == e.At.Path {
			found = append(found, ev)
		}
	}
	return found
}

func (r *relations) required(e Relationship, class string, from, to *codeNode) {
	if !r.proved(class, from, to, nil) {
		r.c.fail(e.At, "%s draws %s %s -> %s, but no source shows %s %s %s", r.c.viewFile(), class, e.From, e.To, e.From, strings.Join(accepted[class], " or "), e.To)
	}
	for _, ev := range r.edgeEvidence(e) {
		if !r.proved(class, from, to, &ev) {
			r.c.fail(ev.At, "evidence %s shows no %s -> %s relation", ev, e.From, e.To)
		}
	}
}

// proved finds a relation the edge class allows, within the linked lines
// when evidence is given. Go satisfies an interface implicitly, so a Go
// realization is proved by go/types instead of a line.
func (r *relations) proved(class string, from, to *codeNode, ev *Evidence) bool {
	if goRealization(class, from, ev) {
		return r.c.g.implements(from.d, to.d)
	}
	for _, o := range r.seen {
		if r.matches(o, class, from, to) && within(ev, o.at) {
			return true
		}
	}
	return false
}

// goRealization reports a Go realization edge without evidence, which
// go/types decides.
func goRealization(class string, from *codeNode, ev *Evidence) bool {
	return class == edgeRealization && from.d.Package != "" && ev == nil
}

// within reports a line inside the evidence, or any line without one.
func within(ev *Evidence, at Location) bool {
	return ev == nil || (ev.Path == at.Path && ev.holds(at.Line))
}

func (r *relations) matches(o observed, class string, from, to *codeNode) bool {
	return (joins(o, from, to) || (class == edgeLink && joins(o, to, from))) && contains(accepted[class], o.kind)
}

func joins(o observed, from, to *codeNode) bool {
	return o.from == from && o.to == to
}

// strays fails each relation between concrete nodes that no drawn edge
// allows, once per pair and kind, at its first source line.
func (r *relations) strays() {
	reported := map[observed]bool{}
	for _, o := range r.seen {
		key := observed{from: o.from, to: o.to, kind: o.kind}
		if reported[key] || r.allowed(o) {
			continue
		}
		reported[key] = true
		r.c.fail(o.at, "%s %s %s, but %s draws no %s -> %s edge that allows it", o.from.e.ID, o.kind, o.to.e.ID, r.c.viewFile(), o.from.e.ID, o.to.e.ID)
	}
}

func (r *relations) allowed(o observed) bool {
	for _, e := range r.c.m.Relationships {
		if e.View == codeView && r.matches(o, arrows[e.Arrow], r.c.nodes[e.From], r.c.nodes[e.To]) {
			return true
		}
	}
	return false
}
