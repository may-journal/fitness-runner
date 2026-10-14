package requirements

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// docTemplate runs `fitness-install -- doc-template` on happyRepo with files
// written over it.
func docTemplate(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "doc-template")
}

// docTemplateNote guides its one section with a comment.
const docTemplateNote = "# Note\n\n## Summary\n\n<!-- gist -->\n"

// docTemplateADR guides each of its three sections with a comment.
const docTemplateADR = "# NNNN Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\n<!-- what -->\n\n## Consequences\n\n<!-- fallout -->\n"

const (
	dtViews     = "docs/architecture/"
	dtTemplates = dtViews + "templates/"
)

// dtCopied is what a user has after following the architecture-contract
// README: the five shipped templates in docs/architecture/templates, and
// each view started as a copy of its template.
func dtCopied(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, view := range []string{acContext, acContainer, acComponent, acCode, acJourney} {
		body, err := os.ReadFile(filepath.Join("..", "..", "docs", "architecture", "templates", view))
		mustDo(t, err)
		files[dtTemplates+view], files[dtViews+view] = string(body), string(body)
	}
	return files
}

// dtEdit replaces old with new in one copied view, failing when old is not
// there to replace.
func dtEdit(t *testing.T, files map[string]string, view, old, new string) {
	t.Helper()
	if !strings.Contains(files[dtViews+view], old) {
		t.Fatalf("%s has no %q", view, old)
	}
	files[dtViews+view] = strings.Replace(files[dtViews+view], old, new, 1)
}

// dtLine is the 1-based line of the first line in a view starting with
// prefix, as an error names it.
func dtLine(t *testing.T, files map[string]string, view, prefix string) string {
	t.Helper()
	for i, line := range strings.Split(files[dtViews+view], "\n") {
		if strings.HasPrefix(line, prefix) {
			return dtViews + view + ":" + strconv.Itoa(i+1)
		}
	}
	t.Fatalf("%s has no line starting %q", view, prefix)
	return ""
}

func Test0023_12(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/arch/templates/adr.md":       docTemplateADR,
		"docs/arch/templates/note.md":      docTemplateNote,
		"docs/arch/04-code.md":             "# Code\n\n## Summary\n\ns\n",
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\n<!-- context -->\n",
		".github/other.md":                 "# Other\n\n## Totally\n\ndifferent\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0023_10(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md":   docTemplateADR,
		"docs/adr/0001-extra.md": "# 0001 Extra\n\n## Context\n\nc\n\n## Status\n\ns\n\n## Decision\n\nd\n\n## Consequences\n\ne\n",
		"docs/adr/0002-short.md": "# 0002 Short\n\n## Context\n\nc\n\n## Consequences\n\ne\n",
	})
	sees(t, out, code, 1,
		"docs/adr/0001-extra.md: Unexpected `## Status` section",
		"docs/adr/0002-short.md: Missing `## Decision` section")
}

func Test0023_4(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/templates/adr.md":           docTemplateADR,
		"docs/deep/note.template.md":      "# Note\n\n## Summary\n\n<!-- gist -->\n",
		"docs/deep/sub/0001-note.md":      "# 0001 Note\n\n## Context\n\nc\n",
		"docs/views/templates/04-code.md": docTemplateNote,
		"docs/views/04-code.md":           "# Code\n\n## Summary\n\ns\n",
		"docs/views/notes.md":             "# Notes\n\n## Open threads\n\no\n",
	})
	sees(t, out, code, 1,
		"docs/deep/sub/0001-note.md: Missing `## Summary` section",
		"docs/views/notes.md: Missing `## Context` section",
		"docs/views/notes.md: Unexpected `## Open threads` section")
}

func Test0023_11(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/adr/template.md":             "# NNNN Title\n\n## Context\n\n<!-- why -->\n\n## Decision\n\nno comment here\n",
		".github/PULL_REQUEST_TEMPLATE.md": "## Background\n\nno comment\n",
		".github/ISSUE_TEMPLATE/plan.md":   "## Goal\n\nno comment\n",
	})
	sees(t, out, code, 1,
		"docs/adr/template.md: section `## Decision` needs a guiding <!-- comment -->",
		".github/PULL_REQUEST_TEMPLATE.md: section `## Background` needs a guiding <!-- comment -->",
		".github/ISSUE_TEMPLATE/plan.md: section `## Goal` needs a guiding <!-- comment -->")
}

func Test0023_9(t *testing.T) {
	t.Parallel()
	out, code := docTemplate(t, map[string]string{
		"docs/adr/adr.template.md":  docTemplateADR,
		"docs/adr/note.template.md": docTemplateNote,
		"docs/adr/0001-bad.md":      "# 0001 Bad\n\n## Other\n\no\n",
	})
	sees(t, out, code, 1,
		"docs/adr/0001-bad.md: sections match none of its templates: docs/adr/adr.template.md, docs/adr/note.template.md")
}

func Test0023_13(t *testing.T) {
	t.Parallel()
	files := dtCopied(t)
	files[dtViews+acContainer] += "\n## Setup\n\nHow a consumer runs the app.\n"
	files[dtViews+acJourney] += "\n## Notes\n\nWhat the journey leaves out.\n"
	dtEdit(t, files, acContext, "fitnessFunctions: ['architecture-contract', 'architecture-actors']", "fitnessFunctions:\n  - architecture-actors\n  - \"architecture-contract\"")
	dtEdit(t, files, acCode, "fitnessFunctions: ['architecture-contract', 'architecture-code']", "fitnessFunctions: [architecture-code, architecture-contract]")
	dtEdit(t, files, acComponent, "match the callout table.", "match the callout table. <!-- kept in step with the diagram -->")
	files[dtViews+"notes.md"] = "# Notes\n\n## Open threads\n\nWhat the views leave out.\n"
	files[dtViews+"archive/"+acCode] = "# Old code\n\n## Before\n\nThe first layout.\n"
	out, code := docTemplate(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "doc-template", "14")
}

func Test0023_14(t *testing.T) {
	t.Parallel()
	files := dtCopied(t)
	dtEdit(t, files, acContext, ", 'architecture-actors'", "")
	dtEdit(t, files, acComponent, "fitnessFunctions: ['architecture-contract', 'architecture-components']\n", "relatedConfigurations: ['../../README.md']\n")
	dtEdit(t, files, acCode, "```mermaid\nclassDiagram\n    class Example", "```mermaid\nflowchart TB\n    class Example")
	dtEdit(t, files, acJourney, "## Journey", "## Steps")
	out, code := docTemplate(t, files)
	sees(t, out, code, 1,
		dtViews+acContext+":1: front matter `fitnessFunctions` needs architecture-actors as "+dtTemplates+acContext+" has",
		dtViews+acComponent+":1: front matter needs `fitnessFunctions` as "+dtTemplates+acComponent+" has",
		dtViews+acCode+": the opening needs a classDiagram legend as "+dtTemplates+acCode+" has",
		dtViews+acJourney+": sections must open with `## Legend`, `## Journey` as "+dtTemplates+acJourney+" does")
}

func Test0023_15(t *testing.T) {
	t.Parallel()
	files := dtCopied(t)
	dtEdit(t, files, acJourney, "| Type    |", "| Kind    |")
	dtEdit(t, files, acCode, "Numbers on classes", "Labels on classes")
	files[dtViews+acContainer] = files[dtViews+acContainer][:strings.Index(files[dtViews+acContainer], "| #")]
	journey := dtLine(t, files, acJourney, "| #   | Description          | Kind")
	code := dtLine(t, files, acCode, "| #")
	out, exit := docTemplate(t, files)
	sees(t, out, exit, 1,
		journey+": callout table columns #, Description, Kind, Why must be #, Description, Type, Why as "+dtTemplates+acJourney+" has",
		code+`: callout table needs the caption "Numbers on classes and relationships match the callout table" above it as `+dtTemplates+acCode+" has",
		dtViews+acContainer+": the opening needs a callout table with columns #, Description, Why as "+dtTemplates+acContainer+" has")
}
