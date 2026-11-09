package mdtemplate

import (
	"fmt"
	"slices"
	"strings"
)

// labels is what a template's C4 diagrams say beyond their styles: whether
// they carry a title, and the kinds that name a technology.
type labels struct {
	titled       bool
	technologies []string
}

// learn adds one template diagram's title and technologies.
func (l *labels) learn(d diagram, f facts) {
	if notC4(d) {
		return
	}
	l.titled = l.titled || f.titled
	for _, e := range f.elements {
		if e.technology != "" {
			l.technologies = union(l.technologies, []string{e.kind})
		}
	}
}

// c4 is one C4 diagram of a doc, read once: its block, statements and kinds.
type c4 struct {
	diagram
	facts facts
	kinds []string
}

// readC4 reads each C4 diagram of a doc once, skipping other types.
func readC4(diagrams []diagram) []c4 {
	var out []c4
	for _, d := range slices.DeleteFunc(slices.Clone(diagrams), notC4) {
		f := readFacts(d)
		out = append(out, c4{d, f, kindsOf(f)})
	}
	return out
}

// labelFindings judges what each C4 diagram says beyond its palette. It needs
// the template's title and technologies, and legends key exactly the kinds
// in use.
func (p palette) labelFindings(diagrams []diagram) []Finding {
	if len(p.kinds) == 0 {
		return nil
	}
	read := readC4(diagrams)
	var out []Finding
	for _, c := range read {
		out = append(out, p.diagramLabelFindings(c)...)
	}
	return append(out, legendKindFindings(read)...)
}

// diagramLabelFindings fails a C4 diagram without the title the template
// gives, and each element without the technology the template gives its kind.
func (p palette) diagramLabelFindings(c c4) []Finding {
	var out []Finding
	if p.labels.titled && !c.facts.titled {
		out = append(out, Finding{c.line, fmt.Sprintf("%s %s needs a `title` line as %s has", c.kind, role(c.diagram), p.path)})
	}
	for _, e := range slices.DeleteFunc(slices.Clone(c.facts.elements), p.labels.named) {
		out = append(out, Finding{e.line, fmt.Sprintf("%s %s needs a technology as %s gives each %s", e.macro, e.alias, p.path, e.kind)})
	}
	return out
}

// named reports an element that names a technology, or whose kind the
// template gives none.
func (l labels) named(e element) bool {
	return e.technology != "" || !slices.Contains(l.technologies, e.kind)
}

// legendKindFindings judges the legends of each section together against
// the kinds that section's diagrams use. A legend in a section of its own,
// such as `## Legend`, keys every diagram of the doc.
func legendKindFindings(read []c4) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for _, c := range read {
		if c.legend && !seen[c.section] {
			seen[c.section] = true
			out = append(out, sectionLegendFindings(read, c)...)
		}
	}
	return out
}

// sectionLegendFindings fails, at the section's first legend, each kind the
// diagrams use that no legend shows, and each kind shown that none uses.
func sectionLegendFindings(read []c4, first c4) []Finding {
	used, drawn := keyedKinds(read, first.section), kindsIn(read, first.section, true)
	if used == nil {
		return nil
	}
	var out []Finding
	for _, kind := range missing(used, drawn) {
		out = append(out, Finding{first.line, fmt.Sprintf("legend needs a %s sample, a kind the diagram uses", kind)})
	}
	for _, kind := range missing(drawn, used) {
		out = append(out, Finding{first.line, fmt.Sprintf("legend shows %s, a kind the diagram does not use", kind)})
	}
	return out
}

// keyedKinds lists the kinds a section's legends key: its own diagrams'
// kinds, or every diagram's when the section holds only legends.
func keyedKinds(read []c4, section string) []string {
	if kinds := kindsIn(read, section, false); kinds != nil {
		return kinds
	}
	var kinds []string
	for _, c := range read {
		if !c.legend {
			kinds = union(kinds, c.kinds)
		}
	}
	return kinds
}

// kindsIn lists the kinds a section's legends, or its other diagrams, use.
func kindsIn(read []c4, section string, legend bool) []string {
	var kinds []string
	for _, c := range read {
		if c.section == section && c.legend == legend {
			kinds = union(kinds, c.kinds)
		}
	}
	return kinds
}

// kindsOf lists the element kinds one diagram declares, in order.
func kindsOf(f facts) []string {
	var kinds []string
	for _, e := range f.elements {
		kinds = union(kinds, []string{e.kind})
	}
	return kinds
}

// missing lists the kinds of want that have does not hold, in order.
func missing(want, have []string) []string {
	var out []string
	for _, kind := range want {
		if !slices.Contains(have, kind) {
			out = append(out, kind)
		}
	}
	return out
}

// union appends each kind of more that kinds does not yet hold.
func union(kinds, more []string) []string {
	return append(kinds, missing(more, kinds)...)
}

// notC4 reports a diagram of another Mermaid type, which these rules skip.
func notC4(d diagram) bool {
	return !strings.HasPrefix(d.kind, "C4")
}

// titleLine reports a title with text, as a `title` statement or a front
// matter `title:` key.
func titleLine(t string) bool {
	for _, key := range []string{"title ", "title:"} {
		if rest, ok := strings.CutPrefix(t, key); ok && strings.TrimSpace(rest) != "" {
			return true
		}
	}
	return false
}

// technology is the technology a container or component macro names, by
// `$techn=` or as its third argument, or "" for any other macro.
func technology(name string, args []string) string {
	if base := strings.TrimSuffix(c4Kind(name), "_Ext"); base != "Container" && base != "Component" {
		return ""
	}
	if i := slices.IndexFunc(args, techArg); i >= 0 {
		return strings.TrimSpace(unquote(strings.TrimPrefix(args[i], "$techn=")))
	}
	return positional(args, 2)
}

// positional is the unquoted argument at i when it is given by position,
// not by a `$name=` key, or "".
func positional(args []string, i int) string {
	if len(args) <= i || strings.HasPrefix(args[i], "$") {
		return ""
	}
	return strings.TrimSpace(unquote(args[i]))
}

// techArg reports the named argument that gives a C4 technology.
func techArg(arg string) bool {
	return strings.HasPrefix(arg, "$techn=")
}

// role names a diagram as a legend or the view's diagram in messages.
func role(d diagram) string {
	if d.legend {
		return "legend"
	}
	return "diagram"
}
