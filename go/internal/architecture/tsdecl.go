package architecture

import (
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/javascript"
)

// tsExts are the TypeScript sources the code view reads.
var tsExts = []string{".ts", ".tsx"}

func typeScript(file string) bool {
	return contains(tsExts, path.Ext(file))
}

var (
	// tsModifiers precede a declaration or member and change nothing the
	// code view states. Export is read apart, since it marks coverage.
	tsModifiers = strings.Fields("export default declare abstract async public private protected static readonly override accessor const")
	// tsNameFollowers are the tokens after a member's name, so a member
	// named like a modifier, such as static(), stays a name.
	tsNameFollowers = strings.Fields("( : = ? ; ! < , }")
	// tsTypeEnds close a type that runs to them.
	tsTypeEnds = strings.Fields("; , ) ] }")
	// tsJoiners end a line that the next line continues.
	tsJoiners = strings.Fields("= , | & : ? . ( [ { < + - * / % ! ^ ~")
	// tsLeaders start a line that continues the one before.
	tsLeaders = strings.Fields(". ? | & = > extends implements as satisfies")
)

// tsStatements read the top-level declarations the code view resolves,
// each from its keyword at token i; start is the statement's first token.
// Each returns the token after the statement.
var tsStatements = map[string]func(r *tsReader, start, i int, exported bool) int{
	"class": func(r *tsReader, start, i int, exported bool) int {
		return r.container(start, i, kindType, exported)
	},
	"interface": func(r *tsReader, start, i int, _ bool) int {
		return r.container(start, i, kindInterface, false)
	},
	"function": (*tsReader).function,
	"enum":     (*tsReader).enum,
	"type":     (*tsReader).alias,
	"{":        (*tsReader).exportList,
}

// tsReader reads one TypeScript file from its tokens, so comments never
// count and strings stay whole. It keeps top-level classes, interfaces,
// functions, enums, and type aliases, with the fields and methods of class
// and interface bodies. Anything inside a body is ignored.
type tsReader struct {
	x       *declIndex
	file    string
	src     string
	tokens  []javascript.Token
	starts  []int
	classes []*decl
	exports map[string]bool
}

func newTSReader(file, src string) *tsReader {
	return &tsReader{file: file, src: src, tokens: tsLex(file, src), starts: lineStarts(src), exports: map[string]bool{}}
}

// tsLex reads a .tsx file's closing tags as tags, so their slashes never
// start a regex that swallows the line's braces.
func tsLex(file, src string) []javascript.Token {
	if path.Ext(file) == ".tsx" {
		return javascript.LexJSX(src)
	}
	return javascript.Lex(src)
}

// tsDecls reads every TypeScript source that is not a test.
func (m *Model) tsDecls(x *declIndex) {
	for _, s := range m.Sources {
		if typeScript(s.Path) && s.Category != "test" && !lang.IsTestFile(langs.All, s.Path) {
			r := newTSReader(s.Path, strings.Join(m.repo.read(s.Path), "\n"))
			r.x = x
			r.read()
		}
	}
}

func (r *tsReader) read() {
	for i := 0; i < len(r.tokens); {
		i = r.statement(i)
	}
	for _, d := range r.classes {
		d.Exported = d.Exported || r.exports[d.Name]
	}
}

// statement reads one top-level statement from token i and returns the
// token after it.
func (r *tsReader) statement(i int) int {
	start := r.decorators(i)
	exported, j := r.modifiers(start)
	if read := tsStatements[r.word(j)]; read != nil {
		return read(r, start, j, exported)
	}
	return r.end(j) + 1
}

// modifiers skips the modifiers from token i, reporting an export.
func (r *tsReader) modifiers(i int) (bool, int) {
	exported := false
	for ; r.modifier(i); i++ {
		exported = exported || r.is(i, "export")
	}
	return exported, i
}

func (r *tsReader) modifier(i int) bool {
	return contains(tsModifiers, r.word(i)) && !contains(tsNameFollowers, r.word(i+1))
}

// decorators skips each @decorator, with its arguments, from token i.
func (r *tsReader) decorators(i int) int {
	for r.is(i, "@") {
		i = r.qualified(i+1) + 1
		if r.is(i, "(") {
			i = r.close(i) + 1
		}
	}
	return i
}

// container reads a class or interface: its name, heritage clause, and
// body members. Only a class can be exported for coverage.
func (r *tsReader) container(start, i int, kind string, exported bool) int {
	name, h := r.header(i + 1)
	close := r.close(h.open)
	if name == "" {
		return close + 1
	}
	d := r.add(name, "", kind, start, close)
	d.Supers, d.Implements, d.Open, d.Body = h.supers, h.implements, r.line(h.open), r.at(h.open).Pos
	if kind == kindType {
		d.Exported, r.classes = exported, append(r.classes, d)
	}
	r.members(name, h.open+1, close)
	return close + 1
}

// header reads a class or interface name from token i, its type
// parameters, and its heritage clause up to the body's brace.
func (r *tsReader) header(i int) (string, heritage) {
	name := ""
	if r.ident(i) && !contains([]string{"extends", "implements"}, r.word(i)) {
		name, i = r.word(i), i+1
	}
	if r.is(i, "<") {
		i = r.angle(i) + 1
	}
	return name, r.heritage(i)
}

// heritage is a class or interface's extends and implements lists, and
// the token that opens its body.
type heritage struct {
	supers, implements []string
	open               int
}

func (r *tsReader) heritage(j int) heritage {
	h := heritage{}
	var list *[]string
	for ; j < len(r.tokens) && !r.is(j, "{"); j = r.typeGroup(j) + 1 {
		list = h.clause(r, j, list)
	}
	h.open = j
	return h
}

// clause reads token j of a heritage clause: extends or implements starts
// a list, and each item's name, without its qualifier, joins it.
func (h *heritage) clause(r *tsReader, j int, list *[]string) *[]string {
	switch r.word(j) {
	case "extends":
		return &h.supers
	case "implements":
		return &h.implements
	}
	if list != nil && r.itemStart(j) {
		*list = append(*list, r.word(r.qualified(j)))
	}
	return list
}

// itemStart reports a name that starts a heritage list item.
func (r *tsReader) itemStart(j int) bool {
	return r.ident(j) && contains([]string{"extends", "implements", ","}, r.word(j-1))
}

func (r *tsReader) members(owner string, from, to int) {
	for i := from; i < to; {
		i = r.member(owner, i)
	}
}

// member reads one class or interface member from token i and returns the
// token after it. A get accessor is a field of its result type; a set
// accessor, an index signature, or a static block declares nothing.
func (r *tsReader) member(owner string, i int) int {
	start := r.decorators(i)
	_, j := r.modifiers(start)
	accessor := ""
	if contains([]string{"get", "set"}, r.word(j)) && r.named(j+1) {
		accessor, j = r.word(j), j+1
	}
	if !r.named(j) {
		return r.end(j) + 1
	}
	return r.memberAt(owner, start, j, accessor)
}

// named reports a member or function name: an identifier or a string.
func (r *tsReader) named(i int) bool {
	return r.ident(i) || r.at(i).Kind == javascript.String
}

func (r *tsReader) memberAt(owner string, start, j int, accessor string) int {
	name, k := r.at(j).Text, j+1
	if r.is(k, "?") || r.is(k, "!") {
		k++
	}
	if r.is(k, "<") {
		k = r.angle(k) + 1
	}
	if r.is(k, "(") {
		return r.method(owner, name, start, k, accessor)
	}
	return r.field(owner, name, start, k)
}

// method reads a function or method from its parameter list at open: the
// parameters, an optional result type, and a body or none.
func (r *tsReader) method(owner, name string, start, open int, accessor string) int {
	close := r.close(open)
	result, next := r.returns(close + 1)
	last := r.body(next)
	if accessor == "set" {
		return last + 1
	}
	d := r.add(name, owner, ownedKind(owner, kindFunc, kindMethod), start, last)
	if accessor == "get" {
		d.Kind, d.Type = kindField, result
		return last + 1
	}
	d.Params, d.Results = r.params(open, close), resultList(result)
	return last + 1
}

// ownedKind is member for a declaration a type owns, else top.
func ownedKind(owner, top, member string) string {
	if owner != "" {
		return member
	}
	return top
}

func resultList(result string) []string {
	if result == "" {
		return nil
	}
	return []string{result}
}

// returns reads a result type annotation from token i, if any, and the
// token after it.
func (r *tsReader) returns(i int) (string, int) {
	if !r.is(i, ":") {
		return "", i
	}
	end := r.typeEnd(i + 1)
	return r.render(i+1, end), end
}

// body returns the last token of a body or signature that ends at i.
func (r *tsReader) body(i int) int {
	switch {
	case r.is(i, "{"):
		return r.close(i)
	case r.is(i, ";") || r.is(i, ","):
		return i
	}
	return i - 1
}

// field reads a field's optional type and initializer from token k.
func (r *tsReader) field(owner, name string, start, k int) int {
	typ, next := r.returns(k)
	last := r.finish(next)
	d := r.add(name, owner, ownedKind(owner, kindValue, kindField), start, last)
	d.Type = typ
	return last + 1
}

// finish returns the last token of a member whose type ends before token
// i: an initializer runs to its statement's end; a semicolon or comma
// closes it.
func (r *tsReader) finish(i int) int {
	switch {
	case r.is(i, "="):
		return r.end(i)
	case r.is(i, ";") || r.is(i, ","):
		return i
	}
	return i - 1
}

// params reads the parameters between the parentheses at open and close.
func (r *tsReader) params(open, close int) []param {
	var params []param
	for start := open + 1; start < close; {
		end := r.paramEnd(start, close)
		params, start = append(params, r.param(start, end)), end+1
	}
	return params
}

// paramEnd returns the comma or closing parenthesis after the parameter
// at token from. Its type steps over type parameters, but its default is
// an expression, where a < compares.
func (r *tsReader) paramEnd(from, close int) int {
	step := r.typeGroup
	j := from
	for ; j < close && !r.is(j, ","); j = step(j) + 1 {
		if r.is(j, "=") && !r.arrow(j+1) {
			step = r.group
		}
	}
	return min(j, close)
}

// param reads one parameter: modifiers and a rest marker, the name or
// destructuring pattern, and the stated type before any default.
func (r *tsReader) param(from, to int) param {
	for r.modifier(from) || r.is(from, ".") {
		from++
	}
	end := r.typeGroup(from) + 1
	p := param{name: r.render(from, end)}
	if r.is(end, "?") {
		end++
	}
	if r.is(end, ":") {
		p.typ = r.render(end+1, r.defaultAt(end+1, to))
	}
	return p
}

// defaultAt finds the = of a parameter default, or to.
func (r *tsReader) defaultAt(from, to int) int {
	for j := from; j < to; j = r.typeGroup(j) + 1 {
		if r.is(j, "=") && !r.arrow(j+1) {
			return j
		}
	}
	return to
}

// function reads a top-level function declaration.
func (r *tsReader) function(start, i int, _ bool) int {
	j := i + 1
	if r.is(j, "*") {
		j++
	}
	if !r.ident(j) {
		return r.end(i) + 1
	}
	return r.memberAt("", start, j, "")
}

// enum reads an enum as a type; its cases are not members here.
func (r *tsReader) enum(start, i int, _ bool) int {
	if !r.ident(i+1) || !r.is(i+2, "{") {
		return r.end(i) + 1
	}
	close := r.close(i + 2)
	r.add(r.word(i+1), "", kindType, start, close)
	return close + 1
}

// alias reads a type alias as a type.
func (r *tsReader) alias(start, i int, _ bool) int {
	last := r.end(i)
	if r.ident(i + 1) {
		r.add(r.word(i+1), "", kindType, start, last)
	}
	return last + 1
}

// exportList marks the names an export list such as export { Store }
// exports from this file. A re-export from another module marks none.
func (r *tsReader) exportList(_, i int, exported bool) int {
	close := r.close(i)
	if exported && !r.is(close+1, "from") {
		for j := i + 1; j < close; j++ {
			r.exportName(j)
		}
	}
	return r.end(i) + 1
}

func (r *tsReader) exportName(j int) {
	if r.ident(j) && (r.is(j-1, "{") || r.is(j-1, ",")) {
		r.exports[r.word(j)] = true
	}
}

func (r *tsReader) add(name, owner, kind string, first, last int) *decl {
	return r.x.add(&decl{File: r.file, Name: name, Owner: owner, Kind: kind, Line: r.line(first), End: r.line(last)})
}
