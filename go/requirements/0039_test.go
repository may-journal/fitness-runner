package requirements

import (
	"regexp"
	"strings"
	"testing"
)

// mermaidCallouts runs `fitness-install -- mermaid-callouts <args>` on
// happyRepo with files written over it.
func mermaidCallouts(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-callouts"}, args...)...)
}

// mermaidCalloutsDiagram fences mermaid lines as a diagram block.
func mermaidCalloutsDiagram(lines ...string) string {
	return "```mermaid\n" + strings.Join(lines, "\n") + "\n```\n"
}

// mermaidCalloutsTable builds a numbered callout table with one row per number.
func mermaidCalloutsTable(numbers ...string) string {
	table := "| # | Description |\n| --- | --- |\n"
	for _, n := range numbers {
		table += "| " + n + " | step " + n + " |\n"
	}
	return table
}

// mermaidCalloutsDoc joins diagram and table blocks into one doc.
func mermaidCalloutsDoc(blocks ...string) map[string]string {
	return map[string]string{"docs/flow.md": strings.Join(blocks, "\n")}
}

func Test0039_1(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Container(app, "1", "UI")`, `Rel(app, api, "2")`),
		mermaidCalloutsTable("1", "2"),
	))
	sees(t, out, code, 0, "All 1 checks passed")
	if !regexp.MustCompile(`│ passed\s+│ 1\s+│`).MatchString(row(out, "mermaid-callouts")) {
		t.Errorf("mermaid-callouts must pass judging 1 file, got row %q", row(out, "mermaid-callouts"))
	}
}

func Test0039_2(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Rel(a, b, "1")`, `Rel(b, c, "2")`),
		mermaidCalloutsTable("1"),
	))
	sees(t, out, code, 1, "docs/flow.md: diagram callout 2 has no matching table row (line 1)")
}

func Test0039_3(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Rel(a, b, "1")`),
		mermaidCalloutsTable("1", "2"),
	))
	sees(t, out, code, 1, "docs/flow.md: callout table row 2 has no matching diagram callout (line 5)")
}

func Test0039_4(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Rel(a, b, "1")`, `Rel(a, c, "1")`, `Rel(b, c, "2")`),
		mermaidCalloutsTable("1", "2", "2"),
	))
	sees(t, out, code, 1,
		"docs/flow.md: diagram callout 1 appears 2 times (line 1)",
		"docs/flow.md: callout table row 2 appears 2 times (line 7)")
}

func Test0039_5(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(mermaidCalloutsDiagram(`Rel(a, b, "1")`)))
	sees(t, out, code, 1, "docs/flow.md: mermaid diagram at line 1 has numbered callouts but no associated callout table")
}

func Test0039_6(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(mermaidCalloutsTable("1")))
	sees(t, out, code, 1, "docs/flow.md: callout table at line 1 has no preceding mermaid diagram")
}

func Test0039_7(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Container(app, "1", "UI")`, `Rel(app, api, "2")`),
		mermaidCalloutsDiagram(`Person(p, "Person")`, `System(s, "System")`),
		mermaidCalloutsTable("1", "2"),
	))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0039_8(t *testing.T) {
	t.Parallel()
	body := mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Rel(a, b, "1")`, `Rel(b, c, "2")`),
		mermaidCalloutsTable("1"),
	)["docs/flow.md"]
	out, code := mermaidCallouts(t, map[string]string{"body.md": body}, "--body-file", "body.md")
	sees(t, out, code, 1, "(description): diagram callout 2 has no matching table row (line 1)")
}
