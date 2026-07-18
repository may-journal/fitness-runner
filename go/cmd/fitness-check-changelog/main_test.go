package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const validChangelog = "# Changelog\n\n### 2026.02.15.1100\n\n- item"

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		files      map[string]string
		ok         bool
		wantErrors []string
	}{
		{"passes with one dated section",
			map[string]string{"CHANGELOG.md": validChangelog},
			true, nil},
		{"fails when only ## heading (no ###)",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n## 2026.02.15.1100\n\n- item"},
			false, []string{"CHANGELOG.md must have at least one ### yyyy.mm.dd.HHMM section"}},
		{"fails when a heading uses the old @time format",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n### 2026-02-15@11am\n\n- item"},
			false, []string{`every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026-02-15@11am")`}},
		{"fails when a heading has date but no time",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n### 2026.02.15\n\n- item"},
			false, []string{`every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026.02.15")`}},
		{"passes when all headings match",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n### 2026.02.15.1100\n\n- a\n\n### 2026.02.14.2100\n\n- b"},
			true, nil},
		{"fails only the one heading that does not match",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n### 2026.02.15.1100\n\n- a\n\n### 2026.02.15\n\n- b"},
			false, []string{`every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026.02.15")`}},
		{"heading regex is not end-anchored: trailing text allowed",
			map[string]string{"CHANGELOG.md": "# Changelog\n\n### 2026.02.15.1100 hotfix\n\n- item"},
			true, nil},
		{"fails when CHANGELOG.md is missing at root",
			nil,
			false, []string{"missing root CHANGELOG.md"}},
		{"passes when package.json and package-lock match first heading",
			map[string]string{
				"CHANGELOG.md":      validChangelog,
				"package.json":      `{"version":"0.1.0-2026.02.15.1100"}`,
				"package-lock.json": `{"version":"0.1.0-2026.02.15.1100","packages":{"":{}}}`,
			},
			true, nil},
		{"passes when package.json matches and no package-lock",
			map[string]string{
				"CHANGELOG.md": validChangelog,
				"package.json": `{"version":"0.1.0-2026.02.15.1100"}`,
			},
			true, nil},
		{"fails when package.json suffix does not match first heading",
			map[string]string{
				"CHANGELOG.md": validChangelog,
				"package.json": `{"version":"0.1.0-2026.02.15.1200"}`,
			},
			false, []string{msgVersionChangelog}},
		{"fails when package.json version has no timestamp suffix",
			map[string]string{
				"CHANGELOG.md": validChangelog,
				"package.json": `{"version":"1.0.0"}`,
			},
			false, []string{msgVersionChangelog}},
		{"suffix regex is end-anchored: trailing text breaks the match",
			map[string]string{
				"CHANGELOG.md": validChangelog,
				"package.json": `{"version":"0.1.0-2026.02.15.1100-beta"}`,
			},
			false, []string{msgVersionChangelog}},
		{"fails when package-lock version differs from package.json",
			map[string]string{
				"CHANGELOG.md":      validChangelog,
				"package.json":      `{"version":"0.1.0-2026.02.15.1100"}`,
				"package-lock.json": `{"version":"0.1.0-2026.02.15.1200","packages":{"":{}}}`,
			},
			false, []string{msgVersionLock}},
		{"collects both version errors together",
			map[string]string{
				"CHANGELOG.md":      validChangelog,
				"package.json":      `{"version":"1.0.0"}`,
				"package-lock.json": `{"version":"2.0.0"}`,
			},
			false, []string{msgVersionChangelog, msgVersionLock}},
		{"format errors short-circuit version errors",
			map[string]string{
				"CHANGELOG.md": "# Changelog\n\n### 2026.02.15\n\n- item",
				"package.json": `{"version":"1.0.0"}`,
			},
			false, []string{`every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "### 2026.02.15")`}},
		{"fails when package.json is invalid JSON",
			map[string]string{
				"CHANGELOG.md": validChangelog,
				"package.json": "not json",
			},
			false, []string{"package.json is invalid JSON"}},
		{"fails when package-lock.json is invalid JSON",
			map[string]string{
				"CHANGELOG.md":      validChangelog,
				"package.json":      `{"version":"0.1.0-2026.02.15.1100"}`,
				"package-lock.json": "not json",
			},
			false, []string{"package-lock.json is invalid JSON"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			res, err := run(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if !tc.ok && !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %q, want %q", res.Errors, tc.wantErrors)
			}
		})
	}
}

func TestPackageChangelogMismatch(t *testing.T) {
	matching := jsonVersion{value: "0.1.0-2026.02.15.1100", present: true}
	if got := packageChangelogMismatch(matching, "2026.02.15.1100"); got != "" {
		t.Fatalf("matching suffix: got %q, want no error", got)
	}
	if got := packageChangelogMismatch(jsonVersion{}, "2026.02.15.1100"); got != msgVersionChangelog {
		t.Fatalf("absent version: got %q, want %q", got, msgVersionChangelog)
	}
}

func TestStrictlyEqualMirrorsJS(t *testing.T) {
	absent := jsonVersion{}
	null := jsonVersion{value: nil, present: true}
	str := jsonVersion{value: "1.0.0", present: true}
	if !strictlyEqual(absent, absent) {
		t.Fatal("undefined === undefined should hold")
	}
	if strictlyEqual(absent, null) {
		t.Fatal("undefined === null should not hold")
	}
	if !strictlyEqual(null, null) {
		t.Fatal("null === null should hold")
	}
	if strictlyEqual(str, jsonVersion{value: float64(1), present: true}) {
		t.Fatal("cross-type strict equality should not hold")
	}
	obj := jsonVersion{value: map[string]any{}, present: true}
	if strictlyEqual(obj, obj) {
		t.Fatal("two parsed objects are never strictly equal")
	}
}
