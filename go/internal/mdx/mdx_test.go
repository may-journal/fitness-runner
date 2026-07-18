package mdx

import (
	"reflect"
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
