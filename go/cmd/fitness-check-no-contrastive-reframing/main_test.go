package main

import (
	"strings"
	"testing"
)

func TestFileErrors(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		// --- positive: the split "It's not X. It's Y." form ---
		{"split form", "It's not a workout. It's a lifestyle.", 1},
		{"split that is", "That's not the point. That's the distraction.", 1},
		{"split they are", "They're not bugs. They're features.", 1},
		{"split this is", "This is not failure. This is feedback.", 1},
		{"split curly apostrophe", "It’s not a workout. It’s a lifestyle.", 1},
		{"split across paragraphs", "It's not a workout.\n\nIt's a lifestyle.", 1},

		// --- positive: the single-sentence form ---
		{"single but", "It's not a workout, but a lifestyle.", 1},
		{"single semicolon its", "This isn't about speed; it's about consistency.", 1},
		{"single rather", "It's not slow, but rather deliberate.", 1},
		{"single is not", "This is not a bug, but a feature.", 1},

		// --- negative: ordinary negations, not the cliche ---
		{"technical negation", "The file is not found, but the fallback loads.", 0},
		{"not only but also", "Not only fast, but also correct.", 0},
		{"plain negation no contrast", "It's not ready yet.", 0},
		{"affirmative only", "It's fast and reliable.", 0},
		{"two negations not restatement", "It's not X. It's not Y.", 0},
		{"non-demonstrative opening", "We should not merge, but wait for review.", 0},
		{"subject is not demonstrative", "The plan is not done. The work continues.", 0},
		{"negation without leading copula", "Speed is not the goal here.", 0},

		// --- exemptions: masking and quotes ---
		{"fenced code exempt", "```\nIt's not a workout. It's a lifestyle.\n```", 0},
		{"heading exempt", "## It's not a workout. It's a lifestyle.", 0},
		{"inline code exempt", "Use `It's not a workout. It's a lifestyle.` verbatim.", 0},
		{"quoted example exempt", `The cliche "It's not a workout. It's a lifestyle." is banned.`, 0},
		{"curly quoted example exempt", "The cliche “It's not a workout. It's a lifestyle.” is banned.", 0},
		{"front matter exempt", "---\ntitle: It's not X. It's Y.\n---\n\nReal prose here.", 0},

		// --- edge: no double counting across a run ---
		{"no double count", "It's not a workout. It's a lifestyle. It's great.", 1},
		{"empty", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fileErrors("f.md", tc.content)
			if len(got) != tc.want {
				t.Fatalf("fileErrors = %d errors, want %d\n%v", len(got), tc.want, got)
			}
		})
	}
}

func TestMessageNamesFileAndSnippet(t *testing.T) {
	errs := fileErrors("docs/x.md", "It's not a workout. It's a lifestyle.")
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %v", errs)
	}
	if !strings.HasPrefix(errs[0], "docs/x.md:") {
		t.Fatalf("message should name the file: %q", errs[0])
	}
	if !strings.Contains(errs[0], "workout") {
		t.Fatalf("message should quote the snippet: %q", errs[0])
	}
}

func TestLongSnippetTruncated(t *testing.T) {
	long := "It's not " + strings.Repeat("a very long clause ", 10) + ", but simple."
	errs := fileErrors("f.md", long)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0], "…") {
		t.Fatalf("long snippet should be truncated with an ellipsis: %q", errs[0])
	}
}
