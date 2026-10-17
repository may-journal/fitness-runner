package views

// typeDecl reads a class, struct, enum, protocol, or actor: its name,
// generic parameters, inheritance, and body. Class before func or var is
// a modifier.
func (p *swiftParser) typeDecl(s *swiftScope) {
	next := p.peekAt(1)
	if swiftAfterClass[next.text] || !isAnyWord(next) {
		p.modifier(s)
		return
	}
	p.i++
	sym := p.declare(s, &swiftSymbol{name: p.next().text, kind: swiftType})
	body := s.child(qualify(s.owner, sym.name), true)
	p.generics(body)
	p.m.declareType(body.owner, p.inherits(body))
	p.where(body)
	p.body(body)
}

// extension reads an extension: its members belong to the extended type
// and the protocols it adopts join that type's inheritance.
func (p *swiftParser) extension(s *swiftScope) {
	p.i++
	body := s.child(p.typeRef(s), true)
	p.m.conform(body.owner, p.inherits(body))
	p.where(body)
	p.body(body)
}

// funcDecl reads a function, operator, or subscript: its labels, defaults,
// result type, and body, whose parameters are its first locals.
func (p *swiftParser) funcDecl(s *swiftScope) {
	name := p.next().text
	if name == "func" {
		name = p.funcName()
	}
	fn := s.child(s.owner, false)
	p.generics(fn)
	sym := p.callable(s, fn, &swiftSymbol{name: name, kind: swiftFunc})
	sym.typ = p.result(fn)
	p.where(fn)
	p.body(fn)
}

// funcName reads a function's name, or an operator's punctuation up to
// its parameters, so < and <= are names, not generic brackets.
func (p *swiftParser) funcName() string {
	if name := p.word(); name != "" {
		return name
	}
	name := ""
	for p.peek().kind == swiftPunct && !p.is("(") {
		name += p.next().text
	}
	return name
}

func (p *swiftParser) callable(s, fn *swiftScope, sym *swiftSymbol) *swiftSymbol {
	sym.labels, sym.defaults = p.params(fn)
	return p.declare(s, sym)
}

// initDecl reads an initializer, whose value is its type.
func (p *swiftParser) initDecl(s *swiftScope) {
	p.i++
	if !p.accept("?") {
		p.accept("!")
	}
	fn := s.child(s.owner, false)
	p.generics(fn)
	p.callable(s, fn, &swiftSymbol{name: "init", kind: swiftInit, typ: s.owner})
	p.result(fn)
	p.where(fn)
	p.body(fn)
}

func (p *swiftParser) deinit(s *swiftScope) {
	p.i++
	p.body(s.child(s.owner, false))
}

// body reads a declaration's braces, if it has them.
func (p *swiftParser) body(s *swiftScope) {
	if p.accept("{") {
		p.block(s, "}")
	}
}

// binding reads let or var: the names it binds, their stated type, and
// the start of their initializer. A property's braces hold accessors; in
// a body, braces after a name open the if let body that follows.
func (p *swiftParser) binding(s *swiftScope) {
	p.i++
	names := p.pattern()
	typ := ""
	if p.accept(":") {
		typ = p.typeRef(s)
	}
	init := p.initializer(s)
	for _, name := range names {
		p.declare(s, &swiftSymbol{name: name, kind: swiftValue, typ: typ, init: init})
	}
	if !s.local {
		p.body(s.child(s.owner, false))
	}
}

// initializer reads the chain an initializer starts with, whose last link
// gives the value its type.
func (p *swiftParser) initializer(s *swiftScope) *swiftUse {
	if p.accept("=") && p.primary() {
		return p.chain(s)
	}
	return nil
}

// pattern reads a name, or the names of a tuple pattern.
func (p *swiftParser) pattern() []string {
	if name := p.word(); name != "" {
		return []string{name}
	}
	var names []string
	if p.accept("(") {
		for p.inside(")") {
			names = appendWord(names, p.next())
		}
		p.accept(")")
	}
	return names
}

func appendWord(names []string, t swiftToken) []string {
	if isWord(t) {
		return append(names, t.text)
	}
	return names
}

// cases reads an enum's cases, whose payload labels a call must fit. In a
// body, case starts a pattern the expression reader takes.
func (p *swiftParser) cases(s *swiftScope) {
	p.i++
	for !s.local {
		name := p.word()
		p.callable(s, s.child(s.owner, false), &swiftSymbol{name: name, kind: swiftCase, typ: s.owner})
		if p.accept("=") {
			p.i++
		}
		if name == "" || !p.accept(",") {
			return
		}
	}
}

func (p *swiftParser) alias(s *swiftScope) {
	p.i++
	sym := &swiftSymbol{name: p.word(), kind: swiftType}
	p.generics(s.child(s.owner, false))
	if p.accept("=") {
		sym.typ = p.typeRef(s)
	}
	p.declare(s, sym)
}

// associated binds a protocol's associated type, which names no symbol
// of the module.
func (p *swiftParser) associated(s *swiftScope) {
	p.i++
	p.bind(s, p.word(), "")
	p.inherits(s)
	if p.accept("=") {
		p.typeRef(s)
	}
}

// generics binds generic parameters and reads their constraints.
func (p *swiftParser) generics(s *swiftScope) {
	if !p.accept("<") {
		return
	}
	for p.inside(">") {
		start := p.i
		p.bind(s, p.word(), "")
		p.constraint(s)
		p.progress(start)
	}
	p.accept(">")
}

func (p *swiftParser) constraint(s *swiftScope) {
	if p.accept(":") {
		p.typeRef(s)
	}
	p.accept(",")
}

// progress moves on when a loop's step read nothing.
func (p *swiftParser) progress(start int) {
	if p.i == start {
		p.i++
	}
}

// inherits reads an inheritance clause: ": Base, Codable".
func (p *swiftParser) inherits(s *swiftScope) []string {
	var supers []string
	if !p.accept(":") {
		return nil
	}
	for {
		supers = append(supers, p.typeRef(s))
		if !p.accept(",") {
			return supers
		}
	}
}

// where reads a where clause to the end of its line or the body's brace.
func (p *swiftParser) where(s *swiftScope) {
	if !p.is("where") {
		return
	}
	line := p.next().line
	for p.inside("{") && p.peek().line == line {
		start := p.i
		p.typeRef(s)
		p.progress(start)
	}
}

// params reads a parameter clause into the function's scope.
func (p *swiftParser) params(fn *swiftScope) ([]string, []bool) {
	if !p.accept("(") {
		return nil, nil
	}
	labels, defaults := []string{}, []bool{}
	for p.inside(")") {
		label, def := p.param(fn)
		labels, defaults = append(labels, label), append(defaults, def)
		p.accept(",")
	}
	p.accept(")")
	return labels, defaults
}

// param reads one parameter, or an unlabeled payload type, reporting its
// label and whether it has a default.
func (p *swiftParser) param(fn *swiftScope) (string, bool) {
	start := p.i
	p.attributes(fn)
	label, name, ok := p.paramNames()
	if !ok {
		p.typeRef(fn)
		p.progress(start)
		return "_", false
	}
	p.bind(fn, name, p.typeRef(fn))
	if !p.accept("=") {
		return label, false
	}
	for p.inside(")") && !p.is(",") {
		p.item(fn)
	}
	return label, true
}

// attributes reads the attributes before a parameter, such as
// @ViewBuilder, recording each as a use.
func (p *swiftParser) attributes(fn *swiftScope) {
	for p.is("@") && isAnyWord(p.peekAt(1)) {
		p.i++
		u := p.use(fn, p.next())
		p.apply(fn, &u)
	}
}

// paramNames reads "label name:" or "name:".
func (p *swiftParser) paramNames() (string, string, bool) {
	words := 0
	for words < 2 && isAnyWord(p.peekAt(words)) {
		words++
	}
	if words == 0 || p.peekAt(words).text != ":" {
		return "", "", false
	}
	label, name := p.peekAt(0).text, p.peekAt(words-1).text
	p.i += words + 1
	return label, name, true
}

// result reads effects and the type after ->.
func (p *swiftParser) result(fn *swiftScope) string {
	for p.accept("async") || p.accept("throws") || p.accept("rethrows") {
		p.thrown(fn)
	}
	if p.accept("->") {
		return p.typeRef(fn)
	}
	return ""
}

// thrown reads a typed throws clause: throws(Failure).
func (p *swiftParser) thrown(fn *swiftScope) {
	if p.accept("(") {
		p.typeRef(fn)
		p.accept(")")
	}
}

// typeRef reads a type, records its names as uses, and returns the name it
// states, such as Store or Outer.Inner; "" for a tuple, array, or function
// type.
func (p *swiftParser) typeRef(s *swiftScope) string {
	r := &typeReader{p: p, s: s, line: p.peek().line}
	for p.more() && r.continues(p.peek()) {
		r.step(p.next())
	}
	return r.name
}

// typeReader reads one type's tokens; a type ends at its own level on a
// new line or at punctuation no type holds.
type typeReader struct {
	p                        *swiftParser
	s                        *swiftScope
	line, depth              int
	name                     string
	prev                     *swiftUse
	dotted, ended, attribute bool
}

func (r *typeReader) continues(t swiftToken) bool {
	if r.depth > 0 {
		return true
	}
	return t.line == r.line && !(swiftTypeEnds[t.text] && t.kind != swiftQuoted)
}

func (r *typeReader) step(t swiftToken) {
	r.depth += swiftDepth[t.text]
	if isWord(t) || swiftSelf[t.text] {
		r.use(t)
	}
	r.dotted, r.attribute = t.text == ".", t.text == "@"
}

// naming reports a name at the type's own level that can still extend
// the stated name; an attribute such as @escaping states none.
func (r *typeReader) naming() bool {
	return r.depth == 0 && !r.ended && !r.attribute
}

var swiftDepth = map[string]int{"(": 1, "[": 1, "<": 1, ")": -1, "]": -1, ">": -1}

// use records a name in the type; at the type's own level it extends the
// stated name, as Outer then Outer.Inner.
func (r *typeReader) use(t swiftToken) {
	u := r.p.use(r.s, t)
	if r.dotted {
		u.member, u.base = true, r.prev
	}
	r.prev = u
	switch {
	case !r.naming():
	case r.dotted:
		r.name += "." + t.text
	case r.name == "":
		r.name = t.text
	default:
		r.ended = true
	}
}
