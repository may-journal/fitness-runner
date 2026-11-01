package views

import "github.com/may-journal/fitness-runner/go/internal/codemodel"

// Method is a method a Java source declares, with the simple names of the
// annotations on it and on its type.
type Method struct {
	Name        string
	Line        int
	Public      bool
	Constructor bool
	Annotations []string
	Type        []string
}

// mark is what the annotations and modifiers before a declaration state.
type mark struct {
	annotations []string
	public      bool
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
			out = append(out, Method{Name: d.Name, Line: d.Line, Public: m.public, Constructor: len(d.Results) == 0, Annotations: m.annotations, Type: types[d.Owner].annotations})
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
