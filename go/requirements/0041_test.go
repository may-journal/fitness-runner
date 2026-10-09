package requirements

import (
	"strings"
	"testing"
)

// tableGap runs `fitness-install -- mermaid-diagram-table-gap <args>` on
// happyRepo with files written over it.
func tableGap(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-diagram-table-gap"}, args...)...)
}

// tableGapDoc joins parts into a doc, one blank line between each.
func tableGapDoc(parts ...string) string {
	return "# Doc\n\n" + strings.Join(parts, "\n\n") + "\n"
}

// The numbered diagram, its legend, the caption, and a matching callout
// table that each Given arranges around the diagram-to-table gap.
const (
	tableGapDiagram = "```mermaid\nContainer(app, \"1\", \"UI\")\nRel(app, api, \"2\")\n```"
	tableGapLegend  = "```mermaid\nPerson(p, \"Person\")\nSystem(s, \"System\")\n```"
	tableGapCaption = "Numbers on nodes and arrows match the callout table."
	tableGapTable   = "| # | Description | Why |\n| --- | --- | --- |\n| 1 | the app | because |\n| 2 | talks to | because |"
	tableGapProse   = "Four containers sit inside one app."
)

// tableGapError is the message a reader sees for prose at line in file.
func tableGapError(file, line string) string {
	return file + ": prose between the diagram and its callout table (line " + line + ") — move detail into the callout table"
}

func Test0041_1(t *testing.T) {
	t.Parallel()
	out, code := tableGap(t, map[string]string{"doc.md": tableGapDoc(tableGapDiagram, tableGapProse, tableGapTable)})
	sees(t, out, code, 1, tableGapError("doc.md", "8"))
}

func Test0041_2(t *testing.T) {
	t.Parallel()
	doc := tableGapDoc(tableGapDiagram, tableGapLegend, tableGapProse, tableGapTable)
	out, code := tableGap(t, map[string]string{"doc.md": doc})
	sees(t, out, code, 1, tableGapError("doc.md", "13"))
}

func Test0041_3(t *testing.T) {
	t.Parallel()
	out, code := tableGap(t, map[string]string{
		"a.md": tableGapDoc(tableGapDiagram, tableGapLegend, tableGapCaption, tableGapTable),
		"b.md": tableGapDoc(tableGapDiagram, "Numbers on classes and relationships match the callout table.", tableGapTable),
		"c.md": tableGapDoc(tableGapDiagram, `Numbers match the callout table. "Screen" is a terminal state.`, tableGapTable),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0041_4(t *testing.T) {
	t.Parallel()
	doc := tableGapDoc(tableGapDiagram, tableGapLegend, tableGapCaption, tableGapProse, tableGapTable)
	out, code := tableGap(t, map[string]string{"doc.md": doc})
	sees(t, out, code, 1, tableGapError("doc.md", "15"))
	if strings.Contains(read(out), read(tableGapError("doc.md", "13"))) {
		t.Errorf("the caption at line 13 must not be flagged:\n%s", out)
	}
}

func Test0041_5(t *testing.T) {
	t.Parallel()
	out, code := tableGap(t, map[string]string{
		"a.md": tableGapDoc("Intro prose before the diagram.", tableGapDiagram, tableGapCaption, tableGapTable, "Trailing prose after the table."),
		"b.md": tableGapDoc("Prose above.", "```mermaid\nContainer(app, \"UI\")\n```", "Prose below."),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0041_6(t *testing.T) {
	t.Parallel()
	out, code := tableGap(t, map[string]string{"doc.md": tableGapDoc("Just prose, nothing to see here.")})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-diagram-table-gap")
}

func Test0041_7(t *testing.T) {
	t.Parallel()
	body := strings.Join([]string{tableGapDiagram, tableGapProse, tableGapTable}, "\n\n") + "\n"
	out, code := tableGap(t, map[string]string{"body.md": body}, "--body-file", "body.md")
	sees(t, out, code, 1, tableGapError("(description)", "6"))
}
