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
	// The numbered diagram, its legend, a caption, and a matching table —
	// the building blocks each case arranges around the diagram-to-table gap.
	diagram := mermaidFence("Container(app, \"1\", \"UI\")\nRel(app, api, \"2\")")
	legend := mermaidFence("Person(p, \"Person\")\nSystem(s, \"System\")")
	caption := "Numbers on nodes and arrows match the callout table."
	table := calloutTable([][2]string{{"1", "the app"}, {"2", "talks to"}})
	join := func(parts ...string) string { return strings.Join(parts, "\n") }

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
			name:             "passes with only a caption line before the table",
			files:            map[string]string{"doc.md": join(diagram, "", caption, "", table)},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name:             "passes with a legend and a caption between diagram and table",
			files:            map[string]string{"doc.md": join(diagram, "", legend, "", caption, "", table)},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name: "passes with a differently-worded caption",
			files: map[string]string{
				"doc.md": join(diagram, "", "Numbers on classes and relationships match the callout table.", "", table),
			},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name: "passes with a caption carrying a trailing clause",
			files: map[string]string{
				"doc.md": join(diagram, "", `Numbers match the callout table. "Screen" is a terminal output state, not a GUI view.`, "", table),
			},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name:             "passes with prose before the diagram",
			files:            map[string]string{"doc.md": join("Intro prose before the diagram.", "", diagram, "", caption, "", table)},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name:             "passes with trailing prose after the table",
			files:            map[string]string{"doc.md": join(diagram, "", caption, "", table, "", "Trailing prose after the table is fine.")},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name:             "passes for an un-numbered diagram with surrounding prose and no table",
			files:            map[string]string{"doc.md": join("Prose above.", "", mermaidFence("Container(app, \"UI\")"), "", "Prose below.")},
			ok:               true,
			wantFilesChecked: 1,
			wantErrors:       []string{},
		},
		{
			name:             "fails on prose between the diagram and the table",
			files:            map[string]string{"doc.md": join(diagram, "", "Four containers sit inside one app.", "", table)},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors: []string{
				"doc.md: prose between the diagram and its callout table (line 6) — move detail into the callout table",
			},
		},
		{
			name:             "fails on prose between the legend and the table",
			files:            map[string]string{"doc.md": join(diagram, "", legend, "", "Extra slop paragraph here.", "", table)},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors: []string{
				"doc.md: prose between the diagram and its callout table (line 11) — move detail into the callout table",
			},
		},
		{
			name: "fails on slop after the caption but allows the caption itself",
			files: map[string]string{
				"doc.md": join(diagram, "", legend, "", caption, "", "Four containers sit inside one app. UI is SwiftUI screens.", "", table),
			},
			ok:               false,
			wantFilesChecked: 1,
			wantErrors: []string{
				"doc.md: prose between the diagram and its callout table (line 13) — move detail into the callout table",
			},
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

// TestRunBodyMode validates a single supplied description document instead of
// walking files.
func TestRunBodyMode(t *testing.T) {
	diagram := mermaidFence("Container(app, \"1\", \"UI\")\nRel(app, api, \"2\")")
	caption := "Numbers on nodes and arrows match the callout table."
	table := calloutTable([][2]string{{"1", "the app"}, {"2", "talks to"}})

	bad := strings.Join([]string{diagram, "", "Four containers sit inside one app.", "", table}, "\n")
	badPath := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(badPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := run(t.TempDir(), []string{"--body-file", badPath})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok {
		t.Fatalf("expected body-mode failure, got %+v", res)
	}

	good := strings.Join([]string{diagram, "", caption, "", table}, "\n")
	goodPath := filepath.Join(t.TempDir(), "clean.md")
	if err := os.WriteFile(goodPath, []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = run(t.TempDir(), []string{"--body-file", goodPath})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("expected clean body-mode pass with 1 file, got %+v", res)
	}
}
