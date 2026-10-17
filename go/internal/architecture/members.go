package architecture

import "github.com/may-journal/fitness-runner/go/internal/codemodel"

// members compares each member a class body states with the fields and
// methods its type declares, extensions included.
func (c *code) members(n *codeNode) {
	if len(n.e.Members) == 0 {
		return
	}
	if !n.d.TypeLike() {
		c.fail(n.e.Members[0].At, "%s: %s is a %s, which has no members", n.e.ID, n.d.Full(), n.d.Kind)
		return
	}
	all := c.x.Members(n.d)
	for _, mem := range n.e.Members {
		c.member(n, mem, all)
	}
}

func (c *code) member(n *codeNode, mem Member, all []*codemodel.Decl) {
	s, typ, ok := statedMember(mem.Text)
	switch {
	case !ok:
		c.fail(mem.At, "%s: member %q names no field or method", n.e.ID, mem.Text)
	case s.call:
		c.method(n, mem, s, all)
	default:
		c.field(n, mem, s.name, typ, all)
	}
}

func (c *code) method(n *codeNode, mem Member, s stated, all []*codemodel.Decl) {
	found := filter(all, func(d *codemodel.Decl) bool { return d.Name == s.name && d.Kind == codemodel.KindMethod })
	if len(found) == 0 {
		c.fail(mem.At, "%s: %s at %s declares no method %s", n.e.ID, n.d.Full(), n.d.Where(), s.name)
		return
	}
	if len(filter(found, s.matches)) == 0 {
		c.fail(mem.At, "%s: method %s at %s has signature %s, not %s", n.e.ID, found[0].Full(), found[0].Where(), found[0].Signature(), s.text)
	}
}

func (c *code) field(n *codeNode, mem Member, name, typ string, all []*codemodel.Decl) {
	found := filter(all, func(d *codemodel.Decl) bool { return d.Name == name && d.Kind == codemodel.KindField })
	switch {
	case len(found) == 0:
		c.fail(mem.At, "%s: %s at %s declares no field %s", n.e.ID, n.d.Full(), n.d.Where(), name)
	case typ != "" && codemodel.TypeText(found[0].Type) != codemodel.TypeText(typ):
		c.fail(mem.At, "%s: field %s at %s is %s, not %s", n.e.ID, found[0].Full(), found[0].Where(), orNone(found[0].Type), typ)
	}
}

func orNone(typ string) string {
	if typ == "" {
		return "of no stated type"
	}
	return typ
}
