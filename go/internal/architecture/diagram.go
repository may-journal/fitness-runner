package architecture

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// subject is the element or relationship a callout number marks.
type subject struct {
	element *Element
	edge    *Relationship
}

func (s subject) owner() string {
	if s.element != nil {
		return s.element.ID
	}
	return ""
}

func (s subject) name() string {
	if s.element != nil {
		return s.element.ID
	}
	return s.edge.From + "->" + s.edge.To
}

// diagram is one model diagram while it is read: its declared elements, its
// edges, and the callout numbers that bind table rows to them.
type diagram struct {
	m           *Model
	file, view  string
	form        string
	elements    map[string]*Element
	order       []*Element
	edges       []*Relationship
	callouts    map[int]subject
	claims      []Claim
	parents     []string
	inClassBody bool
}

var (
	semanticID   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	leadingCount = regexp.MustCompile(`^\d+\s+`)
	rowNumber    = regexp.MustCompile(`^\[?(\d+)\]?$`)
)

func (m *Model) readDiagram(file, view string, b *mermaid.DiagramBlock) *diagram {
	d := &diagram{m: m, file: file, view: view, elements: map[string]*Element{}, callouts: map[int]subject{}}
	if view == "context" && m.contextAt.Path == "" {
		m.contextAt = Location{file, b.Line}
	}
	lines := strings.Split(b.Body, "\n")
	start := frontMatterEnd(lines)
	d.form = diagramForm(lines[start:])
	if !supportedForm(d.form) {
		m.fail(Location{file, b.Line}, "unsupported Mermaid form %q; use C4Context, C4Container, C4Component, flowchart, or classDiagram", d.form)
		return d
	}
	for i, line := range lines[start:] {
		d.statement(strings.TrimSpace(line), Location{file, b.Line + start + i + 1})
	}
	return d
}

// frontMatterEnd skips a leading "---" block that holds a diagram's title
// or config.
func frontMatterEnd(lines []string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

func diagramForm(lines []string) string {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "%%") {
			return line
		}
	}
	return ""
}

func supportedForm(form string) bool {
	first, _, _ := strings.Cut(form, " ")
	return contains([]string{"C4Context", "C4Container", "C4Component", "classDiagram", "flowchart", "graph"}, first)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

var decorations = []string{"%%", "title ", "direction ", "classDef ", "style ", "linkStyle ", "click ", "note ", "<<"}

// accessibility matches Mermaid's accessible title and description lines.
var accessibility = regexp.MustCompile(`^acc[A-Z][a-z]+\s*[:{]`)

func (d *diagram) statement(line string, at Location) {
	if d.decoration(line) || d.closes(line) {
		return
	}
	switch {
	case strings.HasPrefix(d.form, "C4"):
		d.c4(line, at)
	case d.form == "classDiagram":
		d.class(line, at)
	default:
		d.flow(line, at)
	}
}

func (d *diagram) decoration(line string) bool {
	return d.skipped(line) || hasPrefix(line, decorations) || accessibility.MatchString(line)
}

// skipped reports blank lines, the form line, and class body members.
func (d *diagram) skipped(line string) bool {
	return line == "" || line == d.form || (d.inClassBody && line != "}")
}

func hasPrefix(line string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func (d *diagram) closes(line string) bool {
	if line != "}" && line != "end" {
		return false
	}
	if d.inClassBody {
		d.inClassBody = false
		return true
	}
	d.parents = d.parents[:max(len(d.parents)-1, 0)]
	return true
}

func (d *diagram) unsupported(line string, at Location) {
	d.m.fail(at, "unsupported Mermaid statement %q", line)
}

// add declares an element under the innermost open boundary.
func (d *diagram) add(id, kind, label, line string, at Location) *Element {
	if !semanticID.MatchString(id) {
		d.m.fail(at, "invalid ID %q", id)
		return nil
	}
	if first := d.elements[id]; first != nil {
		d.m.fail(at, "duplicate ID %s; first at %s", id, first.At)
		return nil
	}
	e := &Element{ID: id, Kind: kind, Name: displayName(label), Parent: d.parent(), View: d.view, Callout: callout(line), At: at}
	d.elements[id] = e
	d.order = append(d.order, e)
	d.mark(e.Callout, subject{element: e})
	return e
}

func (d *diagram) parent() string {
	if len(d.parents) == 0 {
		return ""
	}
	return d.parents[len(d.parents)-1]
}

func (d *diagram) relate(from, to, label, line string, at Location) {
	r := &Relationship{From: from, To: to, Label: label, View: d.view, Callout: callout(line), At: at}
	d.edges = append(d.edges, r)
	d.mark(r.Callout, subject{edge: r})
}

func (d *diagram) mark(number int, s subject) {
	if _, taken := d.callouts[number]; number > 0 && !taken {
		d.callouts[number] = s
	}
}

func displayName(label string) string {
	return leadingCount.ReplaceAllString(strings.Trim(strings.TrimSpace(label), `"`), "")
}

func callout(line string) int {
	if numbers := mermaid.ExtractDiagramNumbers(line); len(numbers) > 0 {
		return numbers[0]
	}
	return 0
}

// bindRows reads each callout row's text for the element or edge it marks.
func (d *diagram) bindRows(table *mermaid.CalloutTableBlock) {
	for i, row := range table.Rows {
		match := rowNumber.FindStringSubmatch(row[0])
		if match == nil {
			continue
		}
		n, _ := strconv.Atoi(match[1])
		if s, ok := d.callouts[n]; ok {
			d.rowContent(s, strings.Join(row[1:], " | "), Location{d.file, table.Line + 2 + i})
		}
	}
}
