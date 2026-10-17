package views

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
)

// swiftModule is every product Swift file parsed as one module: the
// declarations at file level by name, the members of each type by owner,
// each type's inheritance from its declaration and extensions, the first
// type its own declaration inherits, which super names, and the names the
// code uses.
type swiftModule struct {
	top, members map[string][]*swiftSymbol
	supers       map[string][]string
	bases        map[string]string
	uses         []*swiftUse
}

// maxTypeHops bounds how far a value's type is followed through
// initializers and bases.
const maxTypeHops = 16

// references parses every product Swift file, then resolves each name the
// code uses to the declarations it means, as the compiler's index would,
// and reports the ones in another file. Comments name nothing.
func references(repo codemodel.Repo) ([]codemodel.Reference, []string) {
	mod := &swiftModule{top: map[string][]*swiftSymbol{}, members: map[string][]*swiftSymbol{}, supers: map[string][]string{}, bases: map[string]string{}}
	for _, file := range codemodel.FilesWith(repo.Product, swift.Ext) {
		mod.parse(file, strings.Join(repo.Read(file), "\n"))
	}
	return mod.references(), nil
}

// parse reads one file; a stray closing brace at file level is passed
// over, so the rest of the file is still read.
func (m *swiftModule) parse(file, src string) {
	p := &swiftParser{m: m, file: file, tokens: lexSwift(src)}
	scope := &swiftScope{names: map[string][]*swiftSymbol{}}
	for p.more() {
		p.block(scope, "")
		p.accept("}")
	}
}

func (m *swiftModule) add(sym *swiftSymbol) {
	if sym.owner == "" {
		m.top[sym.name] = append(m.top[sym.name], sym)
		return
	}
	m.members[sym.owner] = append(m.members[sym.owner], sym)
}

func (m *swiftModule) conform(owner string, supers []string) {
	m.supers[owner] = append(m.supers[owner], supers...)
}

// declareType records a type's own inheritance clause, whose first entry
// is the superclass when it has one.
func (m *swiftModule) declareType(owner string, supers []string) {
	m.conform(owner, supers)
	m.bases[owner] = first(supers)
}

// references lists one reference per line, symbol, and declaring file.
func (m *swiftModule) references() []codemodel.Reference {
	var refs []codemodel.Reference
	seen := map[codemodel.Reference]bool{}
	for _, u := range m.uses {
		for _, sym := range m.resolve(u) {
			r := codemodel.Reference{At: codemodel.Location{Path: u.file, Line: u.line}, To: sym.file, Symbol: sym.display(), Condition: u.under}
			if sym.file != u.file && !seen[r] {
				seen[r] = true
				refs = append(refs, r)
			}
		}
	}
	return refs
}

// resolve finds the module declarations a use means; a local means none.
func (m *swiftModule) resolve(u *swiftUse) []*swiftSymbol {
	found := m.meaning(u, 0)
	if len(found) == 1 && found[0].local {
		return nil
	}
	return found
}

// meaning finds what a use names. A member is looked up on its base's
// type; an implicit member's type comes from context the parser does not
// read, so it means nothing.
func (m *swiftModule) meaning(u *swiftUse, hops int) []*swiftSymbol {
	switch {
	case u.implicit || swiftSelf[u.name] || u.name == "":
		return nil
	case u.member:
		return m.withInits(m.memberOf(m.typeOf(u.base, hops+1), u), u)
	}
	return m.withInits(m.lookup(u), u)
}

// lookup resolves an unqualified name from the innermost scope out: a
// local, then a member of each enclosing type, then a file-level symbol.
func (m *swiftModule) lookup(u *swiftUse) []*swiftSymbol {
	for s := u.scope; s != nil; s = s.parent {
		if local := s.visible(u); local != nil {
			return []*swiftSymbol{local}
		}
		if found := m.ownMembers(s, u); len(found) > 0 {
			return found
		}
	}
	return m.pick(m.top[u.name], u)
}

func (m *swiftModule) ownMembers(s *swiftScope, u *swiftUse) []*swiftSymbol {
	if !s.typeBody {
		return nil
	}
	return m.memberOf(s.owner, u)
}

// memberOf finds a type's members named as u, from its own declaration and
// extensions first, then from what it inherits.
func (m *swiftModule) memberOf(owner string, u *swiftUse) []*swiftSymbol {
	return m.inherited(owner, u, map[string]bool{})
}

func (m *swiftModule) inherited(owner string, u *swiftUse, seen map[string]bool) []*swiftSymbol {
	if owner == "" || seen[owner] {
		return nil
	}
	seen[owner] = true
	if own := m.pick(named(m.members[owner], u.name), u); len(own) > 0 {
		return own
	}
	var found []*swiftSymbol
	for _, super := range m.supers[owner] {
		found = append(found, m.inherited(super, u, seen)...)
	}
	return oneFile(found)
}

func named(symbols []*swiftSymbol, name string) []*swiftSymbol {
	var found []*swiftSymbol
	for _, sym := range symbols {
		if sym.name == name {
			found = append(found, sym)
		}
	}
	return found
}

// withInits adds the initializers a call of a type picks, which an
// extension in another file may declare.
func (m *swiftModule) withInits(found []*swiftSymbol, u *swiftUse) []*swiftSymbol {
	if !u.call || len(found) != 1 || found[0].kind != swiftType {
		return found
	}
	return append(found, m.pick(named(m.members[qualify(found[0].owner, found[0].name)], "init"), u)...)
}

// pick keeps the symbols a call's labels fit, and only when they sit in
// one file: overloads the labels cannot tell apart resolve to none.
func (m *swiftModule) pick(symbols []*swiftSymbol, u *swiftUse) []*swiftSymbol {
	var fit []*swiftSymbol
	for _, sym := range symbols {
		if !u.call || !sym.callable() || labelsFit(u, sym) {
			fit = append(fit, sym)
		}
	}
	return oneFile(fit)
}

func oneFile(symbols []*swiftSymbol) []*swiftSymbol {
	for _, sym := range symbols {
		if sym.file != symbols[0].file {
			return nil
		}
	}
	return symbols
}

// labelsFit reports a call whose labels match the parameters in order,
// skipping ones with defaults; a trailing closure fills one left over.
func labelsFit(u *swiftUse, sym *swiftSymbol) bool {
	used, missing := matchLabels(u.labels, sym)
	return used == len(u.labels) && missing <= u.trailing
}

func matchLabels(labels []string, sym *swiftSymbol) (int, int) {
	used, missing := 0, 0
	for i, label := range sym.labels {
		if used < len(labels) && labels[used] == label {
			used++
		} else if !sym.defaults[i] {
			missing++
		}
	}
	return used, missing
}

// typeOf infers the type of the value a use names: a literal's, the
// enclosing type for self, or the type its symbol states or initializes.
// "" means unknown.
func (m *swiftModule) typeOf(u *swiftUse, hops int) string {
	if u == nil || hops > maxTypeHops {
		return ""
	}
	if known, ok := m.knownType(u); ok {
		return known
	}
	found := m.meaning(u, hops)
	if len(found) == 0 {
		return ""
	}
	return m.symbolType(found[0], u, hops)
}

// knownType is the type of a literal, self, Self, or super.
func (m *swiftModule) knownType(u *swiftUse) (string, bool) {
	switch {
	case u.literal != "":
		return u.literal, true
	case u.name == "super":
		return m.bases[u.scope.owner], true
	case swiftSelf[u.name]:
		return u.scope.owner, true
	}
	return "", false
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// symbolType is the type a symbol gives the use naming it: a type itself,
// a called function's result, or a value's stated or initialized type.
func (m *swiftModule) symbolType(sym *swiftSymbol, u *swiftUse, hops int) string {
	if sym.kind == swiftType && sym.typ == "" {
		return qualify(sym.owner, sym.name)
	}
	return m.statedType(sym, u, hops)
}

func (m *swiftModule) statedType(sym *swiftSymbol, u *swiftUse, hops int) string {
	switch {
	case sym.kind == swiftFunc && !u.call:
		return ""
	case sym.typ != "":
		return m.typeNamed(sym.typ, sym.scope)
	}
	return m.typeOf(sym.init, hops+1)
}

// typeNamed qualifies a stated type name by the enclosing types that
// declare a nested type of that name.
func (m *swiftModule) typeNamed(text string, s *swiftScope) string {
	if text == "Self" {
		return s.owner
	}
	for ; s != nil; s = s.parent {
		if s.owner != "" && m.declaresType(s.owner, text) {
			return s.owner + "." + text
		}
	}
	return text
}

func (m *swiftModule) declaresType(owner, name string) bool {
	for _, sym := range named(m.members[owner], name) {
		if sym.kind == swiftType {
			return true
		}
	}
	return false
}
