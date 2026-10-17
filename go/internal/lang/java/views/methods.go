package views

import "github.com/may-journal/fitness-runner/go/internal/codemodel"

// Method is a method a Java source declares, with the simple names of the
// annotations on it, such as Test for @org.junit.jupiter.api.Test.
type Method struct {
	Name        string
	Line        int
	Annotations []string
}

// Methods lists the methods src declares, nested types' too, in source
// order, read from tokens as the code view reads them.
func Methods(src string) []Method {
	r := newReader("", src)
	r.x, r.marks = &codemodel.Index{}, map[*codemodel.Decl][]string{}
	for i := 0; i < len(r.tokens); {
		i = r.top(i)
	}
	var out []Method
	for _, d := range r.x.Decls {
		if d.Kind == codemodel.KindMethod {
			out = append(out, Method{Name: d.Name, Line: d.Line, Annotations: r.marks[d]})
		}
	}
	return out
}

// annotations lists the simple names of the annotations from token i up
// to token end.
func (r *reader) annotations(i, end int) []string {
	var out []string
	for ; i < end; i++ {
		if r.is(i, "@") && r.Ident(i+1) {
			out = append(out, r.Word(r.qualified(i+1)))
		}
	}
	return out
}
