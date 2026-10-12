package architecture

import (
	"regexp"
	"strings"
)

// statedName reads a class label or member as the code view states it: a
// qualified name, and for a function its parameters and an optional
// result, such as render.Table([]Row, int, Palette) string.
var statedName = regexp.MustCompile(`^([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)\s*(?:\((.*)\)\s*(.*))?$`)

// stated is what one label or member says about a declaration.
type stated struct {
	text, name, params, result string
	qual                       []string
	call                       bool
}

func parseStated(text string) (stated, bool) {
	text = strings.TrimSpace(text)
	match := statedName.FindStringSubmatch(text)
	if match == nil {
		return stated{}, false
	}
	parts := strings.Split(match[1], ".")
	s := stated{text: text, name: parts[len(parts)-1], qual: parts[:len(parts)-1], params: match[2], result: match[3]}
	s.call = strings.Contains(text, "(")
	return s, true
}

func (s stated) kind() string {
	if s.call {
		return kindFunc
	}
	return kindType
}

// accepts reports whether a declaration has the kind the label states.
func (s stated) accepts(d *decl) bool {
	if s.call {
		return d.callable()
	}
	return d.typeLike()
}

// matches compares the stated parameters, and the result when stated,
// with the declaration's.
func (s stated) matches(d *decl) bool {
	items := splitTop(s.params)
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

// sameParam accepts a parameter stated by its type, or by name and type.
func sameParam(item string, p param) bool {
	item = normalSpace(item)
	return contains([]string{p.typ, p.name + " " + p.typ, p.name + ": " + p.typ}, item)
}

// results splits a stated result list, with or without parentheses.
func results(text string) []string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "(") && matching(text, 0) == len(text)-1 {
		text = text[1 : len(text)-1]
	}
	return splitTop(text)
}

func sameTypes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if normalSpace(a[i]) != normalSpace(b[i]) {
			return false
		}
	}
	return true
}

// statedMember reads a class body line: a visibility mark, then a method such as
// Run(string) error, or a field as name, name: Type, or Type name.
func statedMember(text string) (stated, string, bool) {
	text = strings.TrimRight(strings.TrimLeft(strings.TrimSpace(text), "+-#~"), "$* ")
	if strings.Contains(text, "(") {
		s, ok := parseStated(text)
		return s, "", ok
	}
	if name, typ, ok := strings.Cut(text, ":"); ok {
		return stated{text: text, name: strings.TrimSpace(name)}, normalSpace(typ), true
	}
	fields := strings.Fields(text)
	if len(fields) == 2 {
		return stated{text: text, name: fields[1]}, fields[0], true
	}
	return stated{text: text, name: text}, "", len(fields) == 1
}
