package mdtemplate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// palette is a template's one look for C4 and class diagrams: element kinds and
// their style statements; with none it judges nothing.
type palette struct {
	path   string
	kinds  []string
	styles map[string]styleRule
}

// styleRule is the style a kind carries: the key that compares two styles,
// and the statement with {alias} where the element's alias goes.
type styleRule struct {
	key, form string
}

func (r styleRule) show(alias string) string {
	return strings.ReplaceAll(r.form, "{alias}", alias)
}

// styledType reports a Mermaid type the palette judges.
func styledType(kind string) bool {
	return strings.HasPrefix(kind, "C4") || kind == "classDiagram"
}

// readPalette collects the kinds the template's C4 and class diagrams use,
// and the first style each kind carries.
func readPalette(path string, diagrams []diagram) palette {
	p := palette{path: path, styles: map[string]styleRule{}}
	for _, d := range diagrams {
		p.learn(d)
	}
	return p
}

// learn adds one template diagram's kinds and styles to the palette.
func (p *palette) learn(d diagram) {
	if !styledType(d.kind) {
		return
	}
	f := readFacts(d)
	kindOf := map[string]string{}
	for _, e := range f.elements {
		kindOf[e.alias] = e.kind
		p.addKind(e.kind)
	}
	for _, s := range f.styles {
		p.addStyle(kindOf[s.alias], s)
	}
}

func (p *palette) addKind(kind string) {
	if !slices.Contains(p.kinds, kind) {
		p.kinds = append(p.kinds, kind)
	}
}

// addStyle keeps the first style the template gives a kind.
func (p *palette) addStyle(kind string, s styled) {
	if _, set := p.styles[kind]; kind != "" && !set {
		p.styles[kind] = styleRule{s.key, s.form}
	}
}

// allFindings judges every diagram of a doc against the palette.
func (p palette) allFindings(diagrams []diagram) []Finding {
	var out []Finding
	for _, d := range diagrams {
		out = append(out, p.findings(d)...)
	}
	return out
}

// findings judges one diagram against the palette: each element has a used kind
// with its exact style, and no stray style appears.
func (p palette) findings(d diagram) []Finding {
	if len(p.kinds) == 0 || !styledType(d.kind) {
		return nil
	}
	f := readFacts(d)
	out := p.elementsFindings(f)
	out = append(out, f.undeclaredFindings()...)
	return append(out, p.strayFindings(f)...)
}

// elementsFindings judges each declared element once, with every style
// that names it.
func (p palette) elementsFindings(f facts) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for _, e := range f.elements {
		if !seen[e.alias] {
			seen[e.alias] = true
			out = append(out, p.elementFindings(e, f.stylesOf(e.alias))...)
		}
	}
	return out
}

// elementFindings judges one element's kind and the styles that name it.
func (p palette) elementFindings(e element, got []styled) []Finding {
	if !slices.Contains(p.kinds, e.kind) {
		return []Finding{{e.line, fmt.Sprintf("%s %s is a kind %s does not use; use %s", e.macro, e.alias, p.path, strings.Join(p.kinds, ", "))}}
	}
	rule, styledKind := p.styles[e.kind]
	if !styledKind {
		return p.unstyledFindings(e, got)
	}
	if len(got) == 0 {
		return []Finding{{e.line, fmt.Sprintf("%s %s needs `%s` as %s styles each %s", e.macro, e.alias, rule.show(e.alias), p.path, e.kind)}}
	}
	return p.styleFindings(e, rule, got)
}

// unstyledFindings fails every style on an element whose kind the template
// leaves unstyled, such as a boundary.
func (p palette) unstyledFindings(e element, got []styled) []Finding {
	var out []Finding
	for _, s := range got {
		out = append(out, Finding{s.line, fmt.Sprintf("`%s` styles %s %s, which %s leaves unstyled", s.text, e.macro, e.alias, p.path)})
	}
	return out
}

// styleFindings fails an element's first style when it differs from its
// kind's, and every further style.
func (p palette) styleFindings(e element, rule styleRule, got []styled) []Finding {
	var out []Finding
	if got[0].key != rule.key {
		out = append(out, Finding{got[0].line, fmt.Sprintf("`%s` must be `%s` as %s styles each %s", got[0].text, rule.show(e.alias), p.path, e.kind)})
	}
	for _, s := range got[1:] {
		out = append(out, Finding{s.line, fmt.Sprintf("`%s` styles %s again; give it one style", s.text, e.alias)})
	}
	return out
}

func (p palette) strayFindings(f facts) []Finding {
	var out []Finding
	for _, s := range f.strays {
		out = append(out, Finding{s.line, fmt.Sprintf("`%s` is a style %s does not use; give each element only its kind's style", s.text, p.path)})
	}
	return out
}

// facts is what one diagram declares and styles. class marks a class
// diagram, and depth counts the class bodies open while it is read.
type facts struct {
	elements []element
	styles   []styled
	strays   []stray
	class    bool
	depth    int
}

// element is one declared C4 element, boundary or class, its kind folding Db,
// Queue and outside-person forms into their base.
type element struct {
	alias, macro, kind string
	line               int
}

// styled is one style statement that names an element.
type styled struct {
	alias, key, form, text string
	line                   int
}

// stray is a style statement or color the palette never allows.
type stray struct {
	text string
	line int
}

func (f facts) stylesOf(alias string) []styled {
	var out []styled
	for _, s := range f.styles {
		if s.alias == alias {
			out = append(out, s)
		}
	}
	return out
}

// undeclaredFindings fails a style that names no element of the diagram.
func (f facts) undeclaredFindings() []Finding {
	var out []Finding
	for _, s := range f.styles {
		if !slices.ContainsFunc(f.elements, func(e element) bool { return e.alias == s.alias }) {
			out = append(out, Finding{s.line, fmt.Sprintf("`%s` styles %s, which this diagram does not declare", s.text, s.alias)})
		}
	}
	return out
}

// readFacts reads a C4 or class diagram's elements and style statements.
func readFacts(d diagram) facts {
	f := facts{class: d.kind == "classDiagram"}
	for i, raw := range strings.Split(d.body, "\n") {
		f.line(strings.TrimSpace(raw), d.line+1+i)
	}
	return f
}

// line reads one statement. A %%{init} directive sets colors, so it is a
// stray; other comments say nothing.
func (f *facts) line(t string, at int) {
	switch {
	case strings.HasPrefix(t, "%%{"):
		f.strays = append(f.strays, stray{t, at})
	case t == "" || strings.HasPrefix(t, "%%"):
	case f.class:
		f.classLine(t, at)
	default:
		f.c4Line(t, at)
	}
}

var (
	classDecl  = regexp.MustCompile(`^class\s+([A-Za-z_][\w-]*)`)
	classStyle = regexp.MustCompile(`^style\s+(\S+)\s+(.+)$`)
	classNote  = regexp.MustCompile(`^<<[^>]*>>\s*([A-Za-z_][\w-]*)$`)
)

// c4Line reads one C4 macro call.
func (f *facts) c4Line(t string, at int) {
	if m := mermaid.C4Macro.FindStringSubmatch(t); m != nil {
		f.c4Call(m[1], mermaid.SplitArgs(m[2]), t, at)
	}
}

func (f *facts) c4Call(name string, args []string, t string, at int) {
	switch c4Role(name, args) {
	case "style":
		rest := args[1:]
		f.styles = append(f.styles, styled{strings.Trim(args[0], `"`), styleKey(rest), "UpdateElementStyle({alias}, " + strings.Join(rest, ", ") + ")", t, at})
	case "stray":
		f.strays = append(f.strays, stray{t, at})
	case "element":
		f.elements = append(f.elements, element{strings.Trim(args[0], `"`), name, c4Kind(name), at})
	}
}

// c4Role sorts a C4 macro as a style, a stray style or color, an element
// declaration, or neither, as Rel and Lay_R are.
func c4Role(name string, args []string) string {
	switch {
	case name == "UpdateElementStyle":
		return "style"
	case styling(name, args):
		return "stray"
	case mermaid.IsC4Element(name):
		return "element"
	}
	return ""
}

// styling reports a macro that sets a look: an Update macro other than
// UpdateLayoutConfig, which only lays out, or a color argument.
func styling(name string, args []string) bool {
	return strings.HasPrefix(name, "Update") && name != "UpdateLayoutConfig" || slices.ContainsFunc(args, colorArg)
}

// colorArg reports a named C4 argument that sets a color.
func colorArg(arg string) bool {
	return strings.HasPrefix(arg, "$") && strings.Contains(strings.ToLower(arg), "color")
}

// classLine reads one class diagram statement, skipping class bodies.
func (f *facts) classLine(t string, at int) {
	if f.inBody(t) {
		return
	}
	if classStray(t) {
		f.strays = append(f.strays, stray{t, at})
	}
	f.classStatement(t, at)
}

// inBody reports a line inside an open class body, closing it at "}".
func (f *facts) inBody(t string) bool {
	if f.depth == 0 {
		return false
	}
	if t == "}" {
		f.depth--
	}
	return true
}

// classStray reports a class diagram statement that colors by class.
func classStray(t string) bool {
	return strings.Contains(t, ":::") || strings.HasPrefix(t, "classDef ") || strings.HasPrefix(t, "cssClass ")
}

// classStatement reads a style line, a class declaration, or the classes
// another statement names.
func (f *facts) classStatement(t string, at int) {
	if m := classStyle.FindStringSubmatch(t); m != nil {
		f.styles = append(f.styles, styled{m[1], styleKey(strings.Split(m[2], ",")), "style {alias} " + m[2], t, at})
		return
	}
	if m := classDecl.FindStringSubmatch(t); m != nil {
		f.declare(m[1], at, strings.HasSuffix(t, "{"))
		return
	}
	for _, alias := range classesNamed(t) {
		f.declare(alias, at, false)
	}
}

// classesNamed lists the classes an arrow, an annotation such as
// `<<interface>> Store`, or a member line such as `Record : +ID` names.
func classesNamed(t string) []string {
	if m := mermaid.Edge.FindStringSubmatch(t); m != nil {
		return []string{m[1], m[4]}
	}
	if m := classNote.FindStringSubmatch(t); m != nil {
		return m[1:]
	}
	if m := mermaid.ClassMember.FindStringSubmatch(t); m != nil {
		return m[1:]
	}
	return nil
}

func (f *facts) declare(alias string, at int, opens bool) {
	f.elements = append(f.elements, element{alias, "class", "class", at})
	if opens {
		f.depth++
	}
}

// c4Kind folds a C4 macro's Db and Queue forms into their base kind, and an
// outside person into Person.
func c4Kind(name string) string {
	kind := strings.NewReplacer("Db", "", "Queue", "").Replace(name)
	if kind == "Person_Ext" {
		return "Person"
	}
	return kind
}

// styleKey compares two styles whatever their order, spacing, or case.
func styleKey(parts []string) string {
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		keys = append(keys, strings.ToLower(strings.Join(strings.Fields(part), "")))
	}
	slices.Sort(keys)
	return strings.Join(keys, ",")
}
