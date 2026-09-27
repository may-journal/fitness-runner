package main

import (
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/render"
)

func TestCISummaryMarkdown(t *testing.T) {
	rows := []render.Row{
		{Name: "cspell", Ok: true, FilesChecked: 12, Ms: 40},
		{Name: "prose-budget", Ok: false, FilesChecked: 68, Ms: 15, Errors: []string{"docs/checks.md: a list has 9 items (max 8)"}},
		{Name: "timed-out", Ok: false, FilesChecked: -1, Ms: 5000, Errors: []string{"Check timed out after 5s"}},
	}
	md := ciSummaryMarkdown(rows)

	for _, want := range []string{
		"## Fitness checks",
		"| Check | Status | Files | Time |",
		"| cspell | ✅ pass | 12 | 40ms |",
		"| prose-budget | ❌ fail | 68 | 15ms |",
		"| timed-out | ❌ fail | — | 5000ms |", // negative files render as a dash
		"### ❌ prose-budget",
		"- docs/checks.md: a list has 9 items (max 8)",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("summary missing %q\n---\n%s", want, md)
		}
	}
	// A passing check gets no detail section.
	if strings.Contains(md, "### ❌ cspell") {
		t.Error("passing check should not get a detail section")
	}
}

func TestAnnotationLine(t *testing.T) {
	cases := []struct {
		name  string
		check string
		err   string
		want  string
	}{
		{
			"file and line", "cspell", "go/x/README.md:8:85 - Unknown word (foo)",
			"::error file=go/x/README.md,line=8::cspell: go/x/README.md:8:85 - Unknown word (foo)",
		},
		{
			"file only", "prose-budget", "docs/checks.md: a list has 9 items (max 8)",
			"::error file=docs/checks.md::prose-budget: docs/checks.md: a list has 9 items (max 8)",
		},
		{
			"no file", "changelog-updated", "CHANGELOG.md new section heading must use current date and time",
			// "CHANGELOG.md" has no trailing colon here, so it is treated as prose.
			"::error::changelog-updated: CHANGELOG.md new section heading must use current date and time",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := annotationLine(tc.check, tc.err); got != tc.want {
				t.Fatalf("annotationLine\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestAnnotationsCap(t *testing.T) {
	var errs []string
	for i := 0; i < 15; i++ {
		errs = append(errs, "msg")
	}
	rows := []render.Row{{Name: "c", Ok: false, Errors: errs}}

	got := annotations(rows, maxAnnotations)
	if len(got) != maxAnnotations+1 {
		t.Fatalf("want %d lines (cap + note), got %d", maxAnnotations+1, len(got))
	}
	last := got[len(got)-1]
	if !strings.HasPrefix(last, "::warning::") || !strings.Contains(last, "5 more") {
		t.Fatalf("expected a truncation warning naming 5 more, got %q", last)
	}
}

func TestAnnotationsUnderCapHasNoNote(t *testing.T) {
	rows := []render.Row{{Name: "c", Ok: false, Errors: []string{"a", "b"}}}
	got := annotations(rows, maxAnnotations)
	if len(got) != 2 {
		t.Fatalf("want 2 annotations, got %d", len(got))
	}
	for _, l := range got {
		if strings.HasPrefix(l, "::warning::") {
			t.Fatalf("did not expect a truncation warning: %q", l)
		}
	}
}

func TestPassingRowsProduceNoAnnotations(t *testing.T) {
	rows := []render.Row{{Name: "c", Ok: true, FilesChecked: 3}}
	if got := annotations(rows, maxAnnotations); len(got) != 0 {
		t.Fatalf("passing rows should yield no annotations, got %v", got)
	}
}

func TestEscaping(t *testing.T) {
	// A percent and newline in the message are encoded.
	line := annotationLine("c", "weird 100% off\nsecond line")
	if strings.Contains(line, "100% off") || !strings.Contains(line, "100%25 off") {
		t.Fatalf("percent not escaped: %q", line)
	}
	if strings.Contains(line, "\n") || !strings.Contains(line, "%0A") {
		t.Fatalf("newline not escaped: %q", line)
	}
}
