package main

import (
	"os"
	"path/filepath"
	"slices"
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

func TestSectionWordBudget(t *testing.T) {
	lim := tight
	lim.words = 3
	cases := []struct {
		name, md string
		want     []string
	}{
		{"each section under the cap passes", "## A\n\nOne two three.\n\n## B\n\n- one two\n- three\n", nil},
		{"one section over the cap fails", "## A\n\nOne two.\n\n## B\n\nFour words right here.\n",
			[]string{`f.md: section "B" has 4 prose words (max 3)`}},
		{"lists count toward the section", "## A\n\nOne two.\n\n- three four\n",
			[]string{`f.md: section "A" has 4 prose words (max 3)`}},
		{"prose before the first heading is its own section", "Four words up top.\n\n## A\n\nOne.\n",
			[]string{`f.md: section "" has 4 prose words (max 3)`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := check("f.md", tc.md, lim); !slices.Equal(got, tc.want) {
				t.Fatalf("errors = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunJudgesChangelog(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "CHANGELOG.md", "## H\n\n"+strings.Repeat("word ", 50)+"way too many words in one sentence indeed.\n")
	write(t, dir, "README.md", "## H\n\nShort and clean.\n")
	write(t, dir, ".fitnessrc.json", `{}`)

	res, err := run(dir, nil)
	if err != nil || res.Ok || !strings.HasPrefix(firstErr(res.Errors), "CHANGELOG.md: a sentence has 58 words") {
		t.Fatalf("CHANGELOG.md must be judged like any file: %+v (err %v)", res, err)
	}
}

func TestRunBodyMode(t *testing.T) {
	cases := []struct {
		name, body string
		ok         bool
	}{
		{"over-budget description fails", "## H\n\n" + strings.Repeat("word ", 30) + "end.\n", false},
		{"clean description passes", "## H\n\nShort and clean prose.\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "body.md", tc.body)
			res, err := run(dir, []string{"--body-file", filepath.Join(dir, "body.md")})
			if err != nil || res.Ok != tc.ok || res.FilesChecked != 1 {
				t.Fatalf("want ok=%v over one file: %+v (err %v)", tc.ok, res, err)
			}
		})
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
