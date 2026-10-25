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

// mermaidCalloutsTable builds a callout table with one row per label.
func mermaidCalloutsTable(labels ...string) string {
	table := "| # | Description |\n| --- | --- |\n"
	for _, n := range labels {
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
		mermaidCalloutsDiagram(
			"flowchart LR",
			`app[1 App] -->|A| api["2 API"]`,
			"api -- B --> db((3 Store))",
			`db -->|"C"| app`,
			"db -->|YES| app -->|go| Q --> db",
		),
		mermaidCalloutsTable("1", "2", "3", "A", "B", "C"),
		mermaidCalloutsDiagram(`Container(web, "4", "UI")`, `Rel(web, api, "D", "HTTPS")`),
		mermaidCalloutsTable("4", "D"),
	))
	sees(t, out, code, 0, "All 1 checks passed")
	if !regexp.MustCompile(`│ passed\s+│ 1\s+│`).MatchString(row(out, "mermaid-callouts")) {
		t.Errorf("mermaid-callouts must pass judging 1 file, got row %q", row(out, "mermaid-callouts"))
	}
}

func Test0039_9(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Person(user, "1 User")`, `Rel(user, app, "A")`),
		mermaidCalloutsTable("1", "B"),
	))
	sees(t, out, code, 1,
		"docs/flow.md: diagram callout A has no matching table row (line 1)",
		"docs/flow.md: callout table row B has no matching diagram callout (line 6)")
}

func Test0039_4(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram("classDiagram", `class a["1 A"]`, `class b["1 B"]`, "a --> b : A", "b --> a : B"),
		mermaidCalloutsTable("1", "A", "B", "B"),
	))
	sees(t, out, code, 1,
		"docs/flow.md: diagram callout 1 appears 2 times (line 1)",
		"docs/flow.md: callout table row B appears 2 times (line 9)")
}

func Test0039_10(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsTable("A"),
		mermaidCalloutsDiagram(`Rel(a, b, "A")`),
	))
	sees(t, out, code, 1,
		"docs/flow.md: callout table at line 1 has no preceding mermaid diagram",
		"docs/flow.md: mermaid diagram at line 5 has callouts but no associated callout table")
}

func Test0039_7(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Container(app, "1", "UI")`, `Rel(app, api, "A")`),
		mermaidCalloutsDiagram(`Person(p, "Person")`, `System(s, "System")`, `Rel(p, s, "uses")`),
		mermaidCalloutsTable("1", "A"),
	))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0039_8(t *testing.T) {
	t.Parallel()
	body := mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Rel(a, b, "A")`, `Rel(b, c, "B")`),
		mermaidCalloutsTable("A"),
	)["docs/flow.md"]
	out, code := mermaidCallouts(t, map[string]string{"body.md": body}, "--body-file", "body.md")
	sees(t, out, code, 1, "(description): diagram callout B has no matching table row (line 1)")
}

func Test0039_11(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Container(app, "1", "UI")`, `Rel(app, api, "2")`),
		mermaidCalloutsTable("1", "2"),
	))
	sees(t, out, code, 1, "docs/flow.md: connector callout 2 must be a letter; numbers are for nodes (line 1)")
	if strings.Contains(out, "has no matching") {
		t.Errorf("a numbered connector must fail only for its form, not as an orphan:\n%s", out)
	}
}

func Test0039_12(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, mermaidCalloutsDoc(
		mermaidCalloutsDiagram("flowchart LR", "a -->|Z| b", "b -->|AA| c", "c -->|AB| a"),
		mermaidCalloutsTable("Z", "AA", "AB"),
	))
	sees(t, out, code, 1,
		"docs/flow.md: connector callout AB breaks the letter doubling; use A to Z, then AA to ZZ, then AAA (line 1)",
		"docs/flow.md: callout table row AB breaks the letter doubling; use A to Z, then AA to ZZ, then AAA (line 8)")
	if strings.Contains(out, "callout AA ") || strings.Contains(out, "row AA ") {
		t.Errorf("AA follows the doubling and must pass:\n%s", out)
	}
}
