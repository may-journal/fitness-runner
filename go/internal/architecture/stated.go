package architecture

import (
	"regexp"
	"strings"
	"unicode"
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
	close := matching(rest, 0)
	if rest[0] != '(' || rest[close] != ')' {
		return stated{}, false
	}
	s.call, s.params, s.result = true, rest[1:close], strings.TrimSpace(rest[close+1:])
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
	item = typeText(item)
	for _, form := range []string{p.typ, p.name + " " + p.typ, p.name + ": " + p.typ} {
		if typeText(form) == item {
			return true
		}
	}
	return false
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
		if typeText(a[i]) != typeText(b[i]) {
			return false
		}
	}
	return true
}

// statedMember reads a class body line: a visibility mark, then a method such as
// Run(string) error, or a field as name, name: Type, or Type name.
func statedMember(text string) (stated, string, bool) {
	text = generics(strings.TrimRight(strings.TrimLeft(strings.TrimSpace(text), "+-#~"), "$* "))
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

// typeText spells a type with a space only between two words and after
// each comma, so Map< K,V > and Map<K, V> compare equal.
func typeText(text string) string {
	var b strings.Builder
	words := strings.Fields(text)
	for i, word := range words {
		if i > 0 && wordByte(words[i-1][len(words[i-1])-1]) && wordByte(word[0]) {
			b.WriteByte(' ')
		}
		b.WriteString(word)
	}
	return strings.ReplaceAll(b.String(), ",", ", ")
}

func wordByte(c byte) bool {
	return c == '_' || c == '$' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

// generics reads Mermaid's tilde generics as angle brackets, so
// Promise~boolean~ states Promise<boolean>. A tilde before a name or an
// opening bracket opens a list; any other tilde closes one.
func generics(text string) string {
	b := []byte(text)
	for i, c := range b {
		if c == '~' {
			b[i] = tildeBracket(text, i)
		}
	}
	return string(b)
}

func tildeBracket(text string, i int) byte {
	if i+1 < len(text) && (wordByte(text[i+1]) || strings.ContainsRune("([{", rune(text[i+1]))) {
		return '<'
	}
	return '>'
}
