package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// mermaidFence wraps a mermaid diagram body in a fenced block.
func mermaidFence(body string) string {
	return "```mermaid\n" + body + "\n```"
}

// calloutTable builds a numbered callout table from #/description row pairs.
func calloutTable(rows [][2]string) string {
	lines := []string{"| # | Description | Why |", "| --- | --- | --- |"}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("| %s | %s | because |", row[0], row[1]))
	}
	return strings.Join(lines, "\n")
}

// write creates relPath under dir with content, creating parent dirs.
func write(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runOn runs the check over dir and fails the test on a crash.
func runOn(t *testing.T, dir string) checkkit.Result {
	t.Helper()
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRunTableDriven(t *testing.T) {
	cases := []struct {
		name             string
		files            map[string]string
		ok               bool
		wantFilesChecked int
		wantErrors       []string
	}{
		{
			name:             "passes when there are no markdown files",
			files:            map[string]string{},
			ok:               true,
			wantFilesChecked: 0,
			wantErrors:       []string{},
		},
		{
			name:             "skips markdown with no diagram or callout table",
			files:            map[string]string{"doc.md": "# Just prose\n\nNothing to see here."},
			ok:               true,
			wantFilesChecked: 0,
			wantErrors:       []string{},
		},
		{
			name: "passes when diagram and table callouts match 1-1",
			files: map[string]string{
				"architecture/01-context.md": strings.Join([]string{
					mermaidFence("Container(app, \"1\", \"UI\")\nContainer(api, \"2\", \"API\")\nRel(app, api, \"3\")"),
					"",
					calloutTable([][2]string{{"1", "the app"}, {"2", "the api"}, {"3", "talks to"}}),
				}, "\n"),
			},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name: "fails when a diagram callout has no matching table row",
			files: map[string]string{
				"doc.md": strings.Join([]string{
					mermaidFence("Rel(a, b, \"1\")\nRel(b, c, \"2\")"),
					"",
					calloutTable([][2]string{{"1", "x"}}),
				}, "\n"),
			},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors:       []string{"doc.md: diagram callout 2 has no matching table row (line 1)"},
		},
		{
			name: "fails when a table row has no matching diagram callout",
			files: map[string]string{
				"doc.md": strings.Join([]string{
					mermaidFence(`Rel(a, b, "1")`),
					"",
					calloutTable([][2]string{{"1", "x"}, {"2", "orphan"}}),
				}, "\n"),
			},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors:       []string{"doc.md: callout table row 2 has no matching diagram callout (line 5)"},
		},
		{
			name:             "fails when a numbered diagram has no associated table",
			files:            map[string]string{"doc.md": mermaidFence(`Rel(a, b, "1")`)},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors: []string{
				"doc.md: mermaid diagram at line 1 has numbered callouts but no associated callout table",
			},
		},
		{
			name:             "fails when a callout table has no preceding diagram",
			files:            map[string]string{"doc.md": calloutTable([][2]string{{"1", "x"}})},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors:       []string{"doc.md: callout table at line 1 has no preceding mermaid diagram"},
		},
		{
			name: "fails on duplicate callout numbers on either side",
			files: map[string]string{
				"diagram-dup.md": strings.Join([]string{
					mermaidFence("Rel(a, b, \"1\")\nRel(a, c, \"1\")"),
					"",
					calloutTable([][2]string{{"1", "x"}}),
				}, "\n"),
				"table-dup.md": strings.Join([]string{
					mermaidFence(`Rel(a, b, "1")`),
					"",
					calloutTable([][2]string{{"1", "x"}, {"1", "again"}}),
				}, "\n"),
			},
			ok:               false,
			wantFilesChecked: 2,
			wantErrors: []string{
				"diagram-dup.md: diagram callout 1 appears 2 times (line 1)",
				"table-dup.md: callout table row 1 appears 2 times (line 5)",
			},
		},
		{
			name:             "passes for an un-numbered diagram with no table",
			files:            map[string]string{"doc.md": mermaidFence("Container(app, \"SwiftUI\")\nContainer(api, \"Vapor\")")},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name: "passes when a legend diagram sits between the numbered diagram and its table",
			files: map[string]string{
				"architecture/01-context.md": strings.Join([]string{
					mermaidFence("Container(app, \"1\", \"UI\")\nRel(app, api, \"2\")"),
					"",
					mermaidFence("Person(p, \"Person\")\nSystem(s, \"System\")"), // legend: no callouts
					"",
					calloutTable([][2]string{{"1", "the app"}, {"2", "talks to"}}),
				}, "\n"),
			},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for relPath, content := range tc.files {
				write(t, dir, relPath, content)
			}
			res := runOn(t, dir)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != tc.wantFilesChecked {
				t.Fatalf("filesChecked = %d, want %d", res.FilesChecked, tc.wantFilesChecked)
			}
			if res.Errors == nil {
				res.Errors = []string{}
			}
			if !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
			}
		})
	}
}

// TestValidateDocErrorOrdering pins the TS error ordering inside one pair:
// diagram duplicates, table duplicates, diagram-missing, table-missing —
// each group in first-seen number order.
func TestValidateDocErrorOrdering(t *testing.T) {
	content := strings.Join([]string{
		mermaidFence("a[\"3 x\"]\nb[\"3 y\"]\nc[\"1 z\"]"),
		"",
		calloutTable([][2]string{{"2", "dup"}, {"2", "again"}, {"4", "orphan"}}),
	}, "\n")
	got := validateDoc("doc.md", content)
	want := []string{
		"doc.md: diagram callout 3 appears 2 times (line 1)",
		"doc.md: callout table row 2 appears 2 times (line 7)",
		"doc.md: diagram callout 3 has no matching table row (line 1)",
		"doc.md: diagram callout 1 has no matching table row (line 1)",
		"doc.md: callout table row 2 has no matching diagram callout (line 7)",
		"doc.md: callout table row 4 has no matching diagram callout (line 7)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validateDoc errors = %v, want %v", got, want)
	}
}

// TestValidateDocOrphanBeforePairErrors pins that orphan-table errors come
// before pair errors regardless of document position.
func TestValidateDocOrphanBeforePairErrors(t *testing.T) {
	orphanFirst := strings.Join([]string{
		calloutTable([][2]string{{"9", "orphan"}}),
		"",
		mermaidFence(`Rel(a, b, "1")`),
	}, "\n")
	got := validateDoc("doc.md", orphanFirst)
	want := []string{
		"doc.md: callout table at line 1 has no preceding mermaid diagram",
		"doc.md: mermaid diagram at line 5 has numbered callouts but no associated callout table",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validateDoc errors = %v, want %v", got, want)
	}
}
