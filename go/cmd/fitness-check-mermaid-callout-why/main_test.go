package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// diagram is the fenced mermaid block the TS suite prefixes its documents
// with.
const diagram = "```mermaid\nRel(a, b, \"1\")\n```"

// doc joins document lines with newlines, mirroring the TS fixtures.
func doc(lines ...string) string { return strings.Join(lines, "\n") }

// TestValidateDoc ports the TS validateDoc suite plus pins for row-shape
// edge cases.
func TestValidateDoc(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"non-empty Why column passes",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| 1 | an edge | because it matters |"),
			[]string{}},
		{"missing Why column fails",
			doc(diagram, "", "| # | Description |", "| --- | --- |", "| 1 | an edge |"),
			[]string{`doc.md: callout table at line 5 is missing a "Why" column`}},
		{"numbered row with empty Why cell flagged",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| 1 | an edge |  |"),
			[]string{`doc.md: callout table row 1 has an empty "Why" cell (line 5)`}},
		{"Why accepted case-insensitively",
			doc(diagram, "", "| # | Description | WHY |", "| --- | --- | --- |",
				"| 1 | an edge | reason |"),
			[]string{}},
		{"bracketed row number kept verbatim in the error",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| [2] | an edge | |"),
			[]string{`doc.md: callout table row [2] has an empty "Why" cell (line 5)`}},
		{"row shorter than the Why index counts as empty",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| 3 | an edge |"),
			[]string{`doc.md: callout table row 3 has an empty "Why" cell (line 5)`}},
		{"unnumbered row with empty Why cell ignored",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| n/a | an edge |  |"),
			[]string{}},
		{"orphan callout table still validated",
			doc("| # | Description |", "| --- | --- |", "| 1 | an edge |"),
			[]string{`doc.md: callout table at line 1 is missing a "Why" column`}},
		{"non-callout table ignored",
			doc(diagram, "", "| Name | Value |", "| --- | --- |", "| a | b |"),
			[]string{}},
		{"errors keep document order across tables",
			doc(diagram, "", "| # | Description |", "| --- | --- |", "| 1 | an edge |", "",
				"| # | Description | Why |", "| --- | --- | --- |", "| 1 | an edge | |"),
			[]string{
				`doc.md: callout table at line 5 is missing a "Why" column`,
				`doc.md: callout table row 1 has an empty "Why" cell (line 9)`,
			}},
		{"diagram-only doc has nothing to flag", diagram, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateDoc("doc.md", tc.content)
			if got == nil {
				got = []string{}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("validateDoc = %v, want %v", got, tc.want)
			}
		})
	}
}

// write creates rel under dir, making parent directories as needed.
func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRun ports the TS check-level suite: filesChecked counts only files
// with at least one parsed mermaid block.
func TestRun(t *testing.T) {
	t.Run("passes when there are no relevant files", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "plain.md", "# no mermaid here\n")
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Ok || res.FilesChecked != 0 || len(res.Errors) != 0 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("fails a doc whose callout table lacks a Why column", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "a.md",
			doc(diagram, "", "| # | Description |", "| --- | --- |", "| 1 | an edge |"))
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || res.FilesChecked != 1 {
			t.Fatalf("unexpected result: %+v", res)
		}
		want := `a.md: callout table at line 5 is missing a "Why" column`
		if len(res.Errors) != 1 || res.Errors[0] != want {
			t.Fatalf("errors = %v, want [%s]", res.Errors, want)
		}
	})

	t.Run("counts block-bearing files and skips block-free ones", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "good.md",
			doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |",
				"| 1 | an edge | because |"))
		write(t, dir, "docs/bad.md",
			doc(diagram, "", "| # | Description |", "| --- | --- |", "| 1 | an edge |"))
		write(t, dir, "docs/plain.md", "just prose\n")
		res, err := run(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || res.FilesChecked != 2 || len(res.Errors) != 1 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

// TestRunBodyMode validates a single supplied description document instead of
// walking files.
func TestRunBodyMode(t *testing.T) {
	bad := doc(diagram, "", "| # | Description |", "| --- | --- |", "| 1 | an edge |")
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

	good := doc(diagram, "", "| # | Description | Why |", "| --- | --- | --- |", "| 1 | an edge | because |")
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
