package mdtemplate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// Shape is the whole shape a named template sets for the one doc it governs:
// the opening level-2 sections in order, and per section its legend, its
// callout table columns, and the caption above that table. The opening,
// before any level-2 heading, is the section named "". Front matter is
// optional, so the shape never reads it.
type Shape struct {
	path     string
	sections []string
	parts    map[string]part
}

// part is what one template section holds: the Mermaid type of its legend,
// and the columns and caption of its first callout table. Empty means none.
type part struct {
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
	for _, l := range v.legends {
		p := s.parts[l.section]
		p.legend = l.kind
		s.parts[l.section] = p
	}
	for _, t := range slices.Backward(v.tables) {
		p := s.parts[t.section]
		p.columns, p.caption = t.columns, firstSentence(t.caption)
		s.parts[t.section] = p
	}
	return s
}

// Validate returns every way body drifts from the shape: the opening
// sections, then each shared section's legend and tables.
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
	return out
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

// partFindings judges one section: its legend, then its callout tables.
func (s Shape) partFindings(name string, v view) []Finding {
	want := s.parts[name]
	var out []Finding
	if want.legend != "" && !v.hasLegend(name, want.legend) {
		out = append(out, Finding{0, fmt.Sprintf("%s needs a %s legend as %s has", sectionName(name), want.legend, s.path)})
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

// view is a doc's legends and callout tables, each in its section.
type view struct {
	legends []legend
	tables  []table
}

type legend struct{ section, kind string }

type table struct {
	section, caption string
	columns          []string
	line             int
}

func (v view) hasLegend(section, kind string) bool {
	return slices.Contains(v.legends, legend{section, kind})
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

// readView finds the legends and callout tables of a doc. A legend is a
// diagram with no callout numbers, one titled Legend, or one under a Legend
// heading, as the architecture checks read them.
func readView(body string) view {
	lines := strings.Split(body, "\n")
	headings := mdx.Headings(body)
	var v view
	for _, block := range mermaid.ParseDoc(body) {
		switch b := block.(type) {
		case *mermaid.DiagramBlock:
			if isLegend(b, headings) {
				v.legends = append(v.legends, legend{sectionAt(headings, b.Line), diagramKind(b.Body)})
			}
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

// diagramKind is a diagram's Mermaid type, its first word past any front
// matter and comments.
func diagramKind(body string) string {
	lines := strings.Split(mdx.StripFrontMatter(strings.TrimLeft(body, "\n")), "\n")
	for _, line := range lines {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "%%") {
			return strings.Fields(t)[0]
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

// inParagraph reports whether a line is prose: not blank, and no heading,
// fence, table row, or line of its own opening or closing a comment.
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

// blankOrComment reports whether line i is blank or closes a comment that
// opens a line of its own, and the line that comment opens on.
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

// front is a doc's parsed front matter: its top-level keys in order, each
// key's list items, flow or block, and the raw value of each key that holds
// a scalar. Indented keys belong to a value and are not keys of their own.
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
