package views

import (
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/codemodel"
)

// Method is a method a Java source declares, with the simple names of the
// annotations on it and on its type. Calls are the calls in its body that
// pass a string first.
type Method struct {
	Name        string
	Line        int
	Public      bool
	Constructor bool
	Annotations []string
	Type        []string
	Calls       []Call
}

// Call is a call whose first argument is a string literal, such as
// dynamicTest("name", ...).
type Call struct {
	// Name is the called name as written, dots and all, such as Named.of.
	Name string
	// Text is the string's value.
	Text string
	// Line is the line the string starts on.
	Line int
}

// mark is what the annotations and modifiers before a declaration state.
type mark struct {
	annotations []string
	public      bool
	calls       []Call
}

// Methods lists the methods src declares, nested types' too, in source
// order, read from tokens as the code view reads them.
func Methods(src string) []Method {
	r := newReader("", src)
	r.x, r.marks = &codemodel.Index{}, map[*codemodel.Decl]mark{}
	for i := 0; i < len(r.tokens); {
		i = r.top(i)
	}
	types := r.typeMarks()
	var out []Method
	for _, d := range r.x.Decls {
		if m := r.marks[d]; d.Kind == codemodel.KindMethod {
			out = append(out, Method{Name: d.Name, Line: d.Line, Public: m.public, Constructor: len(d.Results) == 0, Annotations: m.annotations, Type: types[d.Owner].annotations, Calls: m.calls})
		}
	}
	return out
}

// typeMarks maps each type read, by key, to its marks.
func (r *reader) typeMarks() map[string]mark {
	types := map[string]mark{}
	for _, d := range r.x.Decls {
		if d.TypeLike() {
			types[d.Key()] = r.marks[d]
		}
	}
	return types
}

// mark records, when marks are kept, the annotations and modifiers from
// token i up to token end before d.
func (r *reader) mark(d *codemodel.Decl, i, end int) {
	if r.marks == nil {
		return
	}
	public, _ := r.modifiers(i)
	var names []string
	for ; i < end; i++ {
		if r.is(i, "@") && r.Ident(i+1) {
			names = append(names, r.Word(r.qualified(i+1)))
		}
	}
	r.marks[d] = mark{annotations: names, public: public}
}

// calls lists the calls from token from up to token to whose first
// argument is a plain string literal.
func (r *reader) calls(from, to int) []Call {
	var out []Call
	for i := from; i+2 < to; i++ {
		if text, ok := r.stringCall(i); ok {
			out = append(out, Call{Name: r.dotted(i), Text: text, Line: r.Line(i + 2)})
		}
	}
	return out
}

// stringCall reads the string a call at token i passes first, when it
// passes a plain string literal.
func (r *reader) stringCall(i int) (string, bool) {
	if !r.Ident(i) || !r.is(i+1, "(") || r.at(i+2).kind != literal {
		return "", false
	}
	return javaString(r.at(i + 2).text)
}

// dotted spells the dotted name ending at token i.
func (r *reader) dotted(i int) string {
	name := r.Word(i)
	for ; i >= 2 && r.is(i-1, ".") && r.Ident(i-2); i -= 2 {
		name = r.Word(i-2) + "." + name
	}
	return name
}

// javaString reads a one-line string literal's value, decoding its escapes
// as Java does. A text block, a character, or an escape Java lacks is none.
func javaString(lit string) (string, bool) {
	if !strings.HasPrefix(lit, `"`) || strings.HasPrefix(lit, `"""`) {
		return "", false
	}
	var b strings.Builder
	for rest := lit[1 : len(lit)-1]; rest != ""; {
		r, tail, ok := javaChar(rest)
		if !ok {
			return "", false
		}
		b.WriteRune(r)
		rest = tail
	}
	return b.String(), true
}

// javaOnly are the escapes Java has and a Go string lacks.
var javaOnly = map[string]rune{`\s`: ' ', `\'`: '\''}

// javaChar decodes the first character of a string literal's body.
func javaChar(s string) (rune, string, bool) {
	if len(s) >= 2 {
		if r, ok := javaOnly[s[:2]]; ok {
			return r, s[2:], true
		}
	}
	r, _, tail, err := strconv.UnquoteChar(s, '"')
	return r, tail, err == nil
}
