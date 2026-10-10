package requirements

import "testing"

// docTemplateNote guides its one section with a comment, beside docTemplateADR
// in the same folder.
const docTemplateNote = "# Note\n\n## Summary\n\n<!-- gist -->\n"

func Test0066_1(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/adr/adr.template.md":  docTemplateADR,
		"docs/adr/note.template.md": docTemplateNote,
		"docs/adr/0001-note.md":     "# 0001 Note\n\n## Summary\n\ns\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0066_2(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/adr/adr.template.md":  docTemplateADR,
		"docs/adr/note.template.md": docTemplateNote,
		"docs/adr/0001-bad.md":      "# 0001 Bad\n\n## Context\n\nc\n\n## Decision\n\nd\n",
	})
	sees(t, out, code, 1,
		"docs/adr/0001-bad.md: matches none of 2 templates; closest is docs/adr/adr.template.md: Missing `## Consequences` section")
}

func Test0066_3(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/arch/templates/adr.md":  docTemplateADR,
		"docs/arch/templates/note.md": docTemplateNote,
		"docs/arch/04-code.md":        "# Code\n\n## Summary\n\ns\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0066_4(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/arch/templates/note.md": docTemplateNote,
		"docs/arch/adr/template.md":   docTemplateADR,
		"docs/arch/adr/0001-bad.md":   "# 0001 Bad\n\n## Summary\n\ns\n",
	})
	sees(t, out, code, 1, "docs/arch/adr/0001-bad.md: Missing `## Context` section")
}

func Test0066_5(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/arch/templates/note.md": "# Note\n\n## Summary\n\nno comment here\n",
	})
	sees(t, out, code, 1, "docs/arch/templates/note.md: section `## Summary` needs a guiding <!-- comment -->")
}
