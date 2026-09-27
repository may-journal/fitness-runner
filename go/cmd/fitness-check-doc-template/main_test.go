package main

import (
	"os"
	"path/filepath"
	"testing"
)

// goodTemplate carries a guiding comment in every section.
const goodTemplate = "# NNNN — Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\n<!-- what -->\n\n## Consequences\n\n<!-- fallout -->\n"

func TestIsTemplate(t *testing.T) {
	cases := map[string]struct{ conf, git bool }{
		"docs/adr/template.md":             {true, false},
		"docs/adr/adr.template.md":         {true, false},
		"docs/adr/0001-x.md":               {false, false},
		".github/PULL_REQUEST_TEMPLATE.md": {false, true},
		".github/ISSUE_TEMPLATE/plan.md":   {false, true},
		".github/workflows/README.md":      {false, false},
	}
	for path, want := range cases {
		if got := isConformanceTemplate(path); got != want.conf {
			t.Errorf("isConformanceTemplate(%q) = %v, want %v", path, got, want.conf)
		}
		if got := isGithubTemplate(path); got != want.git {
			t.Errorf("isGithubTemplate(%q) = %v, want %v", path, got, want.git)
		}
	}
}

// writeRepo lays out files under a temp root and returns it.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func fails(t *testing.T, root string) bool {
	t.Helper()
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	return !res.Ok
}

func TestConformance(t *testing.T) {
	adr := "docs/adr/"
	// A governed file matching the template passes.
	root := writeRepo(t, map[string]string{
		adr + "template.md":  goodTemplate,
		adr + "0001-good.md": "# 0001 — Good\n\n## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n",
	})
	if fails(t, root) {
		t.Fatal("matching file should pass")
	}

	// An extra section fails.
	root = writeRepo(t, map[string]string{
		adr + "template.md": goodTemplate,
		adr + "0001-bad.md": "# 0001 — Bad\n\n## Context\n\nc\n\n## Status\n\ns\n\n## Decision\n\nd\n\n## Consequences\n\ne\n",
	})
	if !fails(t, root) {
		t.Fatal("extra Status section should fail")
	}

	// A missing section fails.
	root = writeRepo(t, map[string]string{
		adr + "template.md": goodTemplate,
		adr + "0001-bad.md": "# 0001 — Bad\n\n## Context\n\nc\n\n## Consequences\n\ne\n",
	})
	if !fails(t, root) {
		t.Fatal("missing Decision section should fail")
	}
}

func TestNearestTemplateGovernsDescendants(t *testing.T) {
	// A template governs a file in a descendant folder.
	root := writeRepo(t, map[string]string{
		"docs/template.md":      goodTemplate,
		"docs/deep/0001-bad.md": "# 0001 — Bad\n\n## Context\n\nc\n",
	})
	if !fails(t, root) {
		t.Fatal("descendant file should be governed by the ancestor template")
	}
}

func TestUngovernedFilePasses(t *testing.T) {
	// No template anywhere above: the file is not governed.
	root := writeRepo(t, map[string]string{
		"docs/notes.md": "# Notes\n\n## Anything\n\ngoes\n",
	})
	if fails(t, root) {
		t.Fatal("ungoverned file should pass")
	}
}

func TestTemplateNeedsCommentPerSection(t *testing.T) {
	// A template section without a guiding comment fails.
	root := writeRepo(t, map[string]string{
		"docs/adr/template.md": "# NNNN — Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\nno comment here\n",
	})
	if !fails(t, root) {
		t.Fatal("template section without a comment should fail")
	}
}

func TestGithubTemplateGetsCommentRuleButNotConformance(t *testing.T) {
	// PR template lacking a comment fails the comment rule...
	root := writeRepo(t, map[string]string{
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\nno comment\n",
	})
	if !fails(t, root) {
		t.Fatal("github template without a comment should fail")
	}

	// ...but a github template does not govern sibling files.
	root = writeRepo(t, map[string]string{
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\n<!-- ctx -->\n\n## Changelog\n\n<!-- bullets -->\n",
		".github/other.md":                 "# Other\n\n## Totally\n\ndifferent\n",
	})
	if fails(t, root) {
		t.Fatal("github template must not force sibling files to match")
	}
}
