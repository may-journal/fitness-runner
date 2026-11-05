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
	for _, n := range requiredLines(content, mdx.BlankFenced(lines)) {
		lines[n] = ""
	}
	return strings.Join(lines, "\n")
}

// requiredLines lists the 0-based lines of content that the architecture
// legend convention or GFM make a doc repeat. Text is content's lines with
// fenced code blanked.
func requiredLines(content string, text []string) []int {
	required := separatorRows(text)
	headings := mdx.HeadingsIn(text)
	doc := mermaid.ReadDoc(content)
	for _, block := range doc.Blocks {
		if d, ok := block.(*mermaid.DiagramBlock); ok {
			required = append(required, diagramLines(doc.Lines, d, mermaid.IsNamedLegend(d, headings))...)
		}
	}
	return required
}

// separatorRows lists the 0-based lines of text that are a GFM table's
// separator row, right under its header row.
func separatorRows(text []string) []int {
	var out []int
	for i := 1; i < len(text); i++ {
		if mdx.IsTableHeader(text[i-1], text[i]) {
			out = append(out, i)
		}
	}
	return out
}

// diagramLines lists the 0-based lines of d's body a doc must repeat: all of
// them for a legend, else its style statements. Lines are the doc's.
func diagramLines(lines []string, d *mermaid.DiagramBlock, legend bool) []int {
	var out []int
	for i := d.Line; d.Body != "" && i <= d.Line+strings.Count(d.Body, "\n"); i++ {
		if legend || mermaid.IsStyleStatement(lines[i]) {
			out = append(out, i)
		}
	}
	return out
}
