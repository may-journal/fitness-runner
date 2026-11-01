// Package views reads Swift sources for the architecture views: declarations
// lexically, and every product file parsed as one module whose names resolve.
package views

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
)

// Reader is the Swift module's reader for the architecture views.
var Reader = codemodel.Reader{
	Language:   swift.Technology,
	Exts:       []string{swift.Ext},
	Read:       read,
	References: references,
}

// cspell:ignore fileprivate nonmutating nonisolated typealias
//
// swiftMember matches a declaration after its attributes and modifiers:
// the keyword, the name, and the rest of the line.
var swiftMember = regexp.MustCompile(`^((?:(?:@\w+(?:\([^)]*\))?|public|open|internal|fileprivate|private(?:\(set\))?|final|static|class|override|mutating|nonmutating|convenience|required|lazy|weak|unowned|dynamic|nonisolated|indirect)\s+)*)(class|struct|enum|protocol|actor|extension|typealias|func|init|var|let|case)\b\s*([A-Za-z_][\w.]*)?(.*)$`)

var swiftKinds = map[string]string{
	"class": codemodel.KindType, "struct": codemodel.KindType, "enum": codemodel.KindType, "actor": codemodel.KindType, "typealias": codemodel.KindType,
	"protocol": codemodel.KindInterface, "extension": codemodel.KindExtension, "func": codemodel.KindFunc, "init": codemodel.KindFunc,
	"var": codemodel.KindValue, "let": codemodel.KindValue, "case": codemodel.KindValue,
}

// swiftFrame is an open brace a declaration started. Owner is the type a
// type or extension body adds members to; a function body owns nothing.
type swiftFrame struct {
	d     *codemodel.Decl
	owner string
	depth int
}

// swiftDecls reads one Swift file lexically: comments and string literals
// stripped, braces counted, and #if conditions tracked. A declaration
// directly inside a type or extension body is a member; anything inside a
// function body is a local binding and ignored.
type swiftDecls struct {
	x       *codemodel.Index
	file    string
	code    []string
	under   []string
	depth   int
	frames  []swiftFrame
	pending *swiftFrame
}

// read reads every Swift source that is not a test. Extensions anywhere
// in the module add members to a type, so every declaration shares one
// scope.
func read(repo codemodel.Repo, x *codemodel.Index) codemodel.Code {
	for _, file := range codemodel.FilesWith(repo.Code, swift.Ext) {
		r := &swiftDecls{x: x, file: file}
		r.read(repo.Read(file))
	}
	return codemodel.Code{Relate: func(g *codemodel.Graph) { relate(g, repo) }}
}

func (r *swiftDecls) read(lines []string) {
	sc, cond := &codemodel.Scanner{Literals: true}, &conditions{}
	for _, line := range lines {
		code := sc.Line(line)
		if cond.directive(code) {
			code = ""
		}
		r.code, r.under = append(r.code, code), append(r.under, cond.String())
	}
	for i := range r.code {
		r.line(i)
	}
}

func (r *swiftDecls) line(i int) {
	before := r.depth
	opens, closes := strings.Count(r.code[i], "{"), strings.Count(r.code[i], "}")
	if match := swiftMember.FindStringSubmatch(strings.TrimSpace(r.code[i])); match != nil && r.inScope(before) {
		r.declare(match, i, opens > closes, opens > 0 && opens == closes)
	} else if r.opensPending(before, opens > closes) {
		r.frames, r.pending = append(r.frames, *r.pending), nil
	}
	r.depth = before + opens - closes
	r.close(i)
}

// opensPending reports the brace that opens a declaration whose header
// spanned lines.
func (r *swiftDecls) opensPending(depth int, opens bool) bool {
	return r.pending != nil && opens && depth == r.pending.depth-1
}

// inScope reports a line at file level or directly in a type body.
func (r *swiftDecls) inScope(depth int) bool {
	if len(r.frames) == 0 {
		return depth == 0
	}
	top := r.frames[len(r.frames)-1]
	return top.owner != "" && top.depth == depth
}

func (r *swiftDecls) close(i int) {
	for len(r.frames) > 0 && r.frames[len(r.frames)-1].depth > r.depth {
		r.frames[len(r.frames)-1].d.End = i + 1
		r.frames = r.frames[:len(r.frames)-1]
	}
	if r.pending != nil && r.depth < r.pending.depth-1 {
		r.pending = nil
	}
}

func (r *swiftDecls) owner() string {
	if len(r.frames) == 0 {
		return ""
	}
	return r.frames[len(r.frames)-1].owner
}

// declare records a declaration. Its body opens on this line, closes on
// it too (complete), or waits on a later line as pending.
func (r *swiftDecls) declare(match []string, i int, opens, complete bool) {
	d := r.newDecl(match, i)
	r.pending = nil
	if !complete {
		r.openBody(d, opens)
	}
}

// openBody tracks the body d opens on its line, or the one it waits for.
func (r *swiftDecls) openBody(d *codemodel.Decl, opens bool) {
	frame := swiftFrame{d: d, depth: r.depth + 1}
	if d.Container() {
		frame.owner = d.Key()
	}
	switch {
	case opens:
		r.frames = append(r.frames, frame)
	case d.Callable() || frame.owner != "":
		r.pending = &frame
	}
}

func (r *swiftDecls) newDecl(match []string, i int) *codemodel.Decl {
	keyword, name, rest := match[2], match[3], match[4]
	d := &codemodel.Decl{File: r.file, Name: name, Owner: r.owner(), Kind: swiftKinds[keyword], Scope: swift.Ext, Condition: r.under[i], Line: i + 1, End: i + 1}
	d.Exported = strings.Contains(" "+match[1], " public ") || strings.Contains(" "+match[1], " open ")
	if keyword == "init" {
		d.Name, rest = "init", strings.TrimLeft(name+rest, "?!")
	}
	r.shape(d, rest, i)
	return r.x.Add(d)
}

// shape reads what follows the name: a member's kind, a function's
// signature, a stored value's type, or a type's inheritance clause.
func (r *swiftDecls) shape(d *codemodel.Decl, rest string, i int) {
	switch {
	case d.Callable():
		d.Params, d.Results = swiftSignature(r.header(rest, i))
		d.Kind = memberKind(d, codemodel.KindMethod)
	case d.Kind == codemodel.KindValue:
		d.Type = swiftValueType(rest)
		d.Kind = memberKind(d, codemodel.KindField)
	case d.Container():
		d.Supers = swiftSupers(rest)
	}
}

func memberKind(d *codemodel.Decl, member string) string {
	if d.Owner != "" {
		return member
	}
	return d.Kind
}

// header joins what follows a declaration's name with the next lines until its
// parentheses balance, so a split signature reads whole.
func (r *swiftDecls) header(rest string, i int) string {
	text := rest
	for j := i + 1; j < len(r.code) && strings.Count(text, "(") > strings.Count(text, ")"); j++ {
		text += " " + r.code[j]
	}
	return text
}

// swiftSignature reads parameter types and the result after ->.
func swiftSignature(header string) ([]codemodel.Param, []string) {
	open := strings.Index(header, "(")
	if open < 0 {
		return nil, nil
	}
	close := codemodel.Matching(header, open)
	var params []codemodel.Param
	for _, item := range codemodel.SplitTop(header[open+1 : close]) {
		params = append(params, swiftParam(item))
	}
	return params, swiftResult(header[close+1:])
}

func swiftParam(item string) codemodel.Param {
	labels, typ, _ := strings.Cut(item, ":")
	typ, _, _ = strings.Cut(typ, "=")
	words := strings.Fields(labels)
	name := ""
	if len(words) > 0 {
		name = words[len(words)-1]
	}
	return codemodel.Param{Name: name, Type: codemodel.NormalSpace(typ)}
}

func swiftResult(after string) []string {
	_, result, ok := strings.Cut(after, "->")
	if !ok {
		return nil
	}
	result, _, _ = strings.Cut(result, "{")
	result, _, _ = strings.Cut(result, " where ")
	if result = codemodel.NormalSpace(result); result == "" {
		return nil
	}
	return []string{result}
}

// swiftValueType reads a stored or computed value's stated type.
func swiftValueType(rest string) string {
	_, typ, ok := strings.Cut(rest, ":")
	if !ok {
		return ""
	}
	typ, _, _ = strings.Cut(typ, "=")
	typ, _, _ = strings.Cut(typ, "{")
	return codemodel.NormalSpace(typ)
}

// swiftSupers reads an inheritance clause, such as ": Base, Codable".
func swiftSupers(rest string) []string {
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "<") {
		rest = rest[codemodel.Matching(rest, 0)+1:]
	}
	clause, ok := strings.CutPrefix(strings.TrimSpace(rest), ":")
	if !ok {
		return nil
	}
	clause, _, _ = strings.Cut(clause, "{")
	clause, _, _ = strings.Cut(clause, " where ")
	var supers []string
	for _, item := range codemodel.SplitTop(clause) {
		base, _, _ := strings.Cut(strings.TrimSpace(item), "<")
		supers = append(supers, base)
	}
	return supers
}
