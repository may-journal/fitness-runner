package check

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// markdownView blanks the lines a doc must repeat: a legend diagram, a style
// statement inside a mermaid block, and a table separator row. Every line
// stays, so line numbers and the line count hold.
func markdownView(content string) string {
	lines := strings.Split(content, "\n")
	for _, n := range requiredLines(content, lines) {
		lines[n] = ""
	}
	return strings.Join(lines, "\n")
}

// requiredLines lists the 0-based lines of content that mermaid-legend, the
// C4 legend convention or GFM make a doc repeat.
func requiredLines(content string, lines []string) []int {
	var required []int
	blocks := mermaid.ParseDoc(content)
	for i, block := range blocks {
		if d, ok := block.(*mermaid.DiagramBlock); ok {
			required = append(required, diagramLines(d, mermaid.IsLegend(blocks, i))...)
		}
	}
	for i, line := range lines {
		if mdx.IsTableSeparator(line) {
			required = append(required, i)
		}
	}
	return required
}

// diagramLines lists the 0-based lines of d's body a doc must repeat: all of
// them for a legend, else its style statements.
func diagramLines(d *mermaid.DiagramBlock, legend bool) []int {
	var out []int
	for i, line := range strings.Split(d.Body, "\n") {
		if legend || mermaid.IsStyleStatement(line) {
			out = append(out, d.Line+i)
		}
	}
	return out
}
