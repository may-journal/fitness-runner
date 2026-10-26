package architecture

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

var (
	classNode = regexp.MustCompile(`^class\s+([^\s\[{]+)\s*(?:\["([^"]*)"\])?\s*(\{)?$`)
	namespace = regexp.MustCompile(`^namespace\s+([^\s{]+)\s*\{$`)
	subgraph  = regexp.MustCompile(`^subgraph\s+([^\s\[]+)\s*(?:\[(.*)\])?$`)
	flowNode  = regexp.MustCompile(`^([^\s\[\]({>]+)\s*([\[({>].*)$`)
	nodeClass = regexp.MustCompile(`:::([\w-]+)\s*;?$`)
)

var c4Relations = strings.Fields("Rel BiRel Rel_U Rel_Up Rel_D Rel_Down Rel_L Rel_Left Rel_R Rel_Right Rel_Back")

// c4 reads one Mermaid C4 macro. Boundaries become parents; the label and
// technology strings may name repository paths.
func (d *diagram) c4(line string, at Location) {
	call := mermaid.C4Macro.FindStringSubmatch(line)
	if call == nil {
		d.unsupported(line, at)
		return
	}
	if strings.HasPrefix(call[1], "Update") {
		return
	}
	args := c4Args(call[2])
	if len(args) < 2 {
		d.m.fail(at, "malformed C4 statement %q", line)
		return
	}
	d.c4Call(call[1], args, line, at, call[3] == "{")
}

// c4Args splits positional C4 arguments on commas outside quotes. Named
// arguments such as $tags="v1" style a diagram and stay out of the model.
func c4Args(text string) []string {
	var args []string
	for _, arg := range mermaid.SplitArgs(text) {
		if !strings.HasPrefix(arg, "$") {
			args = append(args, strings.Trim(arg, `"`))
		}
	}
	return args
}

func (d *diagram) c4Call(name string, args []string, line string, at Location, opens bool) {
	switch {
	case contains(c4Relations, name):
		d.c4Relation(name, args, line, at)
	case mermaid.IsC4Element(name):
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
	d.relate(from, to, name, strings.Join(args[2:], ", "), line, at)
}

func (d *diagram) c4Element(name string, args []string, line string, at Location, opens bool) {
	if e := d.add(args[0], name, args[1], line, at); e != nil {
		e.Technology, e.Description = c4Strings(name, args)
		d.textClaims(e.ID, args[1], true, at)
		d.textClaims(e.ID, strings.Join(args[2:], " "), false, at)
	}
	if opens {
		d.parents = append(d.parents, args[0])
	}
}

// c4Strings reads the technology and description after a label. Container
// and component macros take a technology first, as in Container(cli,
// "Runner", "Technology", "Description"); persons, systems, and
// boundaries take only a description.
func c4Strings(name string, args []string) (technology, description string) {
	rest := args[2:]
	if technical(name) && len(rest) > 0 {
		technology, rest = rest[0], rest[1:]
	}
	return technology, strings.Join(rest, " ")
}

func technical(name string) bool {
	return (strings.HasPrefix(name, "Container") || strings.HasPrefix(name, "Component")) && !strings.HasSuffix(name, "Boundary")
}

// textClaims reads repository paths from diagram strings, such as a
// boundary label "Runner binary (go/cmd/runner)". A label may also name a
// root-level source by its bare name; a technology or description may not.
func (d *diagram) textClaims(owner, text string, bare bool, at Location) {
	for _, word := range strings.Fields(text) {
		d.claim(owner, word, bare, at)
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
	d.edgeOr(line, at, mermaid.ClassMember)
}

func (d *diagram) classNode(match []string, line string, at Location) {
	label := match[2]
	if label == "" {
		label = match[1]
	}
	e := d.add(match[1], "class", label, line, at)
	if match[3] != "{" {
		return
	}
	if e == nil {
		e = &Element{}
	}
	d.body = e
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
	if match := mermaid.Edge.FindStringSubmatch(line); match != nil {
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
	d.relate(from, to, match[2], match[3]+match[5], line, at)
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
