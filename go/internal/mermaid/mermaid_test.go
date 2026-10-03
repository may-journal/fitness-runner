package mermaid

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mermaidFence wraps a mermaid diagram body in a fenced block.
func mermaidFence(body string) string {
	return "```mermaid\n" + body + "\n```"
}

// TestExtractDiagramNumbers ports the TS suite's cases plus pins for the
// lookahead emulation, each verified against the JavaScript original.
func TestExtractDiagramNumbers(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []int
	}{
		{"quoted label bare", `Rel(a, b, "6")`, []int{6}},
		{"quoted label with prose", `Rel(a, b, "6 Uses")`, []int{6}},
		{"quoted label among args", `Container(app, "1", "SwiftUI")`, []int{1}},
		{"no numbers", `Container(app, "SwiftUI")`, []int{}},
		{"empty body", "", []int{}},
		{"node-shape labels", "persona((1 Persona))\nscreen[2 Screen]\naction(3 Action)\nsystem[(4 System)]",
			[]int{1, 2, 3, 4}},
		{"colon edge label", "getChecks --> resolveCheckNames : 11", []int{11}},
		{"pipe label bare", "app -->|5| api", []int{5}},
		{"pipe label with prose", "app -->|6 calls the api| api", []int{6}},
		{"style lines skipped", "screen[2 Screen]:::screen\nclassDef screen fill:#6366f1,stroke-width:2px",
			[]int{2}},
		{"mixed quoted and colon labels", "class Check[\"1 Check\"]\nclass Run[\"2 Run\"]\nRun --> Check : 3",
			[]int{1, 2, 3}},
		// Emulation pins beyond the TS suite (outputs verified against JS):
		{"inline link label", "A -- 5 --> B", []int{5}},
		{"lookahead consumes nothing between quoted labels", `"1"2"`, []int{1, 2}},
		{"failed lookahead scans again from the next char", `"12x "3 "`, []int{3}},
		{"colon label at end of line", "A --> B : 7", []int{7}},
		{"colon label followed by prose char", "A --> B : 7x", []int{}},
		{"colon label followed by a tab", "A --> B: 8\t", []int{8}},
		{"duplicate numbers at distinct positions kept", "a[\"1 x\"]\nb[\"1 y\"]", []int{1, 1}},
		{"same number twice on one line kept", "x -->|5| y : 5 z", []int{5, 5}},
		{"inline link resumes after the consumed match", "--1--2--", []int{1}},
		{"node-shape retry finds a later opener", "((1x (2 y", []int{2}},
		{"pipe label at end of line has no follower", "|12", []int{}},
		{"colon label match consumes its digits", ": 12 3", []int{12}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractDiagramNumbers(tc.body)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ExtractDiagramNumbers(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestIsCalloutHeader(t *testing.T) {
	cases := []struct {
		name  string
		cells []string
		want  bool
	}{
		{"hash", []string{"#", "Description"}, true},
		{"no", []string{"No", "Description"}, true},
		{"no with dot", []string{"No.", "Description"}, true},
		{"ref", []string{"Ref", "Why"}, true},
		{"callout", []string{"callout", "Description"}, true},
		{"other first cell", []string{"Name", "Value"}, false},
		{"prefix only", []string{"#foo"}, false},
		{"empty cells", []string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCalloutHeader(tc.cells); got != tc.want {
				t.Fatalf("IsCalloutHeader(%v) = %v, want %v", tc.cells, got, tc.want)
			}
		})
	}
}

func TestParseDoc(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []DocBlock
	}{
		{
			"diagram then callout table",
			mermaidFence(`Rel(a, b, "1")`) + "\n\n| # | Description |\n| --- | --- |\n| 1 | an edge |",
			[]DocBlock{
				&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}},
				&CalloutTableBlock{Header: []string{"#", "Description"}, Line: 5,
					Numbers: []int{1}, Rows: [][]string{{"1", "an edge"}}},
			},
		},
		{
			"non-callout table ignored",
			"| Name | Value |\n| --- | --- |\n| a | 1 |",
			nil,
		},
		{
			"tilde fence",
			"~~~mermaid\nRel(a, b, \"1\")\n~~~",
			[]DocBlock{&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}}},
		},
		{
			"uppercase mermaid info",
			"```MERMAID\na[\"1 x\"]\n```",
			[]DocBlock{&DiagramBlock{Body: `a["1 x"]`, Line: 1, Numbers: []int{1}}},
		},
		{
			"longer marker is not closed by a shorter fence line",
			"````mermaid\n```\nRel(a, b, \"1\")\n````\nafter",
			[]DocBlock{&DiagramBlock{Body: "```\nRel(a, b, \"1\")", Line: 1, Numbers: []int{1}}},
		},
		{
			"unterminated fence runs to end of document",
			"```mermaid\nRel(a, b, \"1\")",
			[]DocBlock{&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}}},
		},
		{
			"mermaids fails the word-boundary and is not a mermaid fence",
			"```mermaids\nRel(a, b, \"1\")\n```",
			nil,
		},
		{
			"indented fence and closing fence",
			"  ```mermaid\nRel(a, b, \"1\")\n  ```",
			[]DocBlock{&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}}},
		},
		{
			"space between marker and info string",
			"``` mermaid\nRel(a, b, \"1\")\n```",
			[]DocBlock{&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}}},
		},
		{
			"closing fence may be longer than the opener",
			"```mermaid\nRel(a, b, \"1\")\n`````",
			[]DocBlock{&DiagramBlock{Body: `Rel(a, b, "1")`, Line: 1, Numbers: []int{1}}},
		},
		{
			"bracketed and non-numeric first cells",
			"| # | D |\n| --- | --- |\n| [1] | a |\n| [2] | b |\n| x | c |",
			[]DocBlock{&CalloutTableBlock{Header: []string{"#", "D"}, Line: 1, Numbers: []int{1, 2},
				Rows: [][]string{{"[1]", "a"}, {"[2]", "b"}, {"x", "c"}}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseDoc(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseDoc(%q) = %#v, want %#v", tc.content, got, tc.want)
			}
		})
	}
}

// assertPairCounts checks how many pairs and orphan tables a pairing found.
func assertPairCounts(t *testing.T, result PairResult, pairs, orphans int) {
	t.Helper()
	if len(result.Pairs) != pairs {
		t.Fatalf("pairs = %d, want %d", len(result.Pairs), pairs)
	}
	if len(result.OrphanTables) != orphans {
		t.Fatalf("orphanTables = %d, want %d", len(result.OrphanTables), orphans)
	}
}

// assertPair checks one pair's diagram numbers and its table: a nil table
// wants no table at all, otherwise a table with exactly those numbers.
func assertPair(t *testing.T, index int, pair DiagramTablePair, diagram, table []int) {
	t.Helper()
	if !reflect.DeepEqual(pair.Diagram.Numbers, diagram) {
		t.Fatalf("pair %d diagram numbers = %v, want %v", index, pair.Diagram.Numbers, diagram)
	}
	assertPairTable(t, index, pair.Table, table)
}

func assertPairTable(t *testing.T, index int, got *CalloutTableBlock, want []int) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("pair %d table = %+v, want nil", index, got)
		}
		return
	}
	if got == nil || !reflect.DeepEqual(got.Numbers, want) {
		t.Fatalf("pair %d table = %+v, want numbers %v", index, got, want)
	}
}

func TestPairDiagramsWithTables(t *testing.T) {
	t.Run("pairs each diagram with the following table and flags orphan tables", func(t *testing.T) {
		blocks := ParseDoc("| # | Description |\n| --- | --- |\n| 1 | orphan |\n\n" +
			mermaidFence(`Rel(a, b, "1")`) + "\n\n| # | Description |\n| --- | --- |\n| 1 | paired |")
		result := PairDiagramsWithTables(blocks)
		assertPairCounts(t, result, 1, 1)
		assertPair(t, 0, result.Pairs[0], []int{1}, []int{1})
	})

	t.Run("pairs a diagram with nil when no table follows", func(t *testing.T) {
		blocks := ParseDoc(mermaidFence(`Rel(a, b, "1")`))
		result := PairDiagramsWithTables(blocks)
		want := []DiagramTablePair{{Diagram: blocks[0].(*DiagramBlock), Table: nil}}
		if !reflect.DeepEqual(result.Pairs, want) {
			t.Fatalf("pairs = %+v, want %+v", result.Pairs, want)
		}
		assertPairCounts(t, result, 1, 0)
	})

	t.Run("skips a legend diagram so the numbered diagram pairs with the table", func(t *testing.T) {
		blocks := ParseDoc(mermaidFence(`Rel(a, b, "1")`) + "\n\n" +
			mermaidFence("Person(p, \"Person\")\nSystem(s, \"System\")") + // legend: no callout numbers
			"\n\n| # | Description |\n| --- | --- |\n| 1 | paired past the legend |")
		result := PairDiagramsWithTables(blocks)
		assertPairCounts(t, result, 1, 0)
		assertPair(t, 0, result.Pairs[0], []int{1}, []int{1})
	})

	t.Run("flushes a pending diagram when a second numbered diagram starts", func(t *testing.T) {
		blocks := ParseDoc(mermaidFence(`a["1 x"]`) + "\n" + mermaidFence(`b["2 y"]`) +
			"\n\n| # | D |\n| --- | --- |\n| 2 | b |")
		result := PairDiagramsWithTables(blocks)
		assertPairCounts(t, result, 2, 0)
		assertPair(t, 0, result.Pairs[0], []int{1}, nil)
		assertPair(t, 1, result.Pairs[1], []int{2}, []int{2})
	})
}

// writeDocs creates each slash-separated name:content pair under root.
func writeDocs(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunDocCheck(t *testing.T) {
	root := t.TempDir()
	writeDocs(t, root, map[string]string{
		"diagram.md":               mermaidFence(`a["1 x"]`) + "\n",
		"prose.md":                 "# Title\n\njust prose, no blocks\n",
		"table-only.md":            "| # | D |\n| --- | --- |\n| 1 | a |\n",
		"docs/nested.md":           mermaidFence(`b["2 y"]`) + "\n",
		"node_modules/included.md": mermaidFence(`c["3 z"]`) + "\n",
	})

	var seen []string
	errs, filesChecked, err := RunDocCheck(root, func(file, content string) []string {
		seen = append(seen, file)
		if file == "diagram.md" {
			return []string{file + ": boom"}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// No directory is skipped any more; only block-free files are left out.
	if filesChecked != 4 {
		t.Fatalf("filesChecked = %d, want 4 (block-free files excluded)", filesChecked)
	}
	wantSeen := []string{"diagram.md", "docs/nested.md", "node_modules/included.md", "table-only.md"}
	if !reflect.DeepEqual(seen, wantSeen) {
		t.Fatalf("validated files = %v, want %v", seen, wantSeen)
	}
	wantErrs := []string{"diagram.md: boom"}
	if !reflect.DeepEqual(errs, wantErrs) {
		t.Fatalf("errors = %v, want %v", errs, wantErrs)
	}
}
