package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func doc(bullets ...string) string {
	s := "# Changelog\n\n## Changes\n\n### 2026.07.18.1900\n\n"
	for _, b := range bullets {
		s += "- " + b + "\n"
	}
	s += "\n### 2026.07.18.1800\n\n- Feat: a\n- Fix: b\n- Docs: c\n"
	return s
}

func ok3() []string {
	return []string{"Feat: one", "Fix: two", "Docs: three"}
}

func TestBulletCountBounds(t *testing.T) {
	cases := []struct {
		name    string
		bullets []string
		wantErr string
	}{
		{"three passes", ok3(), ""},
		{"five passes", []string{"Feat: a", "Fix: b", "Docs: c", "Chore: d", "Perf: e"}, ""},
		{"two fails", []string{"Feat: a", "Fix: b"}, "has 2 bullets"},
		{"six fails", []string{"Feat: a", "Fix: b", "Docs: c", "Chore: d", "Perf: e", "Refactor: f"}, "has 6 bullets"},
		{"zero fails", nil, "has 0 bullets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := judge(doc(tc.bullets...))
			if tc.wantErr == "" && len(errs) != 0 {
				t.Fatalf("want pass, got %v", errs)
			}
			if tc.wantErr != "" && (len(errs) == 0 || !strings.Contains(errs[0], tc.wantErr)) {
				t.Fatalf("want %q, got %v", tc.wantErr, errs)
			}
		})
	}
}

func TestBulletLengthBoundary(t *testing.T) {
	pad := strings.Repeat("x", 364-len("Feat: "))
	exact := "Feat: " + pad // 364 runes: passes
	over := exact + "y"     // 365 runes: fails
	if errs := judge(doc(exact, "Fix: b", "Docs: c")); len(errs) != 0 {
		t.Fatalf("364 chars must pass: %v", errs)
	}
	errs := judge(doc(over, "Fix: b", "Docs: c"))
	if len(errs) != 1 || !strings.Contains(errs[0], "bullet is 365 characters; keep each under 365") {
		t.Fatalf("365 chars must fail: %v", errs)
	}
}

func TestContinuationLinesCountTowardLength(t *testing.T) {
	content := "### 2026.07.18.1900\n\n- Feat: " + strings.Repeat("x", 300) + "\n  " +
		strings.Repeat("y", 300) + "\n- Fix: b\n- Docs: c\n"
	errs := judge(content)
	if len(errs) != 1 || !strings.Contains(errs[0], "keep each under 365") {
		t.Fatalf("wrapped bullet must count as one long bullet: %v", errs)
	}
	if !strings.Contains(errs[0], "CHANGELOG.md:3:") {
		t.Fatalf("error anchors to the bullet's first line: %v", errs)
	}
}

func TestSemanticTypePrefix(t *testing.T) {
	for _, good := range []string{"Feat: x", "Fix: x", "Docs: x", "Style: x", "Refactor: x", "Perf: x", "Test: x", "Build: x", "Ci: x", "Chore: x", "Revert: x"} {
		if errs := judge(doc(good, "Fix: b", "Docs: c")); len(errs) != 0 {
			t.Fatalf("%q must pass: %v", good, errs)
		}
	}
	for _, bad := range []string{"feat: lowercase", "Added something", "Feature: wrong word", "Feat:no space"} {
		errs := judge(doc(bad, "Fix: b", "Docs: c"))
		if len(errs) != 1 || !strings.Contains(errs[0], "must start with a semantic type") {
			t.Fatalf("%q must fail with the type error: %v", bad, errs)
		}
	}
}

func TestEverySectionIsJudged(t *testing.T) {
	content := doc(ok3()...) + "\n### 2026.05.01.0000\n\n- sloppy old bullet without a type\n"
	errs := judge(content)
	if len(errs) != 2 {
		t.Fatalf("old sections are judged too (count + type): %v", errs)
	}
	if !strings.Contains(errs[0], `section "2026.05.01.0000"`) {
		t.Fatalf("count error names its section: %v", errs)
	}
}

func TestRunFileHandling(t *testing.T) {
	dir := t.TempDir()
	res, err := run(dir, nil)
	if err != nil || !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("missing changelog passes with 0 files: %+v %v", res, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(doc(ok3()...)), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = run(dir, nil)
	if err != nil || !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("valid changelog passes with 1 file: %+v %v", res, err)
	}
}
