package architecture

import (
	"encoding/csv"
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

type element struct {
	local, from, to string
	at              Location
	declaration     *Declaration
}

type diagramGraph map[string]*element

var (
	macro = regexp.MustCompile(`^([A-Za-z_]+)\((.*)\)\s*\{?$`)
	node  = regexp.MustCompile(`^(?:class\s+)?([A-Za-z][A-Za-z0-9_-]*)\s*[\[(]`)
	edge  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*)\s+(-->|-\.->|==>|--|\.\.>|<\|--|<\|\.\.|<--|<\.\.|--\|>|\.\.\|>|\*--|o--)\s*(?:\|([^|]+)\|\s*)?([A-Za-z][A-Za-z0-9_-]*)(?:\s*:\s*(.*))?\s*$`)
)

func (m *Model) readDiagram(file string, diagram *mermaid.DiagramBlock) diagramGraph {
	graph := diagramGraph{}
	lines := strings.Split(diagram.Body, "\n")
	form := diagramForm(lines)
	if !syntaxLine(form) {
		m.fail(Location{file, diagram.Line}, "unsupported Mermaid form; use C4, flowchart, or classDiagram")
		return graph
	}
	for i, line := range lines {
		m.diagramLine(graph, strings.TrimSpace(line), Location{file, diagram.Line + i + 1}, strings.HasPrefix(form, "C4"))
	}
	return graph
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

func syntaxLine(line string) bool {
	return contains([]string{"C4Context", "C4Container", "C4Component", "classDiagram"}, line) ||
		strings.HasPrefix(line, "flowchart ") || strings.HasPrefix(line, "graph ")
}

func decoration(line string) bool {
	if contains([]string{"", "}", "end"}, line) || syntaxLine(line) {
		return true
	}
	for _, prefix := range []string{"%%", "title ", "direction ", "classDef ", "style ", "linkStyle ", "subgraph ", "+", "-", "#", "~"} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func (m *Model) diagramLine(graph diagramGraph, line string, at Location, c4 bool) {
	if decoration(line) {
		return
	}
	if c4 {
		m.c4Statement(graph, line, at)
		return
	}
	if match := edge.FindStringSubmatch(line); match != nil {
		m.addElement(graph, line, relationshipElement(match, at))
		return
	}
	if match := node.FindStringSubmatch(line); match != nil {
		m.addElement(graph, line, element{local: match[1], at: at})
		return
	}
	m.fail(at, "unsupported Mermaid statement %q", line)
}

func relationshipElement(match []string, at Location) element {
	from, to := match[1], match[4]
	if strings.HasPrefix(match[2], "<") {
		from, to = to, from
	}
	return element{from: from, to: to, at: at}
}

func (m *Model) c4Statement(graph diagramGraph, line string, at Location) {
	call := macro.FindStringSubmatch(line)
	if call == nil {
		m.fail(at, "unsupported C4 statement %q", line)
		return
	}
	m.macroLine(graph, call, at)
}

func (m *Model) macroLine(graph diagramGraph, call []string, at Location) {
	r := csv.NewReader(strings.NewReader(call[2]))
	r.TrimLeadingSpace = true
	args, err := r.Read()
	if err != nil || len(args) < 2 {
		m.fail(at, "malformed C4 declaration %q", call[0])
		return
	}
	if contains([]string{"System_Boundary", "Container_Boundary", "Boundary"}, call[1]) {
		return
	}
	if contains([]string{"Rel", "Rel_U", "Rel_D", "Rel_L", "Rel_R"}, call[1]) {
		m.addElement(graph, call[0], element{from: args[0], to: args[1], at: at})
		return
	}
	m.c4Node(graph, call, args, at)
}

func (m *Model) c4Node(graph diagramGraph, call, args []string, at Location) {
	if !c4NodeMacro(call[1]) {
		m.fail(at, "unsupported C4 macro %q", call[1])
		return
	}
	m.addElement(graph, call[0], element{local: args[0], at: at})
}

func c4NodeMacro(name string) bool {
	base := strings.TrimSuffix(name, "_Ext")
	return contains([]string{"Person", "System", "SystemDb", "SystemQueue", "Container", "ContainerDb", "ContainerQueue", "Component", "ComponentDb", "ComponentQueue"}, base)
}

func (m *Model) addElement(graph diagramGraph, line string, value element) {
	numbers := mermaid.ExtractDiagramNumbers(line)
	if len(numbers) != 1 {
		m.fail(value.at, "contracted node or relationship needs exactly one display callout")
		return
	}
	key := strconv.Itoa(numbers[0])
	if previous := graph[key]; previous != nil {
		m.fail(value.at, "duplicate callout %s; first at %s", key, previous.at)
	}
	graph[key] = &value
}

func (m *Model) matchCallout(number string, d Declaration, graph diagramGraph) {
	e := graph[number]
	if e == nil {
		m.fail(d.At, "callout %s has no node or relationship at %s", number, d.Diagram)
		return
	}
	if e.declaration != nil {
		m.fail(d.At, "duplicate table callout %s; first at %s", number, e.declaration.At)
	}
	e.declaration = &d
}

func (m *Model) unclaimed(graph diagramGraph, ids map[string]Declaration) {
	locals := map[string]string{}
	for _, key := range sortedKeys(graph) {
		e := graph[key]
		if e.declaration == nil {
			m.fail(e.at, "callout %s has no contract declaration", key)
			continue
		}
		locals[e.local] = declarationID(*e.declaration)
	}
	for _, key := range sortedKeys(graph) {
		m.elementContract(graph[key], locals, ids)
	}
}

func declarationID(d Declaration) string {
	if d.Ref != "" {
		return d.Ref
	}
	return d.ID
}

func (m *Model) elementContract(e *element, locals map[string]string, ids map[string]Declaration) {
	if e.declaration == nil {
		return
	}
	d, exists := ids[declarationID(*e.declaration)]
	if !exists {
		return
	}
	if (e.from != "") != (d.Type == "relationship") {
		m.fail(e.declaration.At, "callout type disagrees with diagram at %s; declaration at %s", e.at, d.At)
		return
	}
	m.edgeEndpoints(e, locals, d)
}

func (m *Model) edgeEndpoints(e *element, locals map[string]string, d Declaration) {
	if e.from == "" {
		return
	}
	if locals[e.from] != d.From || locals[e.to] != d.To {
		m.fail(e.declaration.At, "relationship endpoints disagree with diagram at %s (%s to %s); declaration at %s", e.at, e.from, e.to, d.At)
	}
}
