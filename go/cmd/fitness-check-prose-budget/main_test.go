package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tight limits make small fixtures trip each rule.
var tight = limits{
	sentenceWords:      5,
	paragraphSentences: 2,
	sectionParagraphs:  2,
	listItemWords:      5,
	listItems:          3,
	words:              1000,
}

func firstErr(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	return errs[0]
}

func TestCheckLimits(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string // substring expected in some error ("" = must pass)
	}{
		{"clean passes", "## H\n\nShort words here.\n", ""},
		{"long sentence", "## H\n\nThis particular sentence has far too many words indeed.\n", "a sentence has"},
		{"too many sentences", "## H\n\nOne. Two. Three.\n", "has 3 sentences"},
		{"too many paragraphs", "## H\n\nA one.\n\nB two.\n\nC three.\n", "has 3 paragraphs"},
		{"long list item", "## H\n\n- this item runs on far too long\n", "a list item has"},
		{"too many list items", "## H\n\n- a\n- b\n- c\n- d\n", "has 4 items"},
		{"fenced code is not prose", "## H\n\n```\nthis fenced line has plenty of words but is code\n```\n", ""},
		{"table is not prose", "## H\n\n| a very wide header cell | b |\n| - | - |\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := check("f.md", tc.md, tight)
			if tc.want == "" {
				if len(errs) != 0 {
					t.Fatalf("expected pass, got %v", errs)
				}
				return
			}
			if !strings.Contains(strings.Join(errs, "\n"), tc.want) {
				t.Fatalf("errors %v missing %q", errs, tc.want)
			}
		})
	}
}

func TestTotalWordsFallback(t *testing.T) {
	lim := tight
	lim.words = 3
	errs := check("f.md", "## H\n\nfour words right here.\n", lim)
	if !strings.Contains(firstErr(errs), "over the 3-word budget") {
		t.Fatalf("expected word-budget error, got %v", errs)
	}
}

func TestIsExempt(t *testing.T) {
	exempt := []string{"CHANGELOG.md", "docs/x/**"}
	for _, rel := range []string{"CHANGELOG.md", "docs/x/a.md", "docs/x/sub/b.md"} {
		if !isExempt(rel, exempt) {
			t.Errorf("%q should be exempt", rel)
		}
	}
	for _, rel := range []string{"README.md", "docs/x.md", "docs/xy/a.md"} {
		if isExempt(rel, exempt) {
			t.Errorf("%q should not be exempt", rel)
		}
	}
}

func TestRunExemptsChangelogOnly(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "CHANGELOG.md", "## H\n\n"+strings.Repeat("word ", 50)+"way too many words in one sentence indeed.\n")
	write(t, dir, "README.md", "## H\n\nShort and clean.\n")
	write(t, dir, ".fitnessrc.json", `{}`)

	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("CHANGELOG.md must be exempt and README clean: %+v", res)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
