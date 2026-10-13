package architecture

import "strings"

// Kinds of Swift symbol. A type alias is a type whose typ names its target.
const (
	swiftType  = "type"
	swiftFunc  = "func"
	swiftInit  = "init"
	swiftValue = "value"
	swiftCase  = "case"
)

// cspell:ignore associatedtype deinit fileprivate precedencegroup rethrows typealias fallthrough nonmutating nonisolated

// swiftKeywords are the words that never name a declaration.
var swiftKeywords = setOf("associatedtype", "class", "deinit", "enum", "extension", "fileprivate", "func", "import", "init", "inout", "internal", "let", "operator", "private", "precedencegroup", "protocol", "public", "rethrows", "static", "struct", "subscript", "typealias", "var",
	"break", "case", "catch", "continue", "default", "defer", "do", "else", "fallthrough", "for", "guard", "if", "in", "repeat", "return", "throw", "switch", "where", "while",
	"as", "is", "nil", "true", "false", "self", "Self", "super", "throws", "try", "await", "async", "some", "any", "get", "set", "willSet", "didSet")

// swiftModifiers are contextual words that modify the declaration after
// them and name nothing themselves.
var swiftModifiers = setOf("open", "optional", "package", "lazy", "weak", "unowned", "required", "convenience", "dynamic", "indirect", "nonisolated", "isolated", "consuming", "borrowing", "prefix", "postfix", "infix", "mutating", "nonmutating", "override", "final", "public", "private", "fileprivate", "internal", "static")

// swiftAfterClass are the words that make class a modifier, as in class func.
var swiftAfterClass = setOf("func", "var", "let", "subscript", "class", "static", "final", "override", "init")

// swiftSelf are the names of the enclosing type and its instance.
var swiftSelf = setOf("self", "Self", "super")

// swiftTypeEnds end a type at its own nesting level.
var swiftTypeEnds = setOf("{", "}", "=", ",", ")", "]", ">", ":", ";", "where", "in")

// swiftHead are the punctuation a closure's parameters and capture list
// may hold before in.
var swiftHead = setOf("(", ")", "[", "]", ",", ":", ".", "?", "!", "<", ">", "&", "=", "@", "->", "self", "Self", "async", "throws", "inout", "some", "any")

func setOf(words ...string) map[string]bool {
	set := map[string]bool{}
	for _, w := range words {
		set[w] = true
	}
	return set
}

// swiftSymbol is one Swift declaration: a type, function, initializer,
// value, or enum case, at file level or a member of owner. Labels are a
// callable's argument labels, "_" where it has none. Typ is a value's
// stated type, a function's result, or a type alias's target; a value
// with none takes the type of its initializer. A local symbol lives in a
// body and is seen only by the names after it. A static member, like an
// enum case, can be an implicit member such as .shared.
type swiftSymbol struct {
	file, name, owner, kind, typ string
	labels                       []string
	defaults                     []bool
	init                         *swiftUse
	scope                        *swiftScope
	at                           int
	local, static                bool
}

// callable reports a symbol a call's argument labels must fit.
func (s *swiftSymbol) callable() bool {
	return s.kind == swiftFunc || s.kind == swiftInit || s.kind == swiftCase && s.labels != nil
}

// display names the symbol as Swift documentation does: Store,
// Store.save(all:), or load(from:).
func (s *swiftSymbol) display() string {
	name := s.name
	if s.callable() {
		name += "(" + strings.Join(s.labels, ":") + strings.Repeat(":", min(len(s.labels), 1)) + ")"
	}
	return qualify(s.owner, name)
}

func qualify(owner, name string) string {
	if owner == "" {
		return name
	}
	return owner + "." + name
}

// swiftScope is a file, a type or extension body, or a function, closure,
// or block body. Owner is the type the code inside belongs to.
type swiftScope struct {
	parent   *swiftScope
	owner    string
	typeBody bool
	local    bool
	names    map[string][]*swiftSymbol
}

// child opens a scope inside s. Everything inside a function, closure, or
// block body is local, nested types included.
func (s *swiftScope) child(owner string, typeBody bool) *swiftScope {
	return &swiftScope{parent: s, owner: owner, typeBody: typeBody, local: s.local || !typeBody, names: map[string][]*swiftSymbol{}}
}

func (s *swiftScope) add(sym *swiftSymbol) {
	sym.local = true
	s.names[sym.name] = append(s.names[sym.name], sym)
}

// visible finds the latest local named as u is that u comes after.
func (s *swiftScope) visible(u *swiftUse) *swiftSymbol {
	found := s.names[u.name]
	for i := len(found) - 1; i >= 0; i-- {
		if found[i].at <= u.at {
			return found[i]
		}
	}
	return nil
}

// swiftUse is one name in Swift code: unqualified, a member after a dot
// with the expression before it as base, or an implicit member such as
// .loaded whose type the context decides. A call records its argument
// labels and trailing closures. A literal is no name; it gives its base
// the literal's type.
type swiftUse struct {
	file, name, under, literal string
	line, at, trailing         int
	scope                      *swiftScope
	base                       *swiftUse
	member, implicit, call     bool
	labels                     []string
}

func (u *swiftUse) called(labels []string) {
	u.call, u.labels = true, labels
}

// swiftParser reads one file's tokens into the module: declarations, the
// scopes they sit in, and every name the code uses. Condition marks an
// if, guard, while, for, switch, or catch head, where a brace opens the
// body, not a trailing closure. Static marks a static or class modifier
// waiting for its declaration.
type swiftParser struct {
	m         *swiftModule
	file      string
	tokens    []swiftToken
	i         int
	condition bool
	static    bool
}

var swiftItems map[string]func(*swiftParser, *swiftScope)

func init() {
	swiftItems = map[string]func(*swiftParser, *swiftScope){
		"class": (*swiftParser).typeDecl, "struct": (*swiftParser).typeDecl, "enum": (*swiftParser).typeDecl,
		"protocol": (*swiftParser).typeDecl, "actor": (*swiftParser).typeDecl,
		"extension": (*swiftParser).extension, "func": (*swiftParser).funcDecl, "subscript": (*swiftParser).funcDecl,
		"init": (*swiftParser).initDecl, "deinit": (*swiftParser).deinit,
		"var": (*swiftParser).binding, "let": (*swiftParser).binding, "case": (*swiftParser).cases,
		"typealias": (*swiftParser).alias, "associatedtype": (*swiftParser).associated,
		"import": (*swiftParser).skipLine, "operator": (*swiftParser).skipLine, "precedencegroup": (*swiftParser).skipLine,
		"if": (*swiftParser).head, "guard": (*swiftParser).head, "while": (*swiftParser).head, "switch": (*swiftParser).head,
		"for": (*swiftParser).forHead, "catch": (*swiftParser).catchHead,
	}
	for word := range swiftModifiers {
		swiftItems[word] = (*swiftParser).modifier
	}
}

func isAnyWord(t swiftToken) bool { return t.kind == swiftName || t.kind == swiftQuoted }

// isWord reports a name that is no keyword.
func isWord(t swiftToken) bool {
	return isAnyWord(t) && !(t.kind == swiftName && swiftKeywords[t.text])
}

func (p *swiftParser) more() bool { return p.i < len(p.tokens) }

func (p *swiftParser) peekAt(n int) swiftToken {
	if p.i+n < len(p.tokens) {
		return p.tokens[p.i+n]
	}
	return swiftToken{}
}

func (p *swiftParser) peek() swiftToken { return p.peekAt(0) }

func (p *swiftParser) next() swiftToken {
	t := p.peek()
	p.i++
	return t
}

// is reports punctuation or a keyword spelled text.
func (p *swiftParser) is(text string) bool {
	t := p.peek()
	return t.text == text && (t.kind == swiftPunct || t.kind == swiftName)
}

func (p *swiftParser) accept(text string) bool {
	if p.is(text) {
		p.i++
		return true
	}
	return false
}

// inside reports a token before close; a stray } also ends a parenthesis
// or bracket, so one unbalanced line cannot swallow the file.
func (p *swiftParser) inside(close string) bool {
	return p.more() && !p.is(close) && !(close != "}" && p.is("}"))
}

// word reads a name, or returns "" and reads nothing.
func (p *swiftParser) word() string {
	if isWord(p.peek()) {
		return p.next().text
	}
	return ""
}

// block reads items up to and past close; "" reads to the end.
func (p *swiftParser) block(s *swiftScope, close string) {
	for p.inside(close) {
		p.item(s)
	}
	p.accept(close)
}

// item reads one declaration, statement head, or expression piece.
func (p *swiftParser) item(s *swiftScope) {
	t := p.peek()
	if h := swiftItems[t.text]; h != nil && t.kind == swiftName {
		h(p, s)
		return
	}
	p.expr(s)
}

// declare records sym in s: locally inside a body, else in the module.
// A local value is seen from here on; functions and types from anywhere.
func (p *swiftParser) declare(s *swiftScope, sym *swiftSymbol) *swiftSymbol {
	sym.file, sym.owner, sym.scope, sym.at = p.file, s.owner, s, -1
	sym.static, p.static = p.static, false
	if sym.kind == swiftValue {
		sym.at = p.i
	}
	if s.local {
		s.add(sym)
		return sym
	}
	p.m.add(sym)
	return sym
}

// bind declares a local name, such as a parameter, with its stated type.
func (p *swiftParser) bind(s *swiftScope, name, typ string) {
	if name != "" {
		s.add(&swiftSymbol{file: p.file, name: name, kind: swiftValue, typ: typ, scope: s, at: p.i})
	}
}

// use records a name the code uses.
func (p *swiftParser) use(s *swiftScope, t swiftToken) *swiftUse {
	u := &swiftUse{file: p.file, name: t.text, under: t.under, line: t.line, at: p.i, scope: s}
	p.m.uses = append(p.m.uses, u)
	return u
}

func (p *swiftParser) modifier(s *swiftScope) {
	p.markStatic()
	next := p.peekAt(1)
	switch {
	case isAnyWord(next):
		p.i++
	case next.text == "(" && p.peekAt(3).text == ")" && p.peekAt(4).line == next.line:
		p.i += 4
	default:
		p.expr(s)
	}
}

// markStatic notes a static or class modifier for the next declaration.
func (p *swiftParser) markStatic() {
	if t := p.peek().text; t == "static" || t == "class" {
		p.static = true
	}
}

func (p *swiftParser) skipLine(*swiftScope) {
	line := p.next().line
	for p.more() && p.peek().line == line && !p.is("{") {
		p.i++
	}
}

func (p *swiftParser) head(*swiftScope) {
	p.condition = true
	p.i++
}

// forHead binds the loop's pattern names.
func (p *swiftParser) forHead(s *swiftScope) {
	p.head(s)
	for p.inside("{") && !p.is("in") {
		p.bind(s, p.word(), "")
		p.accept(",")
		p.skipPunct()
	}
	p.accept("in")
}

func (p *swiftParser) skipPunct() {
	if t := p.peek(); !isWord(t) && !p.is("in") && !p.is("{") {
		p.i++
	}
}

func (p *swiftParser) catchHead(s *swiftScope) {
	p.head(s)
	p.bind(s, "error", "")
}
