package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// doc wraps a diagram body with a callout table so the diagram is paired
// (prose is only checked when paired) — the TS suite's fixture helper.
func doc(diagramBody string) string {
	return strings.Join([]string{
		"```mermaid",
		diagramBody,
		"```",
		"",
		"| # | Description | Why |",
		"| --- | --- | --- |",
		"| 6 | uses | because |",
	}, "\n")
}

// proseError builds the expected error message for one flagged label.
func proseError(line int, label string) string {
	return fmt.Sprintf(
		`doc.md: diagram at line %d has a relationship label with prose ("%s") — put descriptions in the callout table`,
		line, label)
}

// TestValidateDoc ports the TS suite's cases plus pins for the regex port,
// each extra case verified against the JavaScript original.
func TestValidateDoc(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"passes when Rel labels are numbers only",
			doc(`Rel(app, api, "6")`), []string{}},
		{"flags a Rel label carrying prose beyond the number",
			doc(`Rel(app, api, "6 Uses over HTTP")`),
			[]string{proseError(1, "6 Uses over HTTP")}},
		{"flags a flowchart pipe edge label with prose",
			doc("app -->|6 calls the api| api"),
			[]string{proseError(1, "6 calls the api")}},
		{"flags a colon edge label with prose",
			doc("app --> api : 6 sends data"),
			[]string{proseError(1, "6 sends data")}},
		{"does not flag when the diagram has no paired callout table",
			"```mermaid\nRel(app, api, \"6 Uses\")\n```", []string{}},
		{"does not flag node names (only relationship/edge labels)",
			doc(`Container(app, "6", "SwiftUI")`), []string{}},
		// Pins beyond the TS suite (outputs verified against JS):
		{"flags a BiRel label with prose",
			doc(`BiRel(app, api, "6 talks both ways")`),
			[]string{proseError(1, "6 talks both ways")}},
		{"flags a Rel-suffixed call label with prose",
			doc(`Rel_Back(app, api, "6 replies")`),
			[]string{proseError(1, "6 replies")}},
		{"flags a single-quoted Rel label with prose",
			doc(`Rel(app, api, '6 single quoted')`),
			[]string{proseError(1, "6 single quoted")}},
		{"flags a Rel call spanning lines",
			doc("Rel(app, api,\n  \"6 wrapped prose\")"),
			[]string{proseError(1, "6 wrapped prose")}},
		{"does not flag a bare colon number",
			doc("app --> api : 6"), []string{}},
		{"does not flag a colon label on a comment line",
			doc("%% note : 6 something"), []string{}},
		{"does not flag classDef colors",
			doc("classDef x fill:#6366f1"), []string{}},
		{"does not flag a colon label without a leading number",
			doc("app --> api : uses 6"), []string{}},
		{"does not flag a node label beside a bare pipe number",
			doc(`a["6 node"] -->|6| b`), []string{}},
		{"does not flag Container prose (not a Rel call)",
			doc(`Container(app, "6 App Shell", "SwiftUI")`), []string{}},
		{"orders Rel labels before per-line pipe and colon labels",
			doc("app -->|6 calls the api| api\nRel(a, b, \"6 rel label\")\nc --> d : 6 colon label"),
			[]string{
				proseError(1, "6 rel label"),
				proseError(1, "6 calls the api"),
				proseError(1, "6 colon label"),
			}},
		{"flags both pipe labels on one line",
			doc("a -->|6 first| b -->|6 second| c"),
			[]string{proseError(1, "6 first"), proseError(1, "6 second")}},
		{"flags a pipe label and a colon label on the same line",
			doc("a -->|6 pipe| b : 6 colon tail"),
			[]string{proseError(1, "6 pipe"), proseError(1, "6 colon tail")}},
		{"trims trailing spaces from a colon label",
			doc("app --> api : 6 sends data   "),
			[]string{proseError(1, "6 sends data")}},
		{"reports the diagram's opening-fence line",
			"# Title\n\nprose\n\n" + doc(`Rel(app, api, "6 Uses")`),
			[]string{proseError(5, "6 Uses")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateDoc("doc.md", tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("validateDoc(%q) = %#v, want %#v", tc.content, got, tc.want)
			}
		})
	}
}

// write creates path under root with content, making parent directories.
func write(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunPassesWhenNoRelevantFiles(t *testing.T) {
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 0 || len(res.Errors) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunFailsDocWithProseRelationshipLabel(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", doc(`Rel(app, api, "6 Uses over HTTP")`))
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok {
		t.Fatalf("expected failure, got %+v", res)
	}
	if res.FilesChecked != 1 {
		t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
	}
	want := `a.md: diagram at line 1 has a relationship label with prose ("6 Uses over HTTP") — put descriptions in the callout table`
	if !reflect.DeepEqual(res.Errors, []string{want}) {
		t.Fatalf("errors = %#v, want %#v", res.Errors, []string{want})
	}
}

func TestRunCountsOnlyFilesWithBlocks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "clean.md", doc(`Rel(app, api, "6")`))
	write(t, root, "prose-only.md", "# Title\n\nno mermaid here\n")
	write(t, root, "table-only.md", "| # | D |\n| --- | --- |\n| 1 | a |\n")
	write(t, root, "docs/nested.md", doc("app -->|6 calls the api| api"))
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok {
		t.Fatalf("expected failure from nested doc, got %+v", res)
	}
	if res.FilesChecked != 3 {
		t.Fatalf("filesChecked = %d, want 3 (prose-only file excluded)", res.FilesChecked)
	}
	if len(res.Errors) != 1 || !strings.Contains(res.Errors[0], `docs/nested.md: diagram at line 1`) {
		t.Fatalf("errors = %#v", res.Errors)
	}
}
