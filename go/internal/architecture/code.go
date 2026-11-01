package architecture

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
)

const codeView = "code"

// Code is the code view compared with the declarations every language
// module's reader reads and the relations between them. Files counts the
// code views read.
type Code struct {
	Errors []string
	Files  int
}

// Code resolves each code view class to one declaration, checks its stated
// members, and keeps relations to drawn edges.
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
	d        *codemodel.Decl
}

type code struct {
	Code
	m     *Model
	x     *codemodel.Index
	reads map[string]codemodel.Code
	nodes map[string]*codeNode
	order []*codeNode
}

// newCode reads every module's declarations, in registry order, and
// indexes the view's classes.
func newCode(m *Model) *code {
	c := &code{m: m, x: &codemodel.Index{}, reads: map[string]codemodel.Code{}, nodes: map[string]*codeNode{}}
	repo := m.codeRepo()
	for _, r := range viewReaders {
		c.reads[r.Language] = r.Read(repo, c.x)
	}
	c.Errors = append(c.Errors, c.x.Errors...)
	for _, e := range m.Elements {
		c.element(e)
	}
	for _, ev := range m.Evidence {
		c.link(ev)
	}
	return c
}

// link adds evidence on a node's own row to the node.
func (c *code) link(ev Evidence) {
	if n := c.nodes[ev.Subject]; n != nil && ev.At.Path == n.e.At.Path {
		n.links = append(n.links, ev)
	}
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
// <<interface>>, which must resolve to an interface declaration.
func (c *code) annotations(n *codeNode) {
	for _, a := range n.e.Annotations {
		switch a.Text {
		case "abstract":
			n.abstract = true
		case codemodel.KindInterface:
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
		c.fail(n.e.At, "%s %q names no declaration; name a %s symbol, or mark it <<abstract>>", n.e.ID, n.e.Name, codemodel.Series(viewReaders, "or"))
		return
	}
	files := c.sourceLinks(n)
	if len(files) == 0 {
		c.fail(n.e.At, "%s links no %s source; link the file declaring %s in its callout row, or mark it <<abstract>>", n.e.ID, codemodel.Series(viewReaders, "or"), s.text)
		return
	}
	n.d = c.pick(n, s, files)
	c.check(n)
}

// sourceLinks lists the source files a node's row links that a view
// reader reads, failing any other link.
func (c *code) sourceLinks(n *codeNode) []string {
	var files []string
	for _, ev := range n.links {
		if _, ok := codemodel.Of(viewReaders, ev.Path); !ok {
			c.fail(ev.At, "unsupported evidence %s; %s sources only", ev, codemodel.Series(viewReaders, "and"))
			continue
		}
		files = append(files, ev.Path)
	}
	return files
}

// pick narrows the declarations by linked file and lines, qualifier, kind,
// and signature, failing when none or several remain.
func (c *code) pick(n *codeNode, s stated, files []string) *codemodel.Decl {
	owner, pkg := c.qualifier(s)
	found := c.linked(n, c.candidates(s.name, owner, pkg, files))
	if len(found) == 0 {
		c.missing(n, s, owner, pkg, files)
		return nil
	}
	return c.narrow(n, s, found)
}

// qualifier splits a label's qualifier into a type that owns the member
// and a package name. The type may be declared in any file, since a
// method or an extension can sit apart from its type.
func (c *code) qualifier(s stated) (owner, pkg string) {
	qual := s.qual
	if len(qual) > 0 && c.declaresType(qual[len(qual)-1]) {
		owner, qual = qual[len(qual)-1], qual[:len(qual)-1]
	}
	return owner, strings.Join(qual, ".")
}

func (c *code) declaresType(name string) bool {
	for _, d := range c.x.Named(name, "", nil) {
		if d.TypeLike() {
			return true
		}
	}
	return false
}

// candidates lists declarations with the name and owner, in files when
// given, whose package matches the qualifier. Extensions declare nothing.
func (c *code) candidates(name, owner, pkg string, files []string) []*codemodel.Decl {
	var found []*codemodel.Decl
	for _, d := range c.x.Named(name, owner, files) {
		if d.Kind != codemodel.KindExtension && (pkg == "" || pkg == d.Package) {
			found = append(found, d)
		}
	}
	return found
}

// linked keeps declarations inside a link's lines; a link without a line
// anchor holds its whole file.
func (c *code) linked(n *codeNode, decls []*codemodel.Decl) []*codemodel.Decl {
	var kept []*codemodel.Decl
	for _, d := range decls {
		if n.holds(d) {
			kept = append(kept, d)
		}
	}
	return kept
}

func (n *codeNode) holds(d *codemodel.Decl) bool {
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

func (c *code) outside(n *codeNode, decls []*codemodel.Decl) []*codemodel.Decl {
	var found []*codemodel.Decl
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

func wheres(decls []*codemodel.Decl) string {
	places := make([]string, len(decls))
	for i, d := range decls {
		places[i] = d.Where()
	}
	return strings.Join(places, ", ")
}

// narrow keeps the declarations of the stated kind and signature.
func (c *code) narrow(n *codeNode, s stated, found []*codemodel.Decl) *codemodel.Decl {
	kinds := filter(found, s.accepts)
	if len(kinds) == 0 {
		c.fail(n.e.At, "%s: %s at %s is a %s, not a %s", n.e.ID, found[0].Full(), found[0].Where(), found[0].Kind, s.kind())
		return nil
	}
	if s.call {
		return c.signed(n, s, kinds)
	}
	return c.one(n, kinds)
}

func filter(decls []*codemodel.Decl, keep func(*codemodel.Decl) bool) []*codemodel.Decl {
	var kept []*codemodel.Decl
	for _, d := range decls {
		if keep(d) {
			kept = append(kept, d)
		}
	}
	return kept
}

func (c *code) signed(n *codeNode, s stated, decls []*codemodel.Decl) *codemodel.Decl {
	signed := filter(decls, s.matches)
	if len(signed) == 0 {
		c.fail(n.e.At, "%s: %s at %s has signature %s, not %s", n.e.ID, decls[0].Full(), decls[0].Where(), decls[0].Signature(), s.text)
		return nil
	}
	return c.one(n, signed)
}

// one fails declarations a label cannot tell apart, such as overloads or
// one name under two build conditions.
func (c *code) one(n *codeNode, decls []*codemodel.Decl) *codemodel.Decl {
	if len(decls) > 1 {
		c.fail(n.e.At, "%s: %s is declared %d times, at %s; link one file or line range, or state its signature", n.e.ID, decls[0].Full(), len(decls), wheres(decls))
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
	if n.interfaceNode() && n.d.Kind != codemodel.KindInterface {
		c.fail(n.e.At, "%s: %s at %s is a %s, not an interface", n.e.ID, n.d.Full(), n.d.Where(), n.d.Kind)
	}
	if !c.m.product(n.d.File) {
		c.fail(n.e.At, "%s: %s is a %s source; the code view shows product code", n.e.ID, n.d.File, category(n.d.File, c.m.repo.kept[n.d.File], c.m.repo.dev[n.d.File]))
	}
	c.members(n)
}

func (n *codeNode) interfaceNode() bool {
	for _, a := range n.e.Annotations {
		if a.Text == codemodel.KindInterface {
			return true
		}
	}
	return false
}

// duplicates fails two nodes that name one declaration.
func (c *code) duplicates() {
	first := map[*codemodel.Decl]*codeNode{}
	for _, n := range c.order {
		if n.d == nil {
			continue
		}
		if seen := first[n.d]; seen != nil {
			c.fail(n.e.At, "%s and %s at %s both name %s", n.e.ID, seen.e.ID, seen.e.At, n.d.Full())
			continue
		}
		first[n.d] = n
	}
}

// cover requires a node for every exported type declared in a product
// file of a folder the view already shows.
func (c *code) cover() {
	folders, named := c.shown()
	for _, d := range c.x.Decls {
		if covered(d) && folders[d.Folder()] && c.unnamed(d, named) {
			c.fail(declAt(d), "%s is an exported type in %s, which %s shows, but no node names it", d.Full(), d.Folder(), c.viewFile())
		}
	}
}

// shown lists the folders the view shows and the declarations it names.
func (c *code) shown() (map[string]bool, map[*codemodel.Decl]bool) {
	folders, named := map[string]bool{}, map[*codemodel.Decl]bool{}
	for _, n := range c.order {
		if n.d != nil {
			folders[n.d.Folder()], named[n.d] = true, true
		}
	}
	return folders, named
}

// unnamed reports a product declaration no node names.
func (c *code) unnamed(d *codemodel.Decl, named map[*codemodel.Decl]bool) bool {
	return !named[d] && c.m.product(d.File)
}

// covered reports a top-level exported type, the kind the view must show.
func covered(d *codemodel.Decl) bool {
	return d.Owner == "" && d.TypeLike() && d.Exported
}

func (c *code) viewFile() string {
	for _, file := range c.m.Files {
		if viewFile(file) == codeView {
			return file
		}
	}
	return ""
}
