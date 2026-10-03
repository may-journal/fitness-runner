package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeFiles lays out the fixture tree under dir, creating parent dirs.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunCheck(t *testing.T) {
	cases := []struct {
		name         string
		files        map[string]string
		registered   []string
		ok           bool
		filesChecked int
		wantErrors   []string
	}{
		{
			name:         "passes when no markdown files",
			files:        map[string]string{},
			ok:           true,
			filesChecked: 0,
		},
		{
			name: "judges markdown under node_modules and dist too",
			files: map[string]string{
				"README.md":                  "---\nfitnessFunctions: [\"./package.json\"]\n---\n# Root",
				"package.json":               "{}",
				"node_modules/pkg/readme.md": "---\nfitnessFunctions: [\"./nope\"]\n---",
				"dist/docs.md":               "# Doc",
			},
			ok: false, filesChecked: 3,
			wantErrors: []string{
				"dist/docs.md: missing front matter with fitnessFunctions or relatedConfigurations",
				"node_modules/pkg/readme.md: front matter path missing: ./nope",
			},
		},
		{
			name:  "fails when markdown has no front matter",
			files: map[string]string{"doc.md": "# Doc\n\nBody"},
			ok:    false, filesChecked: 1,
			wantErrors: []string{"doc.md: missing front matter with fitnessFunctions or relatedConfigurations"},
		},
		{
			name:  "fails when front matter has neither key",
			files: map[string]string{"doc.md": "---\ntitle: Foo\n---\n# Doc"},
			ok:    false, filesChecked: 1,
			wantErrors: []string{"doc.md: missing front matter with fitnessFunctions or relatedConfigurations"},
		},
		{
			name: "fails when arrays are empty",
			files: map[string]string{
				"50-59Rules/01-foo.md": "---\nfitnessFunctions: []\nrelatedConfigurations: []\n---\n# Rule",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{
				"50-59Rules/01-foo.md: fitnessFunctions must not be an empty array",
				"50-59Rules/01-foo.md: relatedConfigurations must not be an empty array",
			},
		},
		{
			name: "empty array short-circuits path validation",
			files: map[string]string{
				"doc.md": "---\nfitnessFunctions: []\nrelatedConfigurations: [\"./missing\"]\n---\n# Doc",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{"doc.md: fitnessFunctions must not be an empty array"},
		},
		{
			name: "passes when fitnessFunctions path exists relative to md file",
			files: map[string]string{
				"50-59Rules/01-foo.md":  "---\nfitnessFunctions: [\"./lib/bar.js\"]\n---\n# Rule",
				"50-59Rules/lib/bar.js": "",
			},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "passes when relatedConfigurations path exists relative to md file",
			files: map[string]string{
				"50-59Rules/01-foo.md":       "---\nrelatedConfigurations: [\"config/bar.json\"]\n---\n# Rule",
				"50-59Rules/config/bar.json": "{}",
			},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "skips http, anchor, and mailto entries",
			files: map[string]string{
				"50-59Rules/01-foo.md": "---\nfitnessFunctions: [\"https://x.com\", \"#anchor\", \"mailto:a@b.com\"]\n---\n# Rule",
			},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "passes when entry names a registered check",
			files: map[string]string{
				"src/checks/cspell/README.md": "---\nfitnessFunctions: [\"cspell\"]\nrelatedConfigurations: [\"../../../cspell.json\"]\n---\n# cspell",
				"cspell.json":                 "{}",
			},
			registered:   []string{"cspell", "markdown-no-bold-italic", "changelog"},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "fails on a check name that is not registered",
			files: map[string]string{
				"README.md": "---\nfitnessFunctions: [\"cspell\"]\n---\n# Doc",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{"README.md: front matter path missing: cspell"},
		},
		{
			name: "fails when front matter path is missing",
			files: map[string]string{
				"50-59Rules/01-foo.md": "---\nfitnessFunctions: [\"./missing.js\"]\n---\n# Rule",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{"50-59Rules/01-foo.md: front matter path missing: ./missing.js"},
		},
		{
			name: "fails when path escapes repo",
			files: map[string]string{
				"50-59Rules/01-foo.md": "---\nfitnessFunctions: [\"../../../../../../../../etc/passwd\"]\n---\n# Rule",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{"50-59Rules/01-foo.md: front matter path escapes repo: ../../../../../../../../etc/passwd"},
		},
		{
			name: "resolves parent-relative paths from the md file",
			files: map[string]string{
				"src/checks/README.md": "---\nrelatedConfigurations: [\"../../package.json\"]\n---\n# Checks",
				"package.json":         "{}",
			},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "judges the PR template like any file",
			files: map[string]string{
				".github/PULL_REQUEST_TEMPLATE.md": "> summary\n\n## Background\n\ntext\n",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{".github/PULL_REQUEST_TEMPLATE.md: missing front matter with fitnessFunctions or relatedConfigurations"},
		},
		{
			name: "reads front matter wrapped in an HTML comment",
			files: map[string]string{
				".github/PULL_REQUEST_TEMPLATE.md": "<!--\n---\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n-->\n\n> summary\n",
				".fitnessrc.json":                  "{}",
			},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "checks paths inside commented front matter",
			files: map[string]string{
				"a.md": "<!--\n---\nrelatedConfigurations: ['./gone.json']\n---\n-->\n# A\n",
			},
			ok: false, filesChecked: 1,
			wantErrors: []string{"a.md: front matter path missing: ./gone.json"},
		},
		{
			name: "an unclosed or late comment carries no front matter",
			files: map[string]string{
				"a.md": "<!--\n---\nrelatedConfigurations: ['./a.md']\n---\n",
				"b.md": "# B\n<!--\n---\nrelatedConfigurations: ['./b.md']\n---\n-->\n",
			},
			ok: false, filesChecked: 2,
			wantErrors: []string{
				"a.md: missing front matter with fitnessFunctions or relatedConfigurations",
				"b.md: missing front matter with fitnessFunctions or relatedConfigurations",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)
			res, err := runCheck(dir, tc.registered)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok || res.FilesChecked != tc.filesChecked {
				t.Fatalf("ok = %v, files = %d, want %v, %d (errors: %v)", res.Ok, res.FilesChecked, tc.ok, tc.filesChecked, res.Errors)
			}
			if !slices.Equal(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %q, want %q", res.Errors, tc.wantErrors)
			}
		})
	}
}

func TestRunReadsEnabledChecksFromEnv(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"README.md": "---\nfitnessFunctions: [\"cspell\"]\n---\n# Doc",
	})
	t.Setenv("FITNESS_ENABLED_CHECKS", "cspell\nchangelog")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestFrontMatterPaths(t *testing.T) {
	got := frontMatterPaths("fitnessFunctions: []\nrelatedConfigurations: [\"./x\", 'y']")
	want := []string{"./x", "y"}
	if !slices.Equal(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
}
