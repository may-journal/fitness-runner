package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// levelDoc builds a doc with one diagram and one callout table of
// [#, Description] rows — the twin of the TS suite's helper.
func levelDoc(descriptions ...string) string {
	lines := []string{"```mermaid", `Rel(a, b, "1")`, "```", "", "| # | Description |", "| --- | --- |"}
	for i, d := range descriptions {
		lines = append(lines, "| "+strconv.Itoa(i+1)+" | "+d+" |")
	}
	return strings.Join(lines, "\n")
}

// writeFiles lays the rel→content fixture files out under dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLevelDescriptionsNormalizesWhitespaceAndCase(t *testing.T) {
	set := levelDescriptions(levelDoc("The  App", "THE api"))
	if !set.has("the app") || !set.has("the api") {
		t.Fatalf("set = %v, want normalized 'the app' and 'the api'", set.items)
	}
}

func TestRun(t *testing.T) {
	const bleedError = `architecture/02-containers.md: description "user interacts with the timeline" ` +
		`repeats level 1 verbatim — lower levels should add level-specific rationale`
	cases := []struct {
		name             string
		files            map[string]string
		ok               bool
		wantFilesChecked int
		wantErrors       []string
	}{
		{"passes when adjacent levels have distinct descriptions",
			map[string]string{
				"architecture/01-context.md":    levelDoc("The whole system"),
				"architecture/02-containers.md": levelDoc("The macOS app", "The sync engine"),
			},
			true, 2, nil},
		{"flags a description repeated verbatim from the previous level",
			map[string]string{
				"architecture/01-context.md":    levelDoc("User interacts with the timeline"),
				"architecture/02-containers.md": levelDoc("User interacts with the timeline", "New detail"),
			},
			false, 2, []string{bleedError}},
		{"ignores markdown outside the architecture level convention",
			map[string]string{"README.md": levelDoc("Anything")},
			true, 0, nil},
		{"unnumbered architecture files do not count",
			map[string]string{"architecture/overview.md": levelDoc("Anything")},
			true, 0, nil},
		{"architecture must be its own path segment",
			map[string]string{"not-architecture/02-x.md": levelDoc("Anything")},
			true, 0, nil},
		{"nested architecture directories count",
			map[string]string{
				"docs/architecture/01-context.md": levelDoc("The whole system"),
				"docs/architecture/02-app.md":     levelDoc("The whole system"),
			},
			false, 2, []string{`docs/architecture/02-app.md: description "the whole system" ` +
				`repeats level 1 verbatim — lower levels should add level-specific rationale`}},
		{"level file without callout tables still counts",
			map[string]string{
				"architecture/01-context.md": levelDoc("The whole system"),
				"architecture/03-code.md":    "# Code\n\nProse only.\n",
			},
			true, 2, nil},
		{"non-adjacent levels never compared",
			map[string]string{
				"architecture/01-context.md":    levelDoc("Shared line"),
				"architecture/02-containers.md": levelDoc("Something else"),
				"architecture/03-components.md": levelDoc("Shared line"),
			},
			true, 3, nil},
		{"levels sort numerically not lexically",
			map[string]string{
				"architecture/2-app.md":   levelDoc("The engine"),
				"architecture/10-deep.md": levelDoc("The engine"),
			},
			false, 2, []string{`architecture/10-deep.md: description "the engine" ` +
				`repeats level 2 verbatim — lower levels should add level-specific rationale`}},
		{"duplicate rows dedupe to one error per description",
			map[string]string{
				"architecture/01-context.md": levelDoc("The app"),
				"architecture/02-app.md":     levelDoc("The  APP", "the app", "New detail"),
			},
			false, 2, []string{`architecture/02-app.md: description "the app" ` +
				`repeats level 1 verbatim — lower levels should add level-specific rationale`}},
		{"empty description cells are ignored",
			map[string]string{
				"architecture/01-context.md": levelDoc(""),
				"architecture/02-app.md":     levelDoc("", "Detail"),
			},
			true, 2, nil},
		{"description column found by header not position",
			map[string]string{
				"architecture/01-context.md": "| # | Label | DESCRIPTION |\n| --- | --- | --- |\n| 1 | a | The system |",
				"architecture/02-app.md":     "| # | Label | Description |\n| --- | --- | --- |\n| 1 | b | The system |",
			},
			false, 2, []string{`architecture/02-app.md: description "the system" ` +
				`repeats level 1 verbatim — lower levels should add level-specific rationale`}},
		{"missing description header falls back to column 1",
			map[string]string{
				"architecture/01-context.md": "| # | Text | Notes |\n| --- | --- | --- |\n| 1 | Same | x |",
				"architecture/02-app.md":     "| # | Text | Notes |\n| --- | --- | --- |\n| 1 | same | y |",
			},
			false, 2, []string{`architecture/02-app.md: description "same" ` +
				`repeats level 1 verbatim — lower levels should add level-specific rationale`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)
			res, err := run(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok || res.FilesChecked != tc.wantFilesChecked {
				t.Fatalf("ok=%v filesChecked=%d, want ok=%v filesChecked=%d (errors: %v)",
					res.Ok, res.FilesChecked, tc.ok, tc.wantFilesChecked, res.Errors)
			}
			if tc.wantErrors != nil && !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %#v, want %#v", res.Errors, tc.wantErrors)
			}
			if tc.wantErrors == nil && len(res.Errors) != 0 {
				t.Fatalf("errors = %#v, want none", res.Errors)
			}
		})
	}
}

func TestErrorsFollowSetInsertionOrder(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"architecture/01-context.md": levelDoc("Beta line", "Alpha line"),
		"architecture/02-app.md":     levelDoc("Beta line", "Alpha line"),
	})
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`architecture/02-app.md: description "beta line" repeats level 1 verbatim — lower levels should add level-specific rationale`,
		`architecture/02-app.md: description "alpha line" repeats level 1 verbatim — lower levels should add level-specific rationale`,
	}
	if !reflect.DeepEqual(res.Errors, want) {
		t.Fatalf("errors = %#v, want %#v", res.Errors, want)
	}
}
