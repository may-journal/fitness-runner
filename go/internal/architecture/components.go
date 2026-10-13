package architecture

import (
	"fmt"
	"path"
	"strings"
)

const componentsView = "components"

// Components is the components view compared with the product sources and
// the references between them. Files counts the component views read.
// Note counts the Swift names the parser could not resolve; it fails
// nothing.
type Components struct {
	Errors []string
	Files  int
	Note   string
}

// Components gives every product source one owner among the view's
// components, requires every reference between owners to be drawn, proves
// the evidence edge rows link, and keeps each runtime to the components
// under its own container boundary.
func (m *Model) Components() Components {
	c := newComponents(m)
	for _, s := range m.Sources {
		c.own(s)
	}
	c.observe()
	c.prove()
	c.share()
	c.Files = m.viewCount(componentsView)
	return c.Components
}

type components struct {
	Components
	m          *Model
	view       string
	elements   map[string]Element
	containers map[string]bool
	owners     map[string]string
	refs       []reference
	edges      map[[2]string]reference
	order      [][2]string
}

func newComponents(m *Model) *components {
	c := &components{m: m, elements: map[string]Element{}, containers: map[string]bool{}, owners: map[string]string{}, edges: map[[2]string]reference{}}
	for _, e := range m.Elements {
		c.element(e)
	}
	for _, file := range m.Files {
		if viewFile(file) == componentsView && c.view == "" {
			c.view = file
		}
	}
	return c
}

func (c *components) element(e Element) {
	if e.View == componentsView {
		keepFirst(c.elements, e.ID, e)
	}
	if e.View == "containers" && internal(e.Kind, "Container") {
		c.containers[e.ID] = true
	}
}

func (c *components) fail(at Location, format string, args ...any) {
	c.Errors = append(c.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

// unit reports an element that can own sources: a component of the view,
// not a boundary, person, or external system.
func (c *components) unit(id string) bool {
	e, ok := c.elements[id]
	return ok && !boundary(e.Kind) && !external(e.Kind) && !person(e.Kind)
}

func boundary(kind string) bool {
	return strings.HasSuffix(kind, "Boundary") || kind == "subgraph" || kind == "namespace"
}

func external(kind string) bool {
	return strings.HasSuffix(kind, "_Ext")
}

// own records a product source's one owner, or fails a source no
// component names or two components name.
func (c *components) own(s Source) {
	if s.Category != "product" {
		return
	}
	var units []string
	for _, owner := range s.Owners {
		if c.unit(owner) {
			units = append(units, owner)
		}
	}
	c.owner(s.Path, units)
}

func (c *components) owner(file string, units []string) {
	switch len(units) {
	case 0:
		c.fail(Location{file, 1}, "no component in the components view names this source")
	case 1:
		c.owners[file] = units[0]
	default:
		c.fail(Location{file, 1}, "components %s at %s and %s at %s both name this source", units[0], c.claimAt(units[0], file), units[1], c.claimAt(units[1], file))
	}
}

func (c *components) claimAt(owner, file string) Location {
	for _, claim := range c.m.Claims {
		if claim.Owner == owner && covers(claim.Path, file) {
			return claim.At
		}
	}
	return Location{}
}

// pathOwners lists the owners of a file, or of the files in a folder.
func (c *components) pathOwners(p string) []string {
	if owner, ok := c.owners[p]; ok {
		return []string{owner}
	}
	var owners []string
	for _, file := range sortedKeys(c.owners) {
		if path.Dir(file) == p && !contains(owners, c.owners[file]) {
			owners = append(owners, c.owners[file])
		}
	}
	return owners
}

// observe reads every Go and Swift reference and fails each pair of
// components the view does not draw an edge between.
func (c *components) observe() {
	refs, errs := c.m.goReferences()
	swift, missed := c.m.swiftReferences()
	c.refs, c.Note = append(refs, swift...), missed.note()
	c.Errors = append(c.Errors, errs...)
	for _, r := range c.refs {
		c.reference(r)
	}
}

func (c *components) reference(r reference) {
	from, ok := c.owners[r.At.Path]
	if !ok {
		return
	}
	for _, to := range c.pathOwners(r.To) {
		if to != from {
			c.edge(from, to, r)
		}
	}
}

func (c *components) edge(from, to string, r reference) {
	key := [2]string{from, to}
	if _, seen := c.edges[key]; seen {
		return
	}
	c.edges[key] = r
	c.order = append(c.order, key)
	if c.relation(from, to) == nil {
		c.fail(r.At, "%s %s%s, owned by %s, but %s draws no %s -> %s edge%s", from, r.what(), r.under(), to, c.view, from, to, c.reversed(from, to))
	}
}

// relation finds a drawn edge from an element enclosing from to one
// enclosing to; an edge between boundaries allows every edge between their
// members.
func (c *components) relation(from, to string) *Relationship {
	for i := range c.m.Relationships {
		r := &c.m.Relationships[i]
		if r.View == componentsView && c.within(from, r.From) && c.within(to, r.To) {
			return r
		}
	}
	return nil
}

func (c *components) reversed(from, to string) string {
	if r := c.relation(to, from); r != nil {
		return fmt.Sprintf("; %s draws %s -> %s", r.At, r.From, r.To)
	}
	return ""
}

// within reports whether id is outer or sits inside it.
func (c *components) within(id, outer string) bool {
	for ; id != ""; id = c.elements[id].Parent {
		if id == outer {
			return true
		}
	}
	return false
}

func (m *Model) viewCount(view string) int {
	count := 0
	for _, file := range m.Files {
		if viewFile(file) == view {
			count++
		}
	}
	return count
}
