package requirements

import (
	"strings"
	"testing"
)

// mermaidLegend runs `fitness-install -- mermaid-legend <args>` on happyRepo
// with files written over it.
func mermaidLegend(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-legend"}, args...)...)
}

// mermaidLegendDiagram is a fenced mermaid diagram with the given lines.
func mermaidLegendDiagram(lines ...string) string {
	return "```mermaid\n" + strings.Join(lines, "\n") + "\n```\n"
}

// mermaidLegendDoc is happyRepo's README followed by diagrams, whose first
// fence sits on line 9.
func mermaidLegendDoc(diagrams ...string) map[string]string {
	return map[string]string{"README.md": readme + "\n" + strings.Join(diagrams, "\n")}
}

func Test0042_1(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(mermaidLegendDiagram(
		"flowchart TB", "persona((1 Persona))", "classDef persona fill:#eef")))
	sees(t, out, code, 1, "README.md: callout 1 (persona) has no style class (diagram at line 9)")
}

func Test0042_2(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(mermaidLegendDiagram(
		"flowchart TB", "persona((1 Persona)):::persona")))
	sees(t, out, code, 1, "README.md: diagram at line 9 has numbered callouts but no classDef legend")
}

func Test0042_3(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(mermaidLegendDiagram(
		"flowchart TB",
		"persona((1 Persona)):::persona",
		"screen[2 Screen]",
		"db[(3 Store)]",
		"class screen, db thing",
		"dec{4 Choice}",
		"style dec fill:#eef",
		"classDef persona fill:#eef",
		"classDef thing fill:#f8f")))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0042_4(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(mermaidLegendDiagram(
		"block-beta", "a[1 Store]", "b[2 Screen]")))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0042_5(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(mermaidLegendDiagram(
		"graph LR", "screen[Screen 2]", "persona((1Persona))")))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0042_6(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, mermaidLegendDoc(
		mermaidLegendDiagram("flowchart TB", "a[1 A]:::x", "classDef x fill:#eef"),
		mermaidLegendDiagram("flowchart TB", "b[2 B]:::x", "classDef y fill:#eef", "c[3 C]")))
	sees(t, out, code, 1, "README.md: callout 3 (c) has no style class (diagram at line 15)")
}

func Test0042_7(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, map[string]string{"body.md": mermaidLegendDiagram(
		"flowchart TB", "persona((1 Persona))", "classDef persona fill:#eef")},
		"--body-file", "body.md")
	sees(t, out, code, 1, "(description): callout 1 (persona) has no style class (diagram at line 1)")
}

func Test0042_8(t *testing.T) {
	t.Parallel()
	out, code := mermaidLegend(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-legend")
}
