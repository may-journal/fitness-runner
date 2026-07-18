// Command fitness-check-mermaid-legend validates that numbered flowchart
// callout nodes carry a style class (inline `:::class` or a `class`/`style`
// statement) and that diagrams with callouts declare a `classDef` legend —
// the Go port of the mermaid-legend check. C4/other diagram types (different
// styling model) are skipped.
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-legend"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errors, filesChecked, err := mermaid.RunDocCheck(root, validateDoc)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) > 0 {
		return checkkit.Fail(filesChecked, errors...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

// jsWS is the JavaScript \s character class as an RE2 class body — a copy of
// internal/mermaid's unexported constant, so the ported patterns match
// exactly what the JS regexes matched.
const jsWS = `\t\n\v\f\r \x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`

// isJSSpace reports whether r is in the JavaScript \s class — the rune-level
// twin of jsWS.
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// trimJS trims JavaScript whitespace from both ends — String.prototype.trim.
func trimJS(s string) string { return strings.TrimFunc(s, isJSSpace) }

var (
	// flowchartLine is the TS FLOWCHART pattern: a flowchart/graph heading.
	flowchartLine = regexp.MustCompile(`^[` + jsWS + `]*(flowchart|graph)\b`)
	// classDefLine is the TS CLASSDEF pattern: a classDef legend declaration.
	classDefLine = regexp.MustCompile(`^[` + jsWS + `]*classDef[` + jsWS + `]+\w+`)
	// nodeDef is the TS NODE_DEF pattern: a node definition whose label opens
	// with a callout number — id, shape opener, then digits at a boundary.
	nodeDef = regexp.MustCompile(
		`^[` + jsWS + `]*(\w[\w-]*)[` + jsWS + `]*(?:\(\(|\[\(|\[|\(|\{|>)[` + jsWS + `]*(\d+)\b`)
	// classStmt is the TS CLASS_STMT pattern: `class id1,id2 styleClass`.
	classStmt = regexp.MustCompile(
		`^[` + jsWS + `]*class[` + jsWS + `]+([\w,` + jsWS + `-]+?)[` + jsWS + `]+\w+[` + jsWS + `]*$`)
	// styleStmt is the TS STYLE_STMT pattern: `style id …`.
	styleStmt = regexp.MustCompile(`^[` + jsWS + `]*style[` + jsWS + `]+(\w[\w-]*)\b`)
	// inlineClass is the TS INLINE_CLASS pattern: a `:::class` inline style.
	inlineClass = regexp.MustCompile(`:::\w+`)
)

// numberedNode is one flowchart node whose label starts with a callout number.
type numberedNode struct {
	id           string
	number       int
	styledInline bool
}

// isFlowchart reports whether a diagram body is a flowchart/graph (per its
// first non-blank line).
func isFlowchart(lines []string) bool {
	for _, line := range lines {
		if trimJS(line) != "" {
			return flowchartLine.MatchString(line)
		}
	}
	return false
}

// collectStyledIds returns the node ids given a style class via
// `class`/`style` statements.
func collectStyledIds(lines []string) map[string]bool {
	styledIds := map[string]bool{}
	for _, line := range lines {
		if m := classStmt.FindStringSubmatch(line); m != nil {
			for _, id := range strings.Split(m[1], ",") {
				styledIds[trimJS(id)] = true
			}
		}
		if m := styleStmt.FindStringSubmatch(line); m != nil {
			styledIds[m[1]] = true
		}
	}
	return styledIds
}

// collectNumberedNodes returns the numbered callout nodes defined in a
// flowchart body.
func collectNumberedNodes(lines []string) []numberedNode {
	var nodes []numberedNode
	for _, line := range lines {
		if m := nodeDef.FindStringSubmatch(line); m != nil {
			number, _ := strconv.Atoi(m[2])
			nodes = append(nodes, numberedNode{
				id:           m[1],
				number:       number,
				styledInline: inlineClass.MatchString(line),
			})
		}
	}
	return nodes
}

// unstyledNodeErrors reports numbered nodes carrying no style class (inline
// or via a statement).
func unstyledNodeErrors(
	file string, block *mermaid.DiagramBlock, nodes []numberedNode, styledIds map[string]bool,
) []string {
	var errors []string
	for _, node := range nodes {
		if !node.styledInline && !styledIds[node.id] {
			errors = append(errors, fmt.Sprintf(
				"%s: callout %d (%s) has no style class (diagram at line %d)",
				file, node.number, node.id, block.Line))
		}
	}
	return errors
}

// hasClassDef reports whether any line declares a classDef legend entry.
func hasClassDef(lines []string) bool {
	for _, line := range lines {
		if classDefLine.MatchString(line) {
			return true
		}
	}
	return false
}

// legendErrorsForDiagram returns the legend/styling errors for one flowchart
// diagram that has numbered callouts.
func legendErrorsForDiagram(file string, block *mermaid.DiagramBlock) []string {
	lines := strings.Split(block.Body, "\n")
	if !isFlowchart(lines) {
		return nil
	}
	nodes := collectNumberedNodes(lines)
	if len(nodes) == 0 {
		return nil
	}
	styledIds := collectStyledIds(lines)
	var errors []string
	if !hasClassDef(lines) {
		errors = append(errors, fmt.Sprintf(
			"%s: diagram at line %d has numbered callouts but no classDef legend", file, block.Line))
	}
	return append(errors, unstyledNodeErrors(file, block, nodes, styledIds)...)
}

// validateDoc flags numbered flowchart callout nodes that carry no style
// class, and diagrams with callouts but no classDef legend — so callouts
// render consistently and map to a legend.
func validateDoc(file, content string) []string {
	var errors []string
	for _, block := range mermaid.ParseDoc(content) {
		if diagram, ok := block.(*mermaid.DiagramBlock); ok {
			errors = append(errors, legendErrorsForDiagram(file, diagram)...)
		}
	}
	return errors
}
