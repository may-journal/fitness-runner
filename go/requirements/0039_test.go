package requirements

import (
	"os"
	"path/filepath"
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

// mermaidCalloutsFlow reads docs/flow.md from repo.
func mermaidCalloutsFlow(t *testing.T, repo string) string {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(repo, "docs", "flow.md"))
	mustDo(t, err)
	return string(got)
}

// mermaidCalloutsNumbered is a doc whose flowchart and C4 diagrams label
// connectors with numbers beside numbered nodes and a letter in use.
func mermaidCalloutsNumbered() map[string]string {
	return mermaidCalloutsDoc(
		mermaidCalloutsDiagram(
			"flowchart LR",
			`app[1 App] -->|2| api["3 API"]`,
			"api -->|A| db((4 Store))",
			"db -- 6 --> app",
		),
		mermaidCalloutsTable("1", "2", "3", "4", "A", "6"),
		mermaidCalloutsDiagram(`Container(web, "5", "UI")`, `Rel(web, api, "12", "HTTPS")`),
		"| # | Description |\n| --- | --- |\n| 5  | step 5 |\n| 12 | step 12 |\n",
	)
}

func Test0039_14(t *testing.T) {
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
		mermaidCalloutsDiagram(`Person(p, "Person")`, `System(s, "System")`, `Rel(p, s, "uses")`),
		mermaidCalloutsTable("4", "D"),
	))
	sees(t, out, code, 0, "All 1 checks passed")
	if !regexp.MustCompile(`│ passed\s+│ 1\s+│`).MatchString(row(out, "mermaid-callouts")) {
		t.Errorf("mermaid-callouts must pass judging 1 file, got row %q", row(out, "mermaid-callouts"))
	}
}

func Test0039_13(t *testing.T) {
	t.Parallel()
	out, code := mermaidCallouts(t, map[string]string{
		"docs/flow.md": strings.Join([]string{
			mermaidCalloutsDiagram("classDiagram", `class a["1 A"]`, `class b["1 B"]`, "a --> b : A", "b --> a : C"),
			mermaidCalloutsTable("1", "A", "A", "B"),
		}, "\n"),
		"docs/unpaired.md": strings.Join([]string{
			mermaidCalloutsTable("A"),
			mermaidCalloutsDiagram(`Rel(a, b, "A")`),
		}, "\n"),
	})
	sees(t, out, code, 1,
		"docs/flow.md: diagram callout 1 appears 2 times (line 1)",
		"docs/flow.md: callout table row A appears 2 times (line 9)",
		"docs/flow.md: diagram callout C has no matching table row (line 1)",
		"docs/flow.md: callout table row B has no matching diagram callout (line 9)",
		"docs/unpaired.md: callout table at line 1 has no preceding mermaid diagram",
		"docs/unpaired.md: mermaid diagram at line 5 has callouts but no associated callout table")
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
	sees(t, out, code, 1, "docs/flow.md: connector callout 2 must be a letter; numbers are for nodes; run `fitness mermaid-callouts --fix` (line 1)")
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

func Test0039_15(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", mermaidCalloutsNumbered())
	out, code := fitness(t, repo, nil, "mermaid-callouts")
	sees(t, out, code, 1, "connector callout 2 must be a letter")
	out, code = fitness(t, repo, nil, "mermaid-callouts", "--fix")
	sees(t, out, code, 0, "docs/flow.md: converted connector callouts 2 to B, 6 to C, 12 to A", "--fix changed 1 file")
	want := mermaidCalloutsDoc(
		mermaidCalloutsDiagram(
			"flowchart LR",
			`app[1 App] -->|B| api["3 API"]`,
			"api -->|A| db((4 Store))",
			"db -- C --> app",
		),
		"| # | Description |\n| --- | --- |\n| 1 | step 1 |\n| B | step 2 |\n| 3 | step 3 |\n| 4 | step 4 |\n| A | step A |\n| C | step 6 |\n",
		mermaidCalloutsDiagram(`Container(web, "5", "UI")`, `Rel(web, api, "A", "HTTPS")`),
		"| # | Description |\n| --- | --- |\n| 5  | step 5 |\n| A  | step 12 |\n",
	)["docs/flow.md"]
	if got := mermaidCalloutsFlow(t, repo); got != want {
		t.Errorf("--fix must turn each numbered connector and its row into the next free letter, got:\n%s\nwant:\n%s", got, want)
	}
	out, code = fitness(t, repo, nil, "mermaid-callouts")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0039_16(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", mermaidCalloutsNumbered())
	out, code := fitness(t, repo, nil, "mermaid-callouts", "--fix")
	sees(t, out, code, 0, "--fix changed 1 file")
	fixed := mermaidCalloutsFlow(t, repo)
	out, code = fitness(t, repo, nil, "mermaid-callouts", "--fix")
	sees(t, out, code, 0, "--fix changed 0 files")
	if strings.Contains(out, "converted") {
		t.Errorf("a second --fix must list no file, got:\n%s", out)
	}
	if got := mermaidCalloutsFlow(t, repo); got != fixed {
		t.Errorf("a second --fix must change nothing, got:\n%s\nwant:\n%s", got, fixed)
	}
}

func Test0039_17(t *testing.T) {
	t.Parallel()
	files := mermaidCalloutsDoc(
		mermaidCalloutsDiagram(`Container(app, "2", "UI")`, `Rel(app, api, "2")`),
		mermaidCalloutsTable("2"),
	)
	repo := example(t, "happyRepo", files)
	out, code := fitness(t, repo, nil, "mermaid-callouts", "--fix")
	sees(t, out, code, 1,
		"docs/flow.md: --fix left the mermaid diagram at line 1 unchanged: callout 2 marks a node too",
		"--fix changed 0 files",
		"docs/flow.md: connector callout 2 must be a letter")
	if got := mermaidCalloutsFlow(t, repo); got != files["docs/flow.md"] {
		t.Errorf("--fix must leave a diagram it cannot convert byte for byte, got:\n%s", got)
	}
}
