package mdtemplate

import (
	"fmt"
	"slices"
	"strings"
)

// labelFindings judges what each C4 diagram says beyond its palette. Each
// has a title, each container and component a technology, and each legend
// exactly its section's kinds.
func (p palette) labelFindings(diagrams []diagram) []Finding {
	if len(p.kinds) == 0 {
		return nil
	}
	var out []Finding
	for _, d := range slices.DeleteFunc(slices.Clone(diagrams), notC4) {
		out = append(out, p.diagramLabelFindings(d)...)
		out = append(out, legendKindFindings(d, sectionKinds(diagrams, d.section))...)
	}
	return out
}

// diagramLabelFindings fails a C4 diagram without a title line, and each
// container or component whose technology is empty.
func (p palette) diagramLabelFindings(d diagram) []Finding {
	f := readFacts(d)
	var out []Finding
	if !f.titled {
		out = append(out, Finding{d.line, fmt.Sprintf("%s %s needs a `title` line as %s has", d.kind, role(d), p.path)})
	}
	for _, e := range f.elements {
		if takesTechnology(e.kind) && e.technology == "" {
			out = append(out, Finding{e.line, fmt.Sprintf("%s %s needs a technology as its third argument, as %s gives each %s", e.macro, e.alias, p.path, e.kind)})
		}
	}
	return out
}

// legendKindFindings fails a legend that lacks a kind its section's diagrams
// use, or shows a kind they do not. A diagram that is no legend, or a
// section with no diagram, judges none.
func legendKindFindings(legend diagram, used []string) []Finding {
	if !legend.legend || used == nil {
		return nil
	}
	drawn := kindsOf(legend)
	var out []Finding
	for _, kind := range missing(used, drawn) {
		out = append(out, Finding{legend.line, fmt.Sprintf("legend needs a %s sample, a kind the diagram uses", kind)})
	}
	for _, kind := range missing(drawn, used) {
		out = append(out, Finding{legend.line, fmt.Sprintf("legend shows %s, a kind the diagram does not use", kind)})
	}
	return out
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

// sectionKinds lists the kinds the C4 diagrams of a section use, legends
// aside, or nil when the section has none.
func sectionKinds(diagrams []diagram, section string) []string {
	var kinds []string
	for _, d := range diagrams {
		if d.section == section && !d.legend && !notC4(d) {
			kinds = union(kinds, kindsOf(d))
		}
	}
	return kinds
}

// kindsOf lists the element kinds one diagram declares, in order.
func kindsOf(d diagram) []string {
	var kinds []string
	for _, e := range readFacts(d).elements {
		kinds = union(kinds, []string{e.kind})
	}
	return kinds
}

// union appends each kind of more that kinds does not yet hold.
func union(kinds, more []string) []string {
	return append(kinds, missing(more, kinds)...)
}

// notC4 reports a diagram of another Mermaid type, which these rules skip.
func notC4(d diagram) bool {
	return !strings.HasPrefix(d.kind, "C4")
}

// takesTechnology reports a kind whose macro names a technology, as
// Container(cli, "Runner", "Go", "Description") does.
func takesTechnology(kind string) bool {
	base := strings.TrimSuffix(kind, "_Ext")
	return base == "Container" || base == "Component"
}

// technology is the third argument of a container or component macro,
// unquoted, or "" for any other macro or a missing one.
func technology(name string, args []string) string {
	if !takesTechnology(c4Kind(name)) || len(args) < 3 || strings.HasPrefix(args[2], "$") {
		return ""
	}
	return strings.TrimSpace(strings.Trim(args[2], `"`))
}

// role names a diagram as a legend or the view's diagram in messages.
func role(d diagram) string {
	if d.legend {
		return "legend"
	}
	return "diagram"
}
