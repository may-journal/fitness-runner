package architecture

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
)

// statedName reads a class label or member as the code view states it: a
// qualified name, and for a function its parameters and an optional
// result, such as render.Table([]Row, int, Palette) string.
var statedName = regexp.MustCompile(`^([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)\s*(.*)$`)

// stated is what one label or member says about a declaration.
type stated struct {
	text, name, params, result string
	qual                       []string
	call                       bool
}

func parseStated(text string) (stated, bool) {
	text = generics(strings.TrimSpace(text))
	match := statedName.FindStringSubmatch(text)
	if match == nil {
		return stated{}, false
	}
	parts := strings.Split(match[1], ".")
	s := stated{text: text, name: parts[len(parts)-1], qual: parts[:len(parts)-1]}
	return s.parameters(match[2])
}

// parameters reads what follows the name: nothing for a type, else the
// parenthesized parameters, then the result. The closing parenthesis is
// the one matching the first, so a result list in parentheses stays out.
func (s stated) parameters(rest string) (stated, bool) {
	if rest == "" {
		return s, true
	}
	close := codemodel.Matching(rest, 0)
	if rest[0] != '(' || rest[close] != ')' {
		return stated{}, false
	}
	s.call, s.params, s.result = true, rest[1:close], strings.TrimSpace(rest[close+1:])
	return s, true
}

func (s stated) kind() string {
	if s.call {
		return codemodel.KindFunc
	}
	return codemodel.KindType
}

// accepts reports whether a declaration has the kind the label states.
func (s stated) accepts(d *codemodel.Decl) bool {
	if s.call {
		return d.Callable()
	}
	return d.TypeLike()
}

// matches compares the stated parameters, and the result when stated,
// with the declaration's.
func (s stated) matches(d *codemodel.Decl) bool {
	items := codemodel.SplitTop(s.params)
	if len(items) != len(d.Params) {
		return false
	}
	for i, item := range items {
		if !sameParam(item, d.Params[i]) {
			return false
		}
	}
	return s.result == "" || sameTypes(results(s.result), d.Results)
}

// sameParam accepts a parameter stated by its type, or by name and type
// in either order.
func sameParam(item string, p codemodel.Param) bool {
	item = codemodel.TypeText(item)
	for _, form := range []string{p.Type, p.Name + " " + p.Type, p.Name + ": " + p.Type, p.Type + " " + p.Name} {
		if codemodel.TypeText(form) == item {
			return true
		}
	}
	return false
}

// results splits a stated result list, with or without parentheses.
func results(text string) []string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "(") && codemodel.Matching(text, 0) == len(text)-1 {
		text = text[1 : len(text)-1]
	}
	return codemodel.SplitTop(text)
}

func sameTypes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if codemodel.TypeText(a[i]) != codemodel.TypeText(b[i]) {
			return false
		}
	}
	return true
}

// statedMember reads a class body line: a visibility mark, then a method such as
// Run(string) error, or a field as name, name: Type, or Type name, where
// the type is every word before the name.
func statedMember(text string) (stated, string, bool) {
	text = generics(strings.TrimRight(strings.TrimLeft(strings.TrimSpace(text), "+-#~"), "$* "))
	if strings.Contains(text, "(") {
		s, ok := parseStated(text)
		return s, "", ok
	}
	if name, typ, ok := strings.Cut(text, ":"); ok {
		return stated{text: text, name: strings.TrimSpace(name)}, codemodel.NormalSpace(typ), true
	}
	fields := strings.Fields(text)
	if last := len(fields) - 1; last > 0 {
		return stated{text: text, name: fields[last]}, strings.Join(fields[:last], " "), true
	}
	return stated{text: text, name: text}, "", len(fields) == 1
}

// generics reads Mermaid's tilde generics as angle brackets, so
// Promise~boolean~ states Promise<boolean>. A tilde before a name or a
// wildcard's ? opens a list. A tilde before an opening bracket opens one only outside a list,
// so List~Item~[] states List<Item>[]. Any other tilde closes one.
func generics(text string) string {
	b := []byte(text)
	depth := 0
	for i, c := range b {
		if c == '~' {
			b[i] = tildeBracket(text, i, depth)
			depth += tildeDepth[b[i]]
		}
	}
	return string(b)
}

// tildeDepth is how a tilde read as a bracket changes the list depth.
var tildeDepth = map[byte]int{'<': 1, '>': -1}

func tildeBracket(text string, i, depth int) byte {
	next := byte(' ')
	if i+1 < len(text) {
		next = text[i+1]
	}
	if opensList(next) || (depth == 0 && strings.IndexByte("([{", next) >= 0) {
		return '<'
	}
	return '>'
}

// opensList reports a byte a list's first item can start with: a name's,
// or a wildcard's ?.
func opensList(c byte) bool {
	return codemodel.WordByte(c) || c == '?'
}
