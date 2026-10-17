package views

import "strings"

// swiftLiterals gives the literal keywords their type.
var swiftLiterals = map[string]string{"true": "Bool", "false": "Bool"}

// expr reads one piece of an expression: a postfix chain, an implicit
// member, a closure or block, or a group. Anything else, such as an
// operator or a keyword, is passed over.
func (p *swiftParser) expr(s *swiftScope) {
	switch {
	case p.primary():
		p.chain(s)
	case p.is("."):
		p.implicit(s)
	case p.accept("{"):
		p.condition = false
		p.closure(s)
	case !p.group(s):
		p.i++
	}
}

// group reads a parenthesized group, or an array or dictionary.
func (p *swiftParser) group(s *swiftScope) bool {
	switch {
	case p.accept("("):
		p.args(s, ")")
	case p.accept("["):
		p.elements(s)
	default:
		return false
	}
	return true
}

// elements reads up to and past a closing bracket. A dictionary key is a
// name like any other, not a label.
func (p *swiftParser) elements(s *swiftScope) {
	saved := p.condition
	p.condition = false
	p.block(s, "]")
	p.condition = saved
}

// primary reports a token that starts a postfix chain.
func (p *swiftParser) primary() bool {
	t := p.peek()
	return isWord(t) || t.kind == swiftString || t.kind == swiftNumber || swiftSelf[t.text] || swiftLiterals[t.text] != ""
}

// chain reads a name or literal and every member access, call,
// subscript, and trailing closure after it, returning the last link.
func (p *swiftParser) chain(s *swiftScope) *swiftUse {
	u := p.atom(s)
	for p.postfix(s, &u) {
	}
	return u
}

func (p *swiftParser) atom(s *swiftScope) *swiftUse {
	t := p.next()
	if literal := literalType(t); literal != "" {
		return &swiftUse{literal: literal}
	}
	return p.use(s, t)
}

func literalType(t swiftToken) string {
	switch {
	case t.kind == swiftString:
		return "String"
	case t.kind == swiftNumber && strings.Contains(t.text, "."):
		return "Double"
	case t.kind == swiftNumber:
		return "Int"
	}
	return swiftLiterals[t.text]
}

func (p *swiftParser) postfix(s *swiftScope, u **swiftUse) bool {
	return p.access(s, u) || p.apply(s, u)
}

// access reads optional chaining and member access after a dot; a member
// may spell a keyword, as .init does.
func (p *swiftParser) access(s *swiftScope, u **swiftUse) bool {
	if p.unwraps() {
		return true
	}
	if !p.is(".") || !isAnyWord(p.peekAt(1)) {
		return false
	}
	p.i++
	m := p.use(s, p.next())
	m.member, m.base = true, *u
	*u = m
	return true
}

// unwraps reads the ? or ! of a?.b or a!.b; elsewhere they are operators.
func (p *swiftParser) unwraps() bool {
	if (p.is("?") || p.is("!")) && p.peekAt(1).text == "." {
		p.i++
		return true
	}
	return false
}

// apply reads a call's arguments, a subscript, whose type is unknown, or
// a trailing closure, which a condition's body brace is not.
func (p *swiftParser) apply(s *swiftScope, u **swiftUse) bool {
	switch {
	case p.accept("("):
		(*u).called(p.args(s, ")"))
	case p.accept("["):
		p.elements(s)
		*u = &swiftUse{}
	case p.trails(*u):
		p.i++
		(*u).call, (*u).trailing = true, (*u).trailing+1
		p.closure(s)
	default:
		return false
	}
	return true
}

// trails reports a trailing closure's brace after a name.
func (p *swiftParser) trails(u *swiftUse) bool {
	return p.is("{") && !p.condition && u.literal == ""
}

// args reads a call's arguments up to and past close, returning their
// labels, "_" where an argument has none. A brace inside them opens a
// closure even within a condition, which resumes after them.
func (p *swiftParser) args(s *swiftScope, close string) []string {
	saved := p.condition
	p.condition = false
	labels := []string{}
	for p.inside(close) {
		labels = append(labels, p.label())
		for p.inside(close) && !p.is(",") {
			p.item(s)
		}
		p.accept(",")
	}
	p.accept(close)
	p.condition = saved
	return labels
}

// label reads an argument label: a word, keywords included, before a colon.
func (p *swiftParser) label() string {
	t := p.peek()
	if isAnyWord(t) && p.peekAt(1).text == ":" && p.peekAt(1).kind == swiftPunct {
		p.i += 2
		return t.text
	}
	return "_"
}

// implicit reads .name, whose type the context decides, and its chain.
func (p *swiftParser) implicit(s *swiftScope) {
	p.i++
	if !isAnyWord(p.peek()) {
		return
	}
	u := p.use(s, p.next())
	u.implicit = true
	for p.postfix(s, &u) {
	}
}

// closure reads a closure or block body after its brace: the closure's
// parameters and capture list before in, then its items.
func (p *swiftParser) closure(s *swiftScope) {
	saved := p.condition
	p.condition = false
	body := s.child(s.owner, false)
	if p.closureHead() {
		p.closureParams(body)
	}
	p.block(body, "}")
	p.condition = saved
}

// closureHead reports an in ahead that only parameter tokens precede.
func (p *swiftParser) closureHead() bool {
	for j := p.i; j < len(p.tokens); j++ {
		t := p.tokens[j]
		if t.kind == swiftName && t.text == "in" {
			return true
		}
		if !headToken(t) {
			return false
		}
	}
	return false
}

func headToken(t swiftToken) bool {
	return isWord(t) || swiftHead[t.text] || swiftModifiers[t.text]
}

func (p *swiftParser) closureParams(body *swiftScope) {
	for p.more() && !p.is("in") {
		p.closureParam(body)
	}
	p.accept("in")
}

// closureParam binds a parameter or capture, or reads a stated type.
func (p *swiftParser) closureParam(body *swiftScope) {
	switch {
	case p.accept(":") || p.accept("->"):
		p.typeRef(body)
	case isWord(p.peek()):
		p.bind(body, p.next().text, "")
	default:
		p.i++
	}
}
