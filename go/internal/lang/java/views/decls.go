// Package views reads Java sources for the architecture code view: each
// file's declarations and the relations between the view's nodes, from
// tokens, so comments never count and literals stay whole.
//
// The reader takes the .java product files the views list, so sources in
// a test source set's folder, such as src/test, and generated files stay
// out. It keeps each file's package and its classes, interfaces, enums,
// records and annotation types, nested ones too, with their fields,
// methods and constructors, as JLS chapters 7 to 9 declare them.
//
//   - A label's qualifier may name the package, such as com.example.Store.
//   - A nested type is a member of the type around it, named as
//     Outer.Inner, and a type's members are those its body declares.
//   - An interface and an annotation type are interfaces; a class, an
//     enum and a record are types.
//   - A record's components are its fields, and an enum's constants are
//     fields of the enum's own type.
//   - A constructor is a method named for its type, with no result. A
//     record's compact constructor takes the record's components.
//   - A parameter's type is what it states, with ... for variable arity.
//     Array brackets after a name join its type.
//   - Annotations and modifiers state nothing the view compares, and a
//     receiver parameter is no parameter.
//
// It relates names of other classes, Type.member calls, extends, which is
// inheritance, and implements, which is conformance. The view covers a
// top-level public type. The reader is lexical, without imports, so a
// call through a variable, such as store.save(), relates nothing, and
// overloads differ only by their stated signatures. It reads no
// references, so the components view takes no Java evidence.
package views

import (
	"path"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// Reader is the Java module's reader for the architecture code view.
var Reader = codemodel.Reader{
	Language: java.Technology,
	Exts:     []string{java.Ext},
	Read:     read,
}

// cspell:ignore strictfp

// modifiers precede a declaration and change nothing the view states.
// Public is read apart, since it marks coverage. A sealed or non-sealed
// class names its permitted subclasses, which relate nothing here.
var modifiers = strings.Fields("public protected private abstract static final sealed strictfp transient volatile synchronized native default")

// typeKinds are the keywords that declare a type, by the kind the view
// gives it. An annotation type's keyword is @interface.
var typeKinds = map[string]string{
	"class": codemodel.KindType, "enum": codemodel.KindType, "record": codemodel.KindType,
	"interface": codemodel.KindInterface, "@": codemodel.KindInterface,
}

// reader reads one Java file from its tokens.
type reader struct {
	x      *codemodel.Index
	file   string
	pkg    string
	size   int
	tokens []token
	lines  codemodel.LineIndex
}

func newReader(file, src string) *reader {
	return &reader{file: file, size: len(src), tokens: lex(src), lines: codemodel.NewLineIndex(src)}
}

// read reads every Java source the views list that is not a test.
func read(repo codemodel.Repo, x *codemodel.Index) codemodel.Code {
	for _, file := range codemodel.FilesWith(repo.Code, java.Ext) {
		r := newReader(file, strings.Join(repo.Read(file), "\n"))
		r.x = x
		for i := 0; i < len(r.tokens); {
			i = r.top(i)
		}
	}
	return codemodel.Code{Relate: func(g *codemodel.Graph) { relate(g, repo) }}
}

// Package is the package the Java source src declares, slash-separated
// as compiled classes name it, or "" for the default package.
func Package(src string) string {
	r := newReader("", src)
	_, j := r.modifiers(0)
	if !r.is(j, "package") {
		return ""
	}
	return strings.ReplaceAll(r.render(j+1, r.statementEnd(j)), ".", "/")
}

// relate reads each Java node file's tokens, so comments and literals
// relate nothing.
func relate(g *codemodel.Graph, repo codemodel.Repo) {
	for _, file := range g.Files(func(file string) bool { return path.Ext(file) == java.Ext }) {
		codemodel.RelateTokens(g, file, newReader(file, strings.Join(repo.Read(file), "\n")))
	}
}

// top reads one top-level declaration from token i: the package, an
// import, a type, or a module declaration, which declares nothing here.
// It returns the token after it.
func (r *reader) top(i int) int {
	public, j := r.modifiers(i)
	switch {
	case r.is(j, "package"):
		end := r.statementEnd(j)
		r.pkg = r.render(j+1, end)
		return end + 1
	case r.typeAt(j):
		return r.container(i, j, "", public)
	}
	return max(r.statementEnd(j), i) + 1
}

// modifiers skips the annotations and modifiers from token i, reporting
// public.
func (r *reader) modifiers(i int) (bool, int) {
	public := false
	for next := r.modifier(i); next > i; next = r.modifier(i) {
		public, i = public || r.is(i, "public"), next
	}
	return public, i
}

// modifier returns the token after the annotation or modifier at token i,
// or i when none starts there.
func (r *reader) modifier(i int) int {
	switch {
	case r.is(i, "@") && !r.is(i+1, "interface"):
		return r.annotation(i)
	case r.seq(i, "non", "-", "sealed"):
		return i + 3
	case slices.Contains(modifiers, r.Word(i)):
		return i + 1
	}
	return i
}

// annotation returns the token after the annotation at token i, with its
// arguments.
func (r *reader) annotation(i int) int {
	j := r.qualified(i+1) + 1
	if r.is(j, "(") {
		return r.close(j) + 1
	}
	return j
}

// typeAt reports a type declaration's keyword at token j. A record is a
// keyword only before its name and components.
func (r *reader) typeAt(j int) bool {
	switch r.Word(j) {
	case "@":
		return r.is(j+1, "interface")
	case "record":
		return r.Ident(j+1) && (r.is(j+2, "(") || r.is(j+2, "<"))
	}
	return typeKinds[r.Word(j)] != ""
}

// typeName is the token holding the name of the type whose keyword is at
// token j.
func (r *reader) typeName(j int) int {
	if r.is(j, "@") {
		return j + 2
	}
	return j + 1
}

// container reads a type from its keyword at token j: its name, type
// parameters, record components, the types it extends and implements,
// and its body. Start is the first
// token of its annotations and modifiers. Only a top-level type can be
// public for coverage.
func (r *reader) container(start, j int, owner string, public bool) int {
	keyword, n := r.Word(j), r.typeName(j)
	if !r.Ident(n) {
		return max(r.statementEnd(n), j) + 1
	}
	k, components := r.skipAngle(n+1), -1
	if keyword == "record" && r.is(k, "(") {
		components, k = k, r.close(k)+1
	}
	h := r.heritage(k)
	close := r.close(h.open)
	d := r.add(r.Word(n), owner, typeKinds[keyword], start, close)
	d.Supers, d.Implements, d.Open, d.Body = h.supers, h.implements, r.Line(h.open), r.Pos(h.open)
	d.Exported = public && owner == ""
	r.body(body{d, components}, keyword, h.open, close)
	return close + 1
}

// body is a type whose body is read: its declaration, which owns the
// members by its key, and the parenthesis opening a record's components,
// or -1.
type body struct {
	d          *codemodel.Decl
	components int
}

// body reads a type's record components, enum constants and members.
func (r *reader) body(t body, keyword string, open, close int) {
	if t.components >= 0 {
		r.components(t.d.Key(), t.components)
	}
	from := open + 1
	if keyword == "enum" {
		from = r.constants(t.d, from, close)
	}
	for i := from; i < close; {
		i = r.member(t, i, close)
	}
}

// heritage is a type's extends and implements lists, and the token that
// opens its body.
type heritage struct {
	supers, implements []string
	open               int
}

// clauses start a heritage list by keyword. A permits list relates
// nothing.
var clauses = map[string]func(h *heritage) *[]string{
	"extends":    func(h *heritage) *[]string { return &h.supers },
	"implements": func(h *heritage) *[]string { return &h.implements },
	"permits":    func(*heritage) *[]string { return nil },
}

// heritage reads the clauses from token k to the body's brace.
func (r *reader) heritage(k int) heritage {
	h := heritage{}
	var list *[]string
	for ; k < len(r.tokens) && !r.is(k, "{"); k = r.typeStep(k) + 1 {
		list = h.clause(r, k, list)
	}
	h.open = k
	return h
}

// clause reads token k of a heritage clause: a keyword starts a list,
// and each item's name joins it without its qualifier or type arguments.
func (h *heritage) clause(r *reader, k int, list *[]string) *[]string {
	if start := clauses[r.Word(k)]; start != nil {
		return start(h)
	}
	if list != nil && r.itemStart(k) {
		*list = append(*list, r.Word(r.qualified(k)))
	}
	return list
}

// itemStart reports a name that starts a heritage list item.
func (r *reader) itemStart(k int) bool {
	return r.Ident(k) && slices.Contains([]string{"extends", "implements", ","}, r.Word(k-1))
}

func (r *reader) add(name, owner, kind string, first, last int) *codemodel.Decl {
	return r.x.Add(&codemodel.Decl{File: r.file, Name: name, Owner: owner, Kind: kind, Package: r.pkg, Scope: r.file, Line: r.Line(first), End: r.Line(last)})
}
