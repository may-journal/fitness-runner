package architecture

import (
	"path"
	"regexp"
	"strings"
)

// kindExtension marks a Swift extension, which adds members and
// conformance to a type but declares none itself.
const kindExtension = "extension"

// cspell:ignore fileprivate nonmutating nonisolated typealias
//
// swiftMember matches a declaration after its attributes and modifiers:
// the keyword, the name, and the rest of the line.
var swiftMember = regexp.MustCompile(`^((?:(?:@\w+(?:\([^)]*\))?|public|open|internal|fileprivate|private(?:\(set\))?|final|static|class|override|mutating|nonmutating|convenience|required|lazy|weak|unowned|dynamic|nonisolated|indirect)\s+)*)(class|struct|enum|protocol|actor|extension|typealias|func|init|var|let|case)\b\s*([A-Za-z_][\w.]*)?(.*)$`)

var swiftKinds = map[string]string{
	"class": kindType, "struct": kindType, "enum": kindType, "actor": kindType, "typealias": kindType,
	"protocol": kindInterface, "extension": kindExtension, "func": kindFunc, "init": kindFunc,
	"var": kindValue, "let": kindValue, "case": kindValue,
}

// swiftFrame is an open brace a declaration started. Owner is the type a
// type or extension body adds members to; a function body owns nothing.
type swiftFrame struct {
	d     *decl
	owner string
	depth int
}

// swiftDecls reads one Swift file lexically: comments and string literals
// stripped, braces counted, and #if conditions tracked. A declaration
// directly inside a type or extension body is a member; anything inside a
// function body is a local binding and ignored.
type swiftDecls struct {
	x       *declIndex
	file    string
	code    []string
	under   []string
	depth   int
	frames  []swiftFrame
	pending *swiftFrame
}

func (m *Model) swiftDecls(x *declIndex) {
	for _, s := range m.Sources {
		if path.Ext(s.Path) == ".swift" && s.Category != "test" {
			r := &swiftDecls{x: x, file: s.Path}
			r.read(m.repo.read(s.Path))
		}
	}
}

func (r *swiftDecls) read(lines []string) {
	sc, cond := &scanner{literals: true}, &conditions{}
	for _, line := range lines {
		code := sc.line(line)
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
		r.declare(match, i, opens > closes)
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

func (r *swiftDecls) declare(match []string, i int, opens bool) {
	d := r.newDecl(match, i)
	frame := swiftFrame{d: d, depth: r.depth + 1}
	if d.container() {
		frame.owner = d.Name
	}
	r.pending = nil
	switch {
	case opens:
		r.frames = append(r.frames, frame)
	case d.callable() || frame.owner != "":
		r.pending = &frame
	}
}

func (r *swiftDecls) newDecl(match []string, i int) *decl {
	keyword, name, rest := match[2], match[3], match[4]
	d := &decl{File: r.file, Name: name, Owner: r.owner(), Kind: swiftKinds[keyword], Condition: r.under[i], Line: i + 1, End: i + 1}
	d.Exported = strings.Contains(" "+match[1], " public ") || strings.Contains(" "+match[1], " open ")
	if keyword == "init" {
		d.Name, rest = "init", strings.TrimLeft(name+rest, "?!")
	}
	r.shape(d, rest, i)
	return r.x.add(d)
}

// shape reads what follows the name: a member's kind, a function's
// signature, a stored value's type, or a type's inheritance clause.
func (r *swiftDecls) shape(d *decl, rest string, i int) {
	switch {
	case d.callable():
		d.Params, d.Results = swiftSignature(r.header(i))
		d.Kind = memberKind(d, kindMethod)
	case d.Kind == kindValue:
		d.Type = swiftValueType(rest)
		d.Kind = memberKind(d, kindField)
	case d.container():
		d.Supers = swiftSupers(rest)
	}
}

func memberKind(d *decl, member string) string {
	if d.Owner != "" {
		return member
	}
	return d.Kind
}

// header joins a declaration's lines until its parentheses balance, so a
// signature split over lines still reads whole.
func (r *swiftDecls) header(i int) string {
	text := r.code[i]
	for j := i + 1; j < len(r.code) && strings.Count(text, "(") > strings.Count(text, ")"); j++ {
		text += " " + r.code[j]
	}
	return text
}

// swiftSignature reads parameter types and the result after ->.
func swiftSignature(header string) ([]param, []string) {
	open := strings.Index(header, "(")
	if open < 0 {
		return nil, nil
	}
	close := matching(header, open)
	var params []param
	for _, item := range splitTop(header[open+1 : close]) {
		params = append(params, swiftParam(item))
	}
	return params, swiftResult(header[close+1:])
}

func swiftParam(item string) param {
	labels, typ, _ := strings.Cut(item, ":")
	typ, _, _ = strings.Cut(typ, "=")
	words := strings.Fields(labels)
	name := ""
	if len(words) > 0 {
		name = words[len(words)-1]
	}
	return param{name: name, typ: normalSpace(typ)}
}

func swiftResult(after string) []string {
	_, result, ok := strings.Cut(after, "->")
	if !ok {
		return nil
	}
	result, _, _ = strings.Cut(result, "{")
	result, _, _ = strings.Cut(result, " where ")
	if result = normalSpace(result); result == "" {
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
	return normalSpace(typ)
}

// swiftSupers reads an inheritance clause, such as ": Base, Codable".
func swiftSupers(rest string) []string {
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "<") {
		rest = rest[matching(rest, 0)+1:]
	}
	clause, ok := strings.CutPrefix(strings.TrimSpace(rest), ":")
	if !ok {
		return nil
	}
	clause, _, _ = strings.Cut(clause, "{")
	clause, _, _ = strings.Cut(clause, " where ")
	var supers []string
	for _, item := range splitTop(clause) {
		base, _, _ := strings.Cut(strings.TrimSpace(item), "<")
		supers = append(supers, base)
	}
	return supers
}

// matching finds the bracket closing the one at open, or the text's end.
func matching(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		depth += arrowSafeDepth(text, i)
		if depth == 0 {
			return i
		}
	}
	return len(text) - 1
}

func bracketDepth(c byte) int {
	switch c {
	case '(', '[', '<', '{':
		return 1
	case ')', ']', '>', '}':
		return -1
	}
	return 0
}

// splitTop splits on commas outside brackets; an empty text has no items.
func splitTop(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var items []string
	depth, start := 0, 0
	for i := 0; i < len(text); i++ {
		depth += arrowSafeDepth(text, i)
		if text[i] == ',' && depth == 0 {
			items, start = append(items, strings.TrimSpace(text[start:i])), i+1
		}
	}
	return append(items, strings.TrimSpace(text[start:]))
}

// arrowSafeDepth is bracketDepth, except the > of a -> arrow closes
// nothing.
func arrowSafeDepth(text string, i int) int {
	if text[i] == '>' && i > 0 && text[i-1] == '-' {
		return 0
	}
	return bracketDepth(text[i])
}

func normalSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
