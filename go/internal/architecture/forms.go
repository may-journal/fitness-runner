package architecture

import (
	"encoding/csv"
	"regexp"
	"strings"
)

var (
	macro     = regexp.MustCompile(`^([A-Za-z_]+)\((.*)\)\s*(\{)?$`)
	classNode = regexp.MustCompile(`^class\s+([^\s\[{]+)\s*(?:\["([^"]*)"\])?\s*(\{)?$`)
	namespace = regexp.MustCompile(`^namespace\s+([^\s{]+)\s*\{$`)
	member    = regexp.MustCompile(`^[A-Za-z_][\w-]*\s*:`)
	subgraph  = regexp.MustCompile(`^subgraph\s+([^\s\[]+)\s*(?:\[(.*)\])?$`)
	flowNode  = regexp.MustCompile(`^([^\s\[\]({>]+)\s*([\[({>].*)$`)
	nodeClass = regexp.MustCompile(`:::([\w-]+)\s*;?$`)
	edge      = regexp.MustCompile(`^([A-Za-z_][\w-]*)\s*(?:"[^"]*"\s*)?(<-->|<\|--|<\|\.\.|<--|<\.\.|\*--|o--|--\*|--o|--x|--\|>|\.\.\|>|-->|-\.->|==>|\.\.>|---|--|\.\.)\s*(?:"[^"]*"\s*)?(?:\|([^|]*)\|\s*)?([A-Za-z_][\w-]*)\s*(?::\s*(.*))?$`)
)

var c4Nodes = strings.Fields("Person System SystemDb SystemQueue Container ContainerDb ContainerQueue Component ComponentDb ComponentQueue Boundary Enterprise_Boundary System_Boundary Container_Boundary")

var c4Relations = strings.Fields("Rel BiRel Rel_U Rel_Up Rel_D Rel_Down Rel_L Rel_Left Rel_R Rel_Right Rel_Back")

// c4 reads one Mermaid C4 macro. Boundaries become parents; the label and
// technology strings may name repository paths.
func (d *diagram) c4(line string, at Location) {
	call := macro.FindStringSubmatch(line)
	if call == nil {
		d.unsupported(line, at)
		return
	}
	r := csv.NewReader(strings.NewReader(call[2]))
	r.TrimLeadingSpace = true
	args, err := r.Read()
	if err != nil || len(args) < 2 {
		d.m.fail(at, "malformed C4 statement %q", line)
		return
	}
	d.c4Call(call[1], args, line, at, call[3] == "{")
}

func (d *diagram) c4Call(name string, args []string, line string, at Location, opens bool) {
	switch {
	case contains(c4Relations, name):
		d.c4Relation(name, args, line, at)
	case contains(c4Nodes, strings.TrimSuffix(name, "_Ext")):
		d.c4Element(name, args, line, at, opens)
	default:
		d.m.fail(at, "unsupported C4 macro %q", name)
	}
}

func (d *diagram) c4Relation(name string, args []string, line string, at Location) {
	from, to := args[0], args[1]
	if name == "Rel_Back" {
		from, to = to, from
	}
	d.relate(from, to, strings.Join(args[2:], ", "), line, at)
}

func (d *diagram) c4Element(name string, args []string, line string, at Location, opens bool) {
	if e := d.add(args[0], name, args[1], line, at); e != nil {
		d.textClaims(e.ID, strings.Join(args[1:], " "), at)
	}
	if opens {
		d.parents = append(d.parents, args[0])
	}
}

// textClaims reads repository paths from diagram strings, such as a
// boundary label "Runner binary (go/cmd/runner)".
func (d *diagram) textClaims(owner, text string, at Location) {
	for _, word := range strings.Fields(text) {
		d.claim(owner, word, at)
	}
}

// class reads a class diagram statement. Class bodies hold members only.
func (d *diagram) class(line string, at Location) {
	if match := classNode.FindStringSubmatch(line); match != nil {
		d.classNode(match, line, at)
		return
	}
	if match := namespace.FindStringSubmatch(line); match != nil {
		d.add(match[1], "namespace", match[1], line, at)
		d.parents = append(d.parents, match[1])
		return
	}
	d.edgeOr(line, at, member)
}

func (d *diagram) classNode(match []string, line string, at Location) {
	label := match[2]
	if label == "" {
		label = match[1]
	}
	d.add(match[1], "class", label, line, at)
	d.inClassBody = match[3] == "{"
}

// flow reads a flowchart statement. The node's ":::class" is its kind.
func (d *diagram) flow(line string, at Location) {
	if strings.HasPrefix(line, "class ") {
		return
	}
	if match := subgraph.FindStringSubmatch(line); match != nil {
		d.add(match[1], "subgraph", match[2], line, at)
		d.parents = append(d.parents, match[1])
		return
	}
	d.edgeOr(line, at, nil)
}

// edgeOr reads an edge; otherwise a flowchart node, or a class member when
// members is set.
func (d *diagram) edgeOr(line string, at Location, members *regexp.Regexp) {
	if match := edge.FindStringSubmatch(line); match != nil {
		d.edge(match, line, at)
		return
	}
	if members != nil {
		d.memberOr(line, at, members)
		return
	}
	d.flowNode(line, at)
}

func (d *diagram) memberOr(line string, at Location, members *regexp.Regexp) {
	if !members.MatchString(line) {
		d.unsupported(line, at)
	}
}

func (d *diagram) edge(match []string, line string, at Location) {
	from, to := match[1], match[4]
	if strings.HasPrefix(match[2], "<") && match[2] != "<-->" {
		from, to = to, from
	}
	d.relate(from, to, match[3]+match[5], line, at)
}

func (d *diagram) flowNode(line string, at Location) {
	match := flowNode.FindStringSubmatch(line)
	if match == nil {
		d.unsupported(line, at)
		return
	}
	kind, label := "node", match[2]
	if class := nodeClass.FindStringSubmatch(label); class != nil {
		kind, label = class[1], strings.TrimSuffix(strings.TrimSpace(label), class[0])
	}
	d.add(match[1], kind, strings.Trim(label, `[](){}>/\ ;`), line, at)
}
