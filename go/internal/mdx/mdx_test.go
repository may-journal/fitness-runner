package mdx

import (
	"reflect"
	"strings"
	"testing"
)

func TestFrontMatter(t *testing.T) {
	body, ok := FrontMatter("---\na: 1\nb: 2\n---\n\n# Doc\n")
	if !ok || body != "a: 1\nb: 2" {
		t.Fatalf("got %q ok=%v", body, ok)
	}
	if _, ok := FrontMatter("# Doc\n---\nx\n---\n"); ok {
		t.Fatal("front matter must start on line 1")
	}
	if _, ok := FrontMatter("---\nunterminated\n"); ok {
		t.Fatal("unterminated front matter is absent")
	}
}

func TestFences(t *testing.T) {
	doc := "intro\n```mermaid\nA --> B\n```\ntext\n~~~~go\ncode\n~~~~\n```\nplain\n"
	fences := Fences(doc)
	if len(fences) != 3 {
		t.Fatalf("got %d fences, want 3", len(fences))
	}
	if fences[0].Info != "mermaid" || fences[0].Line != 2 || fences[0].Body != "A --> B" {
		t.Fatalf("fence 0: %+v", fences[0])
	}
	if fences[1].Info != "go" || fences[1].Body != "code" {
		t.Fatalf("fence 1: %+v", fences[1])
	}
	// unterminated block runs to end of document
	if fences[2].Info != "" || fences[2].Body != "plain\n" {
		t.Fatalf("fence 2: %+v", fences[2])
	}
}

func TestSplitTableRow(t *testing.T) {
	got := SplitTableRow("| # | Description | Why |")
	if !reflect.DeepEqual(got, []string{"#", "Description", "Why"}) {
		t.Fatalf("got %v", got)
	}
	got = SplitTableRow("a | b")
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("no outer pipes: got %v", got)
	}
}

func TestIsTableSeparator(t *testing.T) {
	for _, yes := range []string{"| --- | --- |", "|:--:|----|", " --- ", "|-|"} {
		if !IsTableSeparator(yes) {
			t.Errorf("%q should be a separator", yes)
		}
	}
	for _, no := range []string{"| a | b |", "text", "|  |"} {
		if IsTableSeparator(no) {
			t.Errorf("%q should not be a separator", no)
		}
	}
}

func TestHeadings(t *testing.T) {
	doc := "# Title\n\nintro\n\n## Background\n\ntext\n\n```\n## not a heading\n```\n\n### Deep\n#no space\n## What needs to happen\n"
	got := Headings(doc)
	want := []Heading{
		{Level: 1, Text: "Title", Line: 1},
		{Level: 2, Text: "Background", Line: 5},
		{Level: 3, Text: "Deep", Line: 13},
		{Level: 2, Text: "What needs to happen", Line: 15},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestProse(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string
	}{
		{"front matter dropped", "---\ntitle: x\n---\n\nReal prose here.", "Real prose here."},
		{"html comments dropped", "<!-- cspell:ignore weird words -->\n\nReal prose here.", "Real prose here."},
		{"fences skipped", "Before.\n\n```go\nfunc main() {}\n```\n\nAfter.", "Before. After."},
		{"headings skipped", "# Title\n\nBody text.", "Body text."},
		{"tables skipped", "| a | b |\n| - | - |\n\nBody text.", "Body text."},
		{"block end gets period", "A list intro\n\nNext paragraph.", "A list intro. Next paragraph."},
		{"list items are units", "- first item\n- second item\n", "first item. second item."},
		{"checkbox stripped", "- [x] done thing\n", "done thing."},
		{"blockquote marker stripped", "> Quoted line.\n", "Quoted line."},
		{"code span masked", "Run `go build -o bin ./cmd/...` now.", "Run code now."},
		{"link keeps text", "See [the plan](../plans/01.md) here.", "See the plan here."},
		{"url dropped", "Docs at https://example.com/x live on.", "Docs at live on."},
		{"version dropped", "Bump to v1.2.3 today.", "Bump to today."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(strings.Fields(Prose(tc.md)), " ")
			if got != tc.want {
				t.Fatalf("Prose(%q) = %q, want %q", tc.md, got, tc.want)
			}
		})
	}
}

func TestWordCount(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"plain words", "one two three", 3},
		{"punctuation-only tokens ignored", "real words --- and *** more", 4},
		{"empty is zero", "   ", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WordCount(tc.text); got != tc.want {
				t.Fatalf("WordCount(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}
