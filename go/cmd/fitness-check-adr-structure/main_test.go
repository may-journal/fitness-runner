package main

import (
	"os"
	"path/filepath"
	"testing"
)

const goodBody = `---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0007 — A short claim

## Context

Some forces.

## Decision

The decision.

## Consequences

The fallout.
`

func TestValidateADR(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		content string
		wantErr bool
	}{
		{"well formed", "docs/architecture/adr/0007-a.md", goodBody, false},
		{"status first is allowed", "docs/architecture/adr/0007-a.md",
			"# 0007 — T\n\n## Status\n\nSupersedes 0001.\n\n## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", false},
		{"missing h1", "docs/architecture/adr/0007-a.md",
			"## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", true},
		{"id mismatch", "docs/architecture/adr/0007-a.md",
			"# 0009 — T\n\n## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", true},
		{"hyphen not em-dash", "docs/architecture/adr/0007-a.md",
			"# 0007 - T\n\n## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", true},
		{"missing decision", "docs/architecture/adr/0007-a.md",
			"# 0007 — T\n\n## Context\n\nc\n\n## Consequences\n\ne\n", true},
		{"wrong order", "docs/architecture/adr/0007-a.md",
			"# 0007 — T\n\n## Decision\n\nd\n\n## Context\n\nc\n\n## Consequences\n\ne\n", true},
		{"status after context", "docs/architecture/adr/0007-a.md",
			"# 0007 — T\n\n## Context\n\nc\n\n## Status\n\ns\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", true},
		{"unknown section", "docs/architecture/adr/0007-a.md",
			"# 0007 — T\n\n## Context\n\nc\n\n## Notes\n\nn\n\n## Decision\n\nd\n\n## Consequences\n\ne\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateADR(tc.file, tc.content)
			if got := len(errs) > 0; got != tc.wantErr {
				t.Fatalf("validateADR errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

func TestIsADR(t *testing.T) {
	cases := map[string]bool{
		"docs/architecture/adr/0003-quality.md": true,
		"docs/architecture/adr/template.md":     false,
		"docs/architecture/adr/README.md":       false,
		"docs/other/0003-quality.md":            false,
		"docs/architecture/adr/0003.md":         false,
	}
	for file, want := range cases {
		if got := isADR(file); got != want {
			t.Fatalf("isADR(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestRunSelfGatesAndReports(t *testing.T) {
	// No ADR directory: passes with zero files.
	if res, err := run(t.TempDir(), nil); err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("empty repo: want pass/0, got %+v err %v", res, err)
	}

	dir := t.TempDir()
	adr := filepath.Join(dir, "docs", "architecture", "adr")
	if err := os.MkdirAll(adr, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(adr, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("0007-a.md", goodBody)
	write("template.md", "# NNNN — Title\n\n## Context\n\nx\n")

	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The template is unnumbered, so only the one numbered ADR is checked.
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("one valid ADR (template ignored): want pass/1, got %+v", res)
	}

	write("0008-b.md", "# 0008 — Bad\n\n## Decision\n\nd\n")
	if res, _ := run(dir, nil); res.Ok || res.FilesChecked != 2 {
		t.Fatalf("one bad ADR added: want fail/2, got %+v", res)
	}
}
