package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// flowchart builds a mermaid flowchart doc with the given body lines — the
// TS test helper.
func flowchart(body ...string) string {
	lines := append([]string{"```mermaid", "flowchart TB"}, body...)
	return strings.Join(append(lines, "```"), "\n")
}

func TestValidateDoc(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"every callout node inline-classed with classDefs passes", flowchart(
			"persona((1 Persona)):::persona",
			"screen[2 Screen]:::screen",
			"classDef persona fill:#eef",
			"classDef screen fill:#f8f",
		), nil},
		{"callout node with no style class", flowchart(
			"persona((1 Persona))",
			"classDef persona fill:#eef",
		), []string{"doc.md: callout 1 (persona) has no style class (diagram at line 1)"}},
		{"callouts but no classDef legend", flowchart(
			"persona((1 Persona)):::persona",
		), []string{"doc.md: diagram at line 1 has numbered callouts but no classDef legend"}},
		{"separate class statement styles the node", flowchart(
			"persona((1 Persona))",
			"class persona persona",
			"classDef persona fill:#eef",
		), nil},
		{"non-flowchart diagrams skipped",
			"```mermaid\nclassDiagram\nclass Check[\"1 Check\"]\n```", nil},
		{"legend error precedes unstyled-node errors", flowchart(
			"a[1 First]",
			"b[2 Second]",
		), []string{
			"doc.md: diagram at line 1 has numbered callouts but no classDef legend",
			"doc.md: callout 1 (a) has no style class (diagram at line 1)",
			"doc.md: callout 2 (b) has no style class (diagram at line 1)",
		}},
		{"graph heading counts as a flowchart",
			"```mermaid\ngraph LR\na[1 Node]:::x\nclassDef x fill:#eef\n```", nil},
		{"style statement styles the node", flowchart(
			"a[1 Node]",
			"style a fill:#eef",
			"classDef x fill:#eef",
		), nil},
		{"class statement with multiple comma-separated ids", flowchart(
			"a((1 A))",
			"b[2 B]",
			"class a, b thing",
			"classDef thing fill:#eef",
		), nil},
		{"label not starting with a number is not a callout", flowchart(
			"screen[Screen 2]",
		), nil},
		{"digits glued to the label are not a callout", flowchart(
			"persona((1Persona))",
		), nil},
		{"node shapes: stadium, brace, and flag openers", flowchart(
			"db[(4 System)]",
			"dec{5 Choice}",
			"note>6 Flag]",
		), []string{
			"doc.md: diagram at line 1 has numbered callouts but no classDef legend",
			"doc.md: callout 4 (db) has no style class (diagram at line 1)",
			"doc.md: callout 5 (dec) has no style class (diagram at line 1)",
			"doc.md: callout 6 (note) has no style class (diagram at line 1)",
		}},
		{"blank lines before the flowchart heading still flowchart",
			"```mermaid\n\n  flowchart TB\na[1 X]:::x\nclassDef x fill:#eef\n```", nil},
		{"second diagram reports its own fence line",
			flowchart("a[1 A]:::x", "classDef x fill:#eef") + "\n\n" + flowchart("b[2 B]"),
			[]string{
				"doc.md: diagram at line 7 has numbered callouts but no classDef legend",
				"doc.md: callout 2 (b) has no style class (diagram at line 7)",
			}},
		{"document with no diagrams", "# Just prose\n\nNothing to see.", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateDoc("doc.md", tc.content)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("errors = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunNoRelevantFiles(t *testing.T) {
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunFailsUnstyledCallout(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", flowchart("persona((1 Persona))", "classDef persona fill:#eef"))
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	want := "a.md: callout 1 (persona) has no style class (diagram at line 1)"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %#v, want [%q]", res.Errors, want)
	}
}

func TestRunCountsOnlyMermaidFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "good.md", flowchart("a[1 A]:::x", "classDef x fill:#eef"))
	writeFile(t, root, "plain.md", "# No diagrams here\n")
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
