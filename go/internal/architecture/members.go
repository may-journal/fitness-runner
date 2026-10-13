package architecture

// members compares each member a class body states with the fields and
// methods its type declares, extensions included.
func (c *code) members(n *codeNode) {
	if len(n.e.Members) == 0 {
		return
	}
	if !n.d.typeLike() {
		c.fail(n.e.Members[0].At, "%s: %s is a %s, which has no members", n.e.ID, n.d.full(), n.d.Kind)
		return
	}
	all := c.x.members(n.d)
	for _, mem := range n.e.Members {
		c.member(n, mem, all)
	}
}

func (c *code) member(n *codeNode, mem Member, all []*decl) {
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

func (c *code) method(n *codeNode, mem Member, s stated, all []*decl) {
	found := filter(all, func(d *decl) bool { return d.Name == s.name && d.Kind == kindMethod })
	if len(found) == 0 {
		c.fail(mem.At, "%s: %s at %s declares no method %s", n.e.ID, n.d.full(), n.d.where(), s.name)
		return
	}
	if len(filter(found, s.matches)) == 0 {
		c.fail(mem.At, "%s: method %s at %s has signature %s, not %s", n.e.ID, found[0].full(), found[0].where(), found[0].signature(), s.text)
	}
}

func (c *code) field(n *codeNode, mem Member, name, typ string, all []*decl) {
	found := filter(all, func(d *decl) bool { return d.Name == name && d.Kind == kindField })
	switch {
	case len(found) == 0:
		c.fail(mem.At, "%s: %s at %s declares no field %s", n.e.ID, n.d.full(), n.d.where(), name)
	case typ != "" && typeText(found[0].Type) != typeText(typ):
		c.fail(mem.At, "%s: field %s at %s is %s, not %s", n.e.ID, found[0].full(), found[0].where(), orNone(found[0].Type), typ)
	}
}

func orNone(typ string) string {
	if typ == "" {
		return "of no stated type"
	}
	return typ
}
