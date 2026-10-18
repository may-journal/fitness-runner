// cspell:ignore Descr autonumber
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// directions are the flowchart and subgraph directions mermaid accepts.
var directions = map[string]bool{"TB": true, "TD": true, "BT": true, "RL": true, "LR": true}

// flowchart parses a flowchart or graph: header, then node and edge chains,
// subgraph blocks, and styling statements.
func flowchart(header srcLine, lines []srcLine) []problem {
	head := splitStatements(header.text)
	problems := flowHeader(header.n, head[0])
	f := &flow{}
	for _, stmt := range head[1:] {
		f.statement(header.n, stmt)
	}
	for _, l := range lines {
		for _, stmt := range splitStatements(l.text) {
			f.statement(l.n, stmt)
		}
	}
	for _, n := range f.open {
		f.fail(n, "subgraph is never closed with end")
	}
	return append(problems, f.problems...)
}

// flowHeader checks the optional direction after flowchart or graph.
func flowHeader(n int, text string) []problem {
	fields := strings.Fields(text)
	if len(fields) == 1 || len(fields) == 2 && directions[fields[1]] {
		return nil
	}
	return []problem{{n, fmt.Sprintf("%q is not a flowchart header; use flowchart TD, LR, BT, or RL", text)}}
}

// flow is a flowchart parse in progress.
type flow struct {
	open     []int
	problems []problem
}

func (f *flow) fail(n int, msg string) {
	f.problems = append(f.problems, problem{n, msg})
}

// keywordArgs are the styling statements and the fields each needs.
var keywordArgs = map[string]int{
	"classDef": 3, "class": 3, "style": 3, "linkStyle": 3, "click": 2,
	"accTitle:": 1, "accDescr:": 1, "accDescr": 1,
}

// statement parses one flowchart statement on line n.
func (f *flow) statement(n int, stmt string) {
	fields := strings.Fields(stmt)
	switch fields[0] {
	case "subgraph":
		f.subgraph(n, stmt)
	case "end":
		f.end(n)
	case "direction":
		f.direction(n, fields)
	default:
		f.other(n, stmt, fields)
	}
}

// subgraph opens a block; its title only needs balanced brackets.
func (f *flow) subgraph(n int, stmt string) {
	f.open = append(f.open, n)
	for _, p := range balance([]srcLine{{n, stmt}}) {
		f.fail(p.line, "subgraph title: "+p.msg)
	}
}

// end closes the innermost subgraph.
func (f *flow) end(n int) {
	if len(f.open) == 0 {
		f.fail(n, "end closes no subgraph")
		return
	}
	f.open = f.open[:len(f.open)-1]
}

func (f *flow) direction(n int, fields []string) {
	if len(fields) != 2 || !directions[fields[1]] {
		f.fail(n, "direction needs one of TB, TD, BT, RL, LR")
	}
}

// other parses a styling statement or a node and edge chain.
func (f *flow) other(n int, stmt string, fields []string) {
	if need, ok := keywordArgs[fields[0]]; ok {
		if len(fields) < need {
			f.fail(n, fmt.Sprintf("%s needs %d parts", fields[0], need))
		}
		return
	}
	if msg := chain(stmt); msg != "" {
		f.fail(n, msg)
	}
}

// splitStatements splits text on semicolons outside quotes and brackets,
// dropping empty statements.
func splitStatements(text string) []string {
	var out []string
	depth, quoted, start := 0, false, 0
	for i, r := range text {
		depth, quoted = track(r, depth, quoted)
		if r == ';' && depth == 0 && !quoted {
			out = appendStmt(out, text[start:i])
			start = i + 1
		}
	}
	return appendStmt(out, text[start:])
}

// track updates bracket depth and quote state for one rune.
func track(r rune, depth int, quoted bool) (int, bool) {
	switch {
	case r == '"':
		return depth, !quoted
	case quoted:
		return depth, quoted
	case strings.ContainsRune("([{", r):
		return depth + 1, quoted
	case strings.ContainsRune(")]}", r):
		return depth - 1, quoted
	}
	return depth, quoted
}

func appendStmt(out []string, s string) []string {
	if s = strings.TrimSpace(s); s != "" {
		out = append(out, s)
	}
	return out
}

// scanner walks one chain statement.
type scanner struct {
	s string
	i int
}

func (sc *scanner) rest() string { return sc.s[sc.i:] }
func (sc *scanner) done() bool   { return sc.i >= len(sc.s) }
func (sc *scanner) space()       { sc.i = len(sc.s) - len(strings.TrimLeft(sc.rest(), " \t")) }

// near quotes the next few characters for an error message.
func (sc *scanner) near() string {
	r := []rune(sc.rest())
	if len(r) > 12 {
		r = r[:12]
	}
	return string(r)
}

// chain parses `node (edge node)*`, where each node may be an `&` group.
func chain(stmt string) string {
	sc := &scanner{s: stmt}
	msg := sc.group()
	for msg == "" && !sc.done() {
		msg = sc.link()
	}
	return msg
}

// link parses one edge and its target group.
func (sc *scanner) link() string {
	if msg := sc.edge(); msg != "" {
		return msg
	}
	sc.space()
	if sc.done() {
		return "an edge has no target node"
	}
	return sc.group()
}

// group parses `node (& node)*` and the space after it.
func (sc *scanner) group() string {
	msg := sc.node()
	sc.space()
	for msg == "" && strings.HasPrefix(sc.rest(), "&") {
		sc.i++
		sc.space()
		msg = sc.node()
		sc.space()
	}
	return msg
}

// nodeID is a node id: letters, digits, and underscores, with single
// hyphens between them.
var nodeID = regexp.MustCompile(`^[\p{L}\p{N}_]+(?:-[\p{L}\p{N}_]+)*`)

// classSuffix is an inline `:::class` style.
var classSuffix = regexp.MustCompile(`^:::[\w-]+`)

// node parses an id, an optional shape, and an optional :::class.
func (sc *scanner) node() string {
	id := nodeID.FindString(sc.rest())
	if id == "" {
		return fmt.Sprintf("expected a node, got %q", sc.near())
	}
	sc.i += len(id)
	if msg := sc.shape(); msg != "" {
		return fmt.Sprintf("node %s: %s", id, msg)
	}
	sc.i += len(classSuffix.FindString(sc.rest()))
	return ""
}

// shapeKind is one node shape's opener and the closers that end it.
type shapeKind struct {
	open   string
	closes []string
}

// shapes are the node shapes, longest opener first so `((` wins over `(`.
var shapes = []shapeKind{
	{"@{", []string{"}"}}, {"(((", []string{")))"}}, {"([", []string{"])"}}, {"[[", []string{"]]"}},
	{"[(", []string{")]"}}, {"((", []string{"))"}}, {"{{", []string{"}}"}},
	{"[/", []string{"/]", `\]`}}, {`[\`, []string{`\]`, "/]"}},
	{"[", []string{"]"}}, {"(", []string{")"}}, {"{", []string{"}"}}, {">", []string{"]"}},
}

// shape parses the shape after a node id, if any.
func (sc *scanner) shape() string {
	for _, k := range shapes {
		if strings.HasPrefix(sc.rest(), k.open) {
			sc.i += len(k.open)
			return sc.label(k)
		}
	}
	return ""
}

// label parses a shape's text and closer: quoted text may hold anything,
// unquoted text no brackets or quotes.
func (sc *scanner) label(k shapeKind) string {
	if strings.HasPrefix(strings.TrimLeft(sc.rest(), " "), `"`) {
		return sc.quotedLabel(k)
	}
	end, closer := firstCloser(sc.rest(), k.closes)
	if end < 0 {
		return fmt.Sprintf("shape %s is never closed", k.open)
	}
	if text := sc.rest()[:end]; strings.ContainsAny(text, `[](){}"`) && k.open != "@{" {
		return fmt.Sprintf("label %q has brackets or quotes; wrap it in double quotes", text)
	}
	sc.i += end + len(closer)
	return ""
}

// quotedLabel parses `"text"` then the shape's closer.
func (sc *scanner) quotedLabel(k shapeKind) string {
	sc.space()
	end := strings.IndexByte(sc.rest()[1:], '"')
	if end < 0 {
		return "a quoted label is never closed"
	}
	sc.i += end + 2
	sc.space()
	for _, c := range k.closes {
		if strings.HasPrefix(sc.rest(), c) {
			sc.i += len(c)
			return ""
		}
	}
	return fmt.Sprintf("expected %s after the quoted label, got %q", k.closes[0], sc.near())
}

// firstCloser finds the earliest of closes in s.
func firstCloser(s string, closes []string) (int, string) {
	best, closer := -1, ""
	for _, c := range closes {
		if i := strings.Index(s, c); i >= 0 && (best < 0 || i < best) {
			best, closer = i, c
		}
	}
	return best, closer
}

// textEdgeOpen starts an edge with its label inline: `-- text -->`.
var textEdgeOpen = regexp.MustCompile(`^(?:--|==|-\.)\s`)

// textEdgeClose ends an inline edge label.
var textEdgeClose = regexp.MustCompile(`\s(?:-{2,}|={2,}|\.-+)(?:>|[ox](?:\s|$))?`)

// edgeLine is an edge's line: dashes, thick, dotted, or invisible, with an
// optional edge id and start marker.
var edgeLine = regexp.MustCompile(`^(?:\w+@)?[<ox]?(?:-{2,}|={2,}|-\.+-|~{3,})`)

// edge parses one edge, with its optional `|label|`.
func (sc *scanner) edge() string {
	if textEdgeOpen.MatchString(sc.rest()) {
		return sc.textEdge()
	}
	line := edgeLine.FindString(sc.rest())
	if line == "" {
		return fmt.Sprintf("expected an edge such as --> after a node, got %q", sc.near())
	}
	sc.i += len(line)
	sc.arrowHead()
	return sc.pipeLabel()
}

// textEdge parses `-- text -->` and its kin.
func (sc *scanner) textEdge() string {
	loc := textEdgeClose.FindStringIndex(sc.rest()[2:])
	if loc == nil {
		return fmt.Sprintf("edge label after %s is never closed with an arrow", sc.rest()[:2])
	}
	sc.i += 2 + loc[1]
	return ""
}

// arrowHead consumes `>`, or `o` or `x` when a space or label follows.
func (sc *scanner) arrowHead() {
	if arrowTip.MatchString(sc.rest()) {
		sc.i++
	}
}

// arrowTip is an arrowhead: `>`, or `o` or `x` before a space, label, or end.
var arrowTip = regexp.MustCompile(`^(?:>|[ox](?:[ \t|]|$))`)

// pipeLabel consumes an optional `|label|` after an edge.
func (sc *scanner) pipeLabel() string {
	sc.space()
	if !strings.HasPrefix(sc.rest(), "|") {
		return ""
	}
	end := strings.IndexByte(sc.rest()[1:], '|')
	if end < 0 {
		return "edge label opened with | is never closed"
	}
	sc.i += end + 2
	return ""
}
