package mdtemplate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// Shape is what a template sets for its doc: ordered level-2 sections, each
// section's diagram, legend and callout table, and the diagram palette.
type Shape struct {
	path     string
	sections []string
	parts    map[string]part
	palette  palette
}

// part is what one template section holds: its first diagram type, legend
// type, and first callout table's columns and caption.
type part struct {
	diagram string
	legend  string
	columns []string
	caption string
}

// Finding is one way a doc drifts from its shape, at a line or 0 for the
// whole doc.
type Finding struct {
	Line int
	Text string
}

// ShapeFromTemplate reads a named template's shape. path names the template
// in messages.
func ShapeFromTemplate(path, body string) Shape {
	s := Shape{path: path, parts: map[string]part{}}
	s.sections = h2s(body)
	v := readView(body)
	for _, d := range slices.Backward(v.diagrams) {
		p := s.parts[d.section]
		if d.legend {
			p.legend = d.kind
		} else {
			p.diagram = d.kind
		}
		s.parts[d.section] = p
	}
	s.palette = readPalette(path, v.diagrams)
	for _, t := range slices.Backward(v.tables) {
		p := s.parts[t.section]
		p.columns, p.caption = t.columns, firstSentence(t.caption)
		s.parts[t.section] = p
	}
	return s
}

// Validate returns every way body drifts from the shape: opening sections,
// shared sections' diagrams and tables, and diagram palettes.
func (s Shape) Validate(body string) []Finding {
	var out []Finding
	have := h2s(body)
	if opening := have[:min(len(have), len(s.sections))]; !slices.Equal(opening, s.sections) {
		out = append(out, Finding{0, fmt.Sprintf("sections must open with %s as %s does", quoteSections(s.sections), s.path)})
	}
	v := readView(body)
	for _, name := range append([]string{""}, s.sections...) {
		if name == "" || slices.Contains(have, name) {
			out = append(out, s.partFindings(name, v)...)
		}
	}
	return append(out, s.palette.allFindings(v.diagrams)...)
}

// h2s lists a doc's level-2 headings in order.
func h2s(body string) []string {
	var out []string
	for _, h := range mdx.Headings(body) {
		if h.Level == 2 {
			out = append(out, h.Text)
		}
	}
	return out
}

// partFindings judges one section: the types of its diagrams and legends,
// then its callout tables.
func (s Shape) partFindings(name string, v view) []Finding {
	want := s.parts[name]
	var out []Finding
	if want.legend != "" && !v.hasLegend(name, want.legend) {
		out = append(out, Finding{0, fmt.Sprintf("%s needs a %s legend as %s has", sectionName(name), want.legend, s.path)})
	}
	for _, d := range v.diagramsIn(name) {
		out = append(out, s.typeFindings(d, want)...)
	}
	if want.columns != nil {
		out = append(out, s.tableFindings(name, want, v.tablesIn(name))...)
	}
	return out
}

// tableFindings judges a section's callout tables against the template's
// columns and caption, and asks for one when the section has none.
func (s Shape) tableFindings(name string, want part, tables []table) []Finding {
	cols := strings.Join(want.columns, ", ")
	if len(tables) == 0 {
		return []Finding{{0, fmt.Sprintf("%s needs a callout table with columns %s as %s has", sectionName(name), cols, s.path)}}
	}
	var out []Finding
	for _, t := range tables {
		if !slices.Equal(t.columns, want.columns) {
			out = append(out, Finding{t.line, fmt.Sprintf("callout table columns %s must be %s as %s has", strings.Join(t.columns, ", "), cols, s.path)})
		}
		if !strings.HasPrefix(t.caption, want.caption) {
			out = append(out, Finding{t.line, fmt.Sprintf("callout table needs the caption %q above it as %s has", want.caption, s.path)})
		}
	}
	return out
}

// typeFindings fails a diagram or legend whose Mermaid type differs from
// the one the template's section uses.
func (s Shape) typeFindings(d diagram, want part) []Finding {
	what, kind := "diagram", want.diagram
	if d.legend {
		what, kind = "legend", want.legend
	}
	if kind == "" || d.kind == kind {
		return nil
	}
	return []Finding{{d.line, fmt.Sprintf("%s type %s must be %s as %s has", what, d.kind, kind, s.path)}}
}

// view is a doc's diagrams and callout tables, each in its section.
type view struct {
	diagrams []diagram
	tables   []table
}

// diagram is one Mermaid block: its section, its type, whether it is a
// legend, its fence line, and its source.
type diagram struct {
	section, kind string
	legend        bool
	line          int
	body          string
}

type table struct {
	section, caption string
	columns          []string
	line             int
}

func (v view) hasLegend(section, kind string) bool {
	return slices.ContainsFunc(v.diagramsIn(section), func(d diagram) bool { return d.legend && d.kind == kind })
}

func (v view) diagramsIn(section string) []diagram {
	var out []diagram
	for _, d := range v.diagrams {
		if d.section == section {
			out = append(out, d)
		}
	}
	return out
}

func (v view) tablesIn(section string) []table {
	var out []table
	for _, t := range v.tables {
		if t.section == section {
			out = append(out, t)
		}
	}
	return out
}

// readView finds a doc's diagrams and callout tables, reading a legend the
// way the architecture checks do.
func readView(body string) view {
	lines := strings.Split(body, "\n")
	headings := mdx.Headings(body)
	var v view
	for _, block := range mermaid.ParseDoc(body) {
		switch b := block.(type) {
		case *mermaid.DiagramBlock:
			v.diagrams = append(v.diagrams, diagram{sectionAt(headings, b.Line), diagramKind(b.Body), isLegend(b, headings), b.Line, b.Body})
		case *mermaid.CalloutTableBlock:
			v.tables = append(v.tables, table{sectionAt(headings, b.Line), captionAbove(lines, b.Line), b.Header, b.Line})
		}
	}
	return v
}

var titleLegend = regexp.MustCompile(`(?im)^\s*title\s+legend\s*$`)

func isLegend(b *mermaid.DiagramBlock, headings []mdx.Heading) bool {
	return !b.HasCallouts() || titleLegend.MatchString(b.Body) || strings.EqualFold(headingAbove(headings, b.Line), "Legend")
}

// headingAbove is the text of the nearest heading of any level above line.
func headingAbove(headings []mdx.Heading, line int) string {
	text := ""
	for _, h := range headings {
		if h.Line < line {
			text = h.Text
		}
	}
	return text
}

// diagramKind is a diagram's Mermaid type with its direction, such as
// `flowchart TB`: its first line past any front matter and comments.
func diagramKind(body string) string {
	lines := strings.Split(mdx.StripFrontMatter(strings.TrimLeft(body, "\n")), "\n")
	for _, line := range lines {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "%%") {
			return strings.Join(strings.Fields(t), " ")
		}
	}
	return ""
}

// sectionAt names the level-2 section holding a line, or "" for the opening.
func sectionAt(headings []mdx.Heading, line int) string {
	name := ""
	for _, h := range headings {
		if h.Level == 2 && h.Line < line {
			name = h.Text
		}
	}
	return name
}

// captionAbove is the paragraph just above a table's header line, past blank
// lines and HTML comments, joined into one line.
func captionAbove(lines []string, header int) string {
	i := header - 2
	for i >= 0 && skippable(lines, &i) {
	}
	var para []string
	for ; i >= 0 && inParagraph(lines[i]); i-- {
		para = append([]string{strings.TrimSpace(lines[i])}, para...)
	}
	return strings.TrimSpace(htmlComment.ReplaceAllString(strings.Join(para, " "), ""))
}

var htmlComment = regexp.MustCompile(`<!--.*?-->`)

// inParagraph reports whether a line is prose, not blank, a heading, fence,
// table row, or comment delimiter.
func inParagraph(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && t != "-->" && !slices.ContainsFunc(notProse, func(p string) bool { return strings.HasPrefix(t, p) })
}

// notProse are the line openings that end a paragraph.
var notProse = []string{"#", "```", "|", "<!--"}

// skippable steps i past one blank line, or one HTML comment that fills its
// own lines, reporting whether it moved.
func skippable(lines []string, i *int) bool {
	start, ok := blankOrComment(lines, *i)
	if ok {
		*i = start - 1
	}
	return ok
}

// blankOrComment reports whether line i is blank or closes a comment, plus the
// line that comment opens on.
func blankOrComment(lines []string, i int) (int, bool) {
	t := strings.TrimSpace(lines[i])
	if t == "" {
		return i, true
	}
	if !strings.HasSuffix(t, "-->") {
		return i, false
	}
	start := i
	for start > 0 && !strings.Contains(lines[start], "<!--") {
		start--
	}
	return start, strings.HasPrefix(strings.TrimSpace(lines[start]), "<!--")
}

func firstSentence(text string) string {
	if s := mdx.Sentences(text); len(s) > 0 {
		return s[0]
	}
	return ""
}

var (
	frontKey  = regexp.MustCompile(`^([A-Za-z][\w-]*):\s*(.*?)\s*$`)
	blockItem = regexp.MustCompile(`^\s*-\s+(.+?)\s*$`)
)

// front is a doc's parsed front matter: ordered top-level keys, list items,
// scalar values, with indented keys belonging to a value.
type front struct {
	keys    []string
	lists   map[string][]string
	scalars map[string]string
}

// parseFront reads body's front matter in one pass.
func parseFront(body string) front {
	f := front{lists: map[string][]string{}, scalars: map[string]string{}}
	fm, _ := mdx.FrontMatter(body)
	for _, line := range strings.Split(fm, "\n") {
		f.line(line)
	}
	return f
}

// line reads one front matter line: a key with its flow list or scalar, or
// a block item of the key above.
func (f *front) line(line string) {
	if m := frontKey.FindStringSubmatch(line); m != nil {
		f.key(m[1], m[2])
	} else if item := blockItem.FindStringSubmatch(line); item != nil && len(f.keys) > 0 {
		last := f.keys[len(f.keys)-1]
		f.lists[last] = append(f.lists[last], unquote(item[1]))
	}
}

// key records one top-level key and its value, a flow list or a scalar.
func (f *front) key(name, value string) {
	f.keys = append(f.keys, name)
	f.lists[name] = flowItems(value)
	if value != "" && !strings.HasPrefix(value, "[") {
		f.scalars[name] = value
	}
}

// flowItems returns the items of a `[a, 'b']` value, never nil, so a key
// with a scalar value still reads as present.
func flowItems(value string) []string {
	items := []string{}
	inner, open := strings.CutPrefix(value, "[")
	inner, closed := strings.CutSuffix(inner, "]")
	if !open || !closed {
		return items
	}
	for _, part := range strings.Split(inner, ",") {
		if item := unquote(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func unquote(s string) string {
	return strings.Trim(strings.TrimSpace(s), `'"`)
}

func sectionName(name string) string {
	if name == "" {
		return "the opening"
	}
	return "`## " + name + "`"
}

func quoteSections(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "`## " + n + "`"
	}
	return strings.Join(quoted, ", ")
}
