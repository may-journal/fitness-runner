package views

import "github.com/may-journal/fitness-runner/go/internal/codemodel"

// components reads a record's components, from the parenthesis at open,
// as its fields.
func (r *reader) components(owner string, open int) {
	for _, p := range r.params(open, r.close(open)) {
		d := r.add(p.name, owner, codemodel.KindField, p.first, p.last)
		d.Type = p.typ
	}
}

// constants reads an enum's constants from token i up to the semicolon
// that ends them or the enum's end, and returns the token where its
// members start.
func (r *reader) constants(enum *codemodel.Decl, i, close int) int {
	for i < close && !r.is(i, ";") {
		if i = r.constant(enum, i); r.is(i, ",") {
			i++
		}
	}
	return i
}

// constant reads one enum constant from token i, with its annotations,
// arguments and body, as a field of the enum's type, and returns the
// token after it.
func (r *reader) constant(enum *codemodel.Decl, i int) int {
	_, j := r.modifiers(i)
	if !r.Ident(j) {
		return max(j, i+1)
	}
	last := r.groupAfter(r.groupAfter(j, "("), "{")
	d := r.add(r.Word(j), enum.Key(), codemodel.KindField, i, last)
	d.Type = enum.Name
	return last + 1
}

// groupAfter returns the end of the group open starts right after token
// j, or j.
func (r *reader) groupAfter(j int, open string) int {
	if r.is(j+1, open) {
		return r.close(j + 1)
	}
	return j
}

// member reads one member of t's body from token i and returns the token
// after it: an initializer block, a nested type, a constructor, a
// record's compact constructor, or a typed member.
func (r *reader) member(t body, i, close int) int {
	_, j := r.modifiers(i)
	switch {
	case r.is(j, "{"):
		return r.close(j) + 1
	case r.typeAt(j):
		return r.container(i, j, t.d.Key(), false)
	}
	return r.callable(t, i, j, close)
}

// callable reads a constructor, a compact constructor, or a typed member
// from token j after its modifiers.
func (r *reader) callable(t body, i, j, close int) int {
	if t.components >= 0 && r.seq(j, t.d.Name, "{") {
		return r.compact(t, i, j+1)
	}
	if j = r.skipAngle(j); r.seq(j, t.d.Name, "(") {
		return r.method(t.d.Key(), t.d.Name, i, j+1, "", close)
	}
	return r.typed(t.d.Key(), i, j, close)
}

// compact reads a record's compact constructor, whose body opens at
// token open, as a constructor taking the record's components.
func (r *reader) compact(t body, start, open int) int {
	last := r.close(open)
	d := r.add(t.d.Name, t.d.Key(), codemodel.KindMethod, start, last)
	r.signature(d, t.components, r.close(t.components))
	return last + 1
}

// typed reads a method or fields whose type starts at token j. Anything
// else declares nothing.
func (r *reader) typed(owner string, i, j, close int) int {
	end := r.typeEnd(j)
	switch {
	case end == j || !r.Ident(end):
		return max(r.statementEnd(j), i) + 1
	case r.is(end+1, "("):
		return r.method(owner, r.Word(end), i, end+1, r.render(j, end), close)
	}
	return r.fields(owner, r.render(j, end), i, end)
}

// method reads a method or constructor from its parameters at open, with
// any array brackets after them for its result.
func (r *reader) method(owner, name string, start, open int, result string, close int) int {
	end := r.close(open)
	result, k := r.dims(result, end+1)
	last := r.methodEnd(k, close)
	d := r.add(name, owner, codemodel.KindMethod, start, last)
	r.signature(d, open, end)
	if result != "" {
		d.Results = []string{result}
	}
	return last + 1
}

// signature gives d the parameters between the parentheses at open and
// close.
func (r *reader) signature(d *codemodel.Decl, open, close int) {
	for _, p := range r.params(open, close) {
		d.Params = append(d.Params, codemodel.Param{Name: p.name, Type: p.typ})
	}
}

// methodEnd returns the last token of a method from token k after its
// parameters, past a throws clause or an annotation element's default:
// its body's closing brace, or its semicolon.
func (r *reader) methodEnd(k, close int) int {
	for k < close && !r.is(k, "{") && !r.is(k, ";") {
		k = r.typeStep(k) + 1
	}
	return r.group(k)
}

// dims adds a [] to typ for each pair of array brackets from token k,
// and returns the token after them.
func (r *reader) dims(typ string, k int) (string, int) {
	for r.seq(k, "[", "]") {
		typ, k = typ+"[]", k+2
	}
	return typ, k
}

// fields reads each declarator of a field declaration from its first
// name at token i, with array brackets after a name and an initializer.
func (r *reader) fields(owner, typ string, start, i int) int {
	for {
		t, k := r.dims(typ, i+1)
		end := r.initializerEnd(k)
		d := r.add(r.Word(i), owner, codemodel.KindField, start, end)
		d.Type = t
		if !r.is(end, ",") || !r.Ident(end+1) {
			return end + 1
		}
		i, start = end+1, end+1
	}
}

// param is one parameter: its name, its stated type, and its tokens.
type param struct {
	name, typ   string
	first, last int
}

// params reads the parameters between the parentheses at open and close.
func (r *reader) params(open, close int) []param {
	var out []param
	for start := open + 1; start < close; {
		end := r.paramEnd(start, close)
		if p, ok := r.param(start, end); ok {
			out = append(out, p)
		}
		start = end + 1
	}
	return out
}

// paramEnd returns the comma or closing parenthesis after the parameter
// at token j.
func (r *reader) paramEnd(j, close int) int {
	for j < close && !r.is(j, ",") {
		j = r.typeStep(j) + 1
	}
	return j
}

// param reads one parameter between tokens from and to: its modifiers,
// type, name and any array brackets after the name. A receiver
// parameter, named this, is none.
func (r *reader) param(from, to int) (param, bool) {
	_, j := r.modifiers(from)
	end := r.typeEnd(j)
	typ, _ := r.dims(r.render(j, end), end+1)
	return param{name: r.Word(end), typ: typ, first: from, last: to - 1}, r.Ident(end) && !r.is(to-1, "this")
}
