package architecture

import (
	"fmt"
	"strings"
)

const codeView = "code"

// Code is the code view compared with Go and Swift declarations and the
// relations between them. Files counts the code views read.
type Code struct {
	Errors []string
	Files  int
}

// Code resolves each class of the code view to one declaration by its
// linked file, name, kind, and signature, checks its stated members,
// requires a node for each exported type of a folder the view shows, and
// keeps the relations between declarations to the edges the view draws.
func (m *Model) Code() Code {
	c := newCode(m)
	for _, n := range c.order {
		c.resolve(n)
	}
	c.duplicates()
	c.cover()
	c.relate()
	c.Files = m.viewCount(codeView)
	return c.Code
}

// codeNode is one class of the code view. An abstract node names no
// declaration; a concrete one resolves to d.
type codeNode struct {
	e        Element
	abstract bool
	links    []Evidence
	d        *decl
}

type code struct {
	Code
	m     *Model
	x     *declIndex
	g     *goIndex
	nodes map[string]*codeNode
	order []*codeNode
}

func newCode(m *Model) *code {
	x := &declIndex{}
	c := &code{m: m, x: x, g: m.goIndex(x), nodes: map[string]*codeNode{}}
	m.swiftDecls(x)
	c.Errors = append(c.Errors, x.errors...)
	for _, e := range m.Elements {
		c.element(e)
	}
	for _, ev := range m.Evidence {
		if n := c.nodes[ev.Subject]; n != nil && ev.At.Path == n.e.At.Path {
			n.links = append(n.links, ev)
		}
	}
	return c
}

func (c *code) element(e Element) {
	if e.View != codeView || e.Kind != "class" || c.nodes[e.ID] != nil {
		return
	}
	n := &codeNode{e: e}
	c.nodes[e.ID] = n
	c.order = append(c.order, n)
}

func (c *code) fail(at Location, format string, args ...any) {
	c.Errors = append(c.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

// annotations reads <<abstract>>, a node that names no declaration, and
// <<interface>>, which must resolve to a Go interface or Swift protocol.
func (c *code) annotations(n *codeNode) {
	for _, a := range n.e.Annotations {
		switch a.Text {
		case "abstract":
			n.abstract = true
		case kindInterface:
		default:
			c.fail(a.At, "unsupported annotation <<%s>> on %s; use <<abstract>> or <<interface>>", a.Text, n.e.ID)
		}
	}
}

func (c *code) resolve(n *codeNode) {
	c.annotations(n)
	if n.abstract {
		return
	}
	s, ok := parseStated(n.e.Name)
	if !ok {
		c.fail(n.e.At, "%s %q names no declaration; name a Go or Swift symbol, or mark it <<abstract>>", n.e.ID, n.e.Name)
		return
	}
	files := c.sourceLinks(n)
	if len(files) == 0 {
		c.fail(n.e.At, "%s links no Go or Swift source; link the file declaring %s in its callout row, or mark it <<abstract>>", n.e.ID, s.text)
		return
	}
	n.d = c.pick(n, s, files)
	c.check(n)
}

// sourceLinks lists the Go and Swift files a node's row links, failing
// any other link.
func (c *code) sourceLinks(n *codeNode) []string {
	var files []string
	for _, ev := range n.links {
		if !supportedEvidence(ev.Path) {
			c.fail(ev.At, "unsupported evidence %s; Go and Swift sources only", ev)
			continue
		}
		files = append(files, ev.Path)
	}
	return files
}

// pick narrows the declarations by linked file and lines, qualifier, kind,
// and signature, failing when none or several remain.
func (c *code) pick(n *codeNode, s stated, files []string) *decl {
	owner, pkg := c.qualifier(s, files)
	found := c.linked(n, c.candidates(s.name, owner, pkg, files))
	if len(found) == 0 {
		c.missing(n, s, owner, pkg, files)
		return nil
	}
	return c.narrow(n, s, found)
}

// qualifier splits a label's qualifier into a type that owns the member
// and a Go package name.
func (c *code) qualifier(s stated, files []string) (owner, pkg string) {
	qual := s.qual
	if len(qual) > 0 && c.declaresType(qual[len(qual)-1], files) {
		owner, qual = qual[len(qual)-1], qual[:len(qual)-1]
	}
	return owner, strings.Join(qual, ".")
}

func (c *code) declaresType(name string, files []string) bool {
	for _, d := range c.x.named(name, "", files) {
		if d.typeLike() {
			return true
		}
	}
	return false
}

// candidates lists declarations with the name and owner, in files when
// given, whose package matches the qualifier. Extensions declare nothing.
func (c *code) candidates(name, owner, pkg string, files []string) []*decl {
	var found []*decl
	for _, d := range c.x.named(name, owner, files) {
		if d.Kind != kindExtension && (pkg == "" || pkg == d.Package) {
			found = append(found, d)
		}
	}
	return found
}

// linked keeps declarations inside a link's lines; a link without a line
// anchor holds its whole file.
func (c *code) linked(n *codeNode, decls []*decl) []*decl {
	var kept []*decl
	for _, d := range decls {
		if n.holds(d) {
			kept = append(kept, d)
		}
	}
	return kept
}

func (n *codeNode) holds(d *decl) bool {
	for _, ev := range n.links {
		if ev.Path == d.File && ev.holds(d.Line) {
			return true
		}
	}
	return false
}

// missing explains a node whose links hold no declaration: moved
// elsewhere, or gone.
func (c *code) missing(n *codeNode, s stated, owner, pkg string, files []string) {
	if elsewhere := c.outside(n, c.candidates(s.name, owner, pkg, nil)); len(elsewhere) > 0 {
		c.fail(n.e.At, "%s: %s is declared at %s, not in the linked %s", n.e.ID, s.text, wheres(elsewhere), linkList(n))
		return
	}
	c.fail(n.e.At, "%s: %s declares no %s", n.e.ID, linkList(n), s.text)
}

func (c *code) outside(n *codeNode, decls []*decl) []*decl {
	var found []*decl
	for _, d := range decls {
		if !n.holds(d) {
			found = append(found, d)
		}
	}
	return found
}

func linkList(n *codeNode) string {
	links := make([]string, len(n.links))
	for i, ev := range n.links {
		links[i] = ev.String()
	}
	return strings.Join(links, ", ")
}

// narrow keeps the declarations of the stated kind and signature.
func (c *code) narrow(n *codeNode, s stated, found []*decl) *decl {
	kinds := filter(found, s.accepts)
	if len(kinds) == 0 {
		c.fail(n.e.At, "%s: %s at %s is a %s, not a %s", n.e.ID, found[0].full(), found[0].where(), found[0].Kind, s.kind())
		return nil
	}
	if s.call {
		return c.signed(n, s, kinds)
	}
	return c.one(n, kinds)
}

func filter(decls []*decl, keep func(*decl) bool) []*decl {
	var kept []*decl
	for _, d := range decls {
		if keep(d) {
			kept = append(kept, d)
		}
	}
	return kept
}

func (c *code) signed(n *codeNode, s stated, decls []*decl) *decl {
	signed := filter(decls, s.matches)
	if len(signed) == 0 {
		c.fail(n.e.At, "%s: %s at %s has signature %s, not %s", n.e.ID, decls[0].full(), decls[0].where(), decls[0].signature(), s.text)
		return nil
	}
	return c.one(n, signed)
}

// one fails declarations a label cannot tell apart, such as overloads or
// one name under two build conditions.
func (c *code) one(n *codeNode, decls []*decl) *decl {
	if len(decls) > 1 {
		c.fail(n.e.At, "%s: %s is declared %d times, at %s; link one file or line range, or state its signature", n.e.ID, decls[0].full(), len(decls), wheres(decls))
		return nil
	}
	return decls[0]
}

// check fails a node of the wrong annotation or a non-product source,
// then compares its members.
func (c *code) check(n *codeNode) {
	if n.d == nil {
		return
	}
	if n.interfaceNode() && n.d.Kind != kindInterface {
		c.fail(n.e.At, "%s: %s at %s is a %s, not an interface", n.e.ID, n.d.full(), n.d.where(), n.d.Kind)
	}
	if !c.m.product(n.d.File) {
		c.fail(n.e.At, "%s: %s is a %s source; the code view shows product code", n.e.ID, n.d.File, category(n.d.File, c.m.repo.kept[n.d.File]))
	}
	c.members(n)
}

func (n *codeNode) interfaceNode() bool {
	for _, a := range n.e.Annotations {
		if a.Text == kindInterface {
			return true
		}
	}
	return false
}

// duplicates fails two nodes that name one declaration.
func (c *code) duplicates() {
	first := map[*decl]*codeNode{}
	for _, n := range c.order {
		if n.d == nil {
			continue
		}
		if seen := first[n.d]; seen != nil {
			c.fail(n.e.At, "%s and %s at %s both name %s", n.e.ID, seen.e.ID, seen.e.At, n.d.full())
			continue
		}
		first[n.d] = n
	}
}

// cover requires a node for every exported type declared in a product
// file of a folder the view already shows.
func (c *code) cover() {
	folders, named := c.shown()
	for _, d := range c.x.decls {
		if covered(d) && folders[d.folder()] && c.unnamed(d, named) {
			c.fail(d.at(), "%s is an exported type in %s, which %s shows, but no node names it", d.full(), d.folder(), c.viewFile())
		}
	}
}

// shown lists the folders the view shows and the declarations it names.
func (c *code) shown() (map[string]bool, map[*decl]bool) {
	folders, named := map[string]bool{}, map[*decl]bool{}
	for _, n := range c.order {
		if n.d != nil {
			folders[n.d.folder()], named[n.d] = true, true
		}
	}
	return folders, named
}

// unnamed reports a product declaration no node names.
func (c *code) unnamed(d *decl, named map[*decl]bool) bool {
	return !named[d] && c.m.product(d.File)
}

// covered reports a top-level exported type, the kind the view must show.
func covered(d *decl) bool {
	return d.Owner == "" && d.typeLike() && d.Exported
}

func (c *code) viewFile() string {
	for _, file := range c.m.Files {
		if viewFile(file) == codeView {
			return file
		}
	}
	return ""
}
