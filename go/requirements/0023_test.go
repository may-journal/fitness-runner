package requirements

import "testing"

// docTemplate runs `fitness-install -- doc-template` on happyRepo with files
// written over it.
func docTemplate(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "doc-template")
}

// docTemplateADR guides each of its three sections with a comment.
const docTemplateADR = "# NNNN Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\n<!-- what -->\n\n## Consequences\n\n<!-- fallout -->\n"

func Test0023_1(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md":  docTemplateADR,
		"docs/adr/0001-good.md": "# 0001 Good\n\n## Context\n\nc\n\n## Decision\n\nd\n\n## Consequences\n\ne\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0023_2(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md": docTemplateADR,
		"docs/adr/0001-bad.md": "# 0001 Bad\n\n## Context\n\nc\n\n## Status\n\ns\n\n## Decision\n\nd\n\n## Consequences\n\ne\n",
	})
	sees(t, out, code, 1, "docs/adr/0001-bad.md: Unexpected `## Status` section")
}

func Test0023_3(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md": docTemplateADR,
		"docs/adr/0001-bad.md": "# 0001 Bad\n\n## Context\n\nc\n\n## Consequences\n\ne\n",
	})
	sees(t, out, code, 1, "docs/adr/0001-bad.md: Missing `## Decision` section")
}

func Test0023_4(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/template.md":           docTemplateADR,
		"docs/deep/note.template.md": "# Note\n\n## Summary\n\n<!-- gist -->\n",
		"docs/deep/sub/0001-note.md": "# 0001 Note\n\n## Context\n\nc\n",
	})
	sees(t, out, code, 1, "docs/deep/sub/0001-note.md: Missing `## Summary` section")
}

func Test0023_5(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/notes.md": "# Notes\n\n## Anything\n\ngoes\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0023_6(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md": "# NNNN Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\nno comment here\n",
	})
	sees(t, out, code, 1, "docs/adr/template.md: section `## Decision` needs a guiding <!-- comment -->")
}

func Test0023_7(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\nno comment\n",
		".github/ISSUE_TEMPLATE/plan.md":   "## Goal\n\nno comment\n",
	})
	sees(t, out, code, 1,
		".github/PULL_REQUEST_TEMPLATE.md: section `## Background` needs a guiding <!-- comment -->",
		".github/ISSUE_TEMPLATE/plan.md: section `## Goal` needs a guiding <!-- comment -->")
}

func Test0023_8(t *testing.T) {
	out, code := docTemplate(t, map[string]string{
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\n<!-- context -->\n",
		".github/other.md":                 "# Other\n\n## Totally\n\ndifferent\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}
