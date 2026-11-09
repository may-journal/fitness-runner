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

// labelFindings judges what each C4 diagram says beyond its palette: the
// title and technologies the template gives.
func (p palette) labelFindings(diagrams []diagram) []Finding {
	if len(p.kinds) == 0 {
		return nil
	}
	var out []Finding
	for _, d := range slices.DeleteFunc(slices.Clone(diagrams), notC4) {
		out = append(out, p.diagramLabelFindings(d, readFacts(d))...)
	}
	return out
}

// diagramLabelFindings fails a C4 diagram without the title the template
// gives, and each element without the technology the template gives its kind.
func (p palette) diagramLabelFindings(d diagram, f facts) []Finding {
	var out []Finding
	if p.labels.titled && !f.titled {
		out = append(out, Finding{d.line, fmt.Sprintf("%s %s needs a `title` line as %s has", d.kind, role(d), p.path)})
	}
	for _, e := range slices.DeleteFunc(slices.Clone(f.elements), p.labels.named) {
		out = append(out, Finding{e.line, fmt.Sprintf("%s %s needs a technology as %s gives each %s", e.macro, e.alias, p.path, e.kind)})
	}
	return out
}

// named reports an element that names a technology, or whose kind the
// template gives none.
func (l labels) named(e element) bool {
	return e.technology != "" || !slices.Contains(l.technologies, e.kind)
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
