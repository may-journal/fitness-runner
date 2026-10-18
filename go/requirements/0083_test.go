// cspell:ignore Descr mindmap xychart

package requirements

import "testing"

// mermaidSyntax runs `fitness-install -- mermaid-syntax <args>` on happyRepo
// with files written over it.
func mermaidSyntax(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-syntax"}, args...)...)
}

func Test0083_1(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(
		mermaidLegendDiagram("flowchart LR", `A["Start (now)"]:::s --> B{Ready?} -->|yes| C((Done))`,
			"B -.-> D & E", "subgraph S [Group]", "direction TB", "F -- note --> G", "end", "classDef s fill:#eef"),
		mermaidLegendDiagram("graph TD;", "A-->B;"),
		mermaidLegendDiagram("sequenceDiagram", "participant A as Client", "loop every minute",
			"A->>+B: ping", "alt ok", "B-->>-A: pong", "else down", "Note over A,B: retry", "end", "end"),
		mermaidLegendDiagram("C4Context", `Person(user, "1 User")`, `Rel(user, app, "2")`),
		mermaidLegendDiagram("flowchart-elk TD", "accDescr {", "free text: here", "}", "A-- yes -->B"),
		mermaidLegendDiagram("sequenceDiagram", "par_over A", "A->>B: x", "and B", "B->>A: y", "end"),
		mermaidLegendDiagram("mindmap", "root", "  a)cloud("),
		mermaidLegendDiagram("xychart", "bar [1, 2]")))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0083_2(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(mermaidLegendDiagram("flowchart TD", "A[Start (x)] --> B")))
	sees(t, out, code, 1, `README.md: line 11: node A: label "Start (x)" has brackets or quotes; wrap it in double quotes (diagram at line 9)`)
}

func Test0083_3(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(mermaidLegendDiagram("flowchart TD", "A -->", "subgraph S", "B --> C")))
	sees(t, out, code, 1, "README.md: line 11: an edge has no target node (diagram at line 9)",
		"README.md: line 12: subgraph is never closed with end (diagram at line 9)")
}

func Test0083_4(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(mermaidLegendDiagram("sequenceDiagram", "A->>B", "loop every minute", "A->>B: hi")))
	sees(t, out, code, 1, `README.md: line 11: message "A->>B" needs a target and a colon`,
		"README.md: line 12: loop is never closed with end (diagram at line 9)")
}

func Test0083_5(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(mermaidLegendDiagram("graf TD", "A --> B")))
	sees(t, out, code, 1, `README.md: line 10: unknown diagram type "graf" (diagram at line 9)`)
}

func Test0083_6(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, mermaidLegendDoc(mermaidLegendDiagram("C4Context", `Person(user, "1 User")`, `Rel(user, app, "2"`)))
	sees(t, out, code, 1, "README.md: line 12: '(' is never closed (diagram at line 9)")
}

func Test0083_7(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, map[string]string{"body.md": mermaidLegendDiagram("flowchart TD", "A[Start (x)] --> B")},
		"--body-file", "body.md")
	sees(t, out, code, 1, `(description): line 3: node A: label "Start (x)"`)
}

func Test0083_8(t *testing.T) {
	t.Parallel()
	out, code := mermaidSyntax(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-syntax")
}
