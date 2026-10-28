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

// dtShipped is what a user has after following the architecture-contract
// README: each view in docs/architecture started as a copy of its shipped
// template, and no copy of the templates.
func dtShipped(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, view := range []string{acContext, acContainer, acComponent, acCode, acJourney} {
		body, err := os.ReadFile(filepath.Join("..", "internal", "architecture", "templates", view))
		mustDo(t, err)
		files[dtViews+view] = string(body)
	}
	return files
}

// dtCopied is dtShipped with the five templates also copied into
// docs/architecture/templates, as repos did before fitness shipped them.
func dtCopied(t *testing.T) map[string]string {
	t.Helper()
	files := dtShipped(t)
	for _, view := range []string{acContext, acContainer, acComponent, acCode, acJourney} {
		files[dtTemplates+view] = files[dtViews+view]
	}
	return files
}

// dtShippedName is how a message names a view's shipped template.
func dtShippedName(view string) string {
	return "the shipped " + view
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

func Test0023_16(t *testing.T) {
	t.Parallel()
	files := dtCopied(t)
	dtEdit(t, files, acCode, "```mermaid\nclassDiagram\n    class Example", "```mermaid\nflowchart TB\n    class Example")
	dtEdit(t, files, acJourney, "## Journey", "## Steps")
	out, code := docTemplate(t, files)
	sees(t, out, code, 1,
		dtViews+acCode+": the opening needs a classDiagram legend as "+dtTemplates+acCode+" has",
		dtViews+acJourney+": sections must open with `## Legend`, `## Journey` as "+dtTemplates+acJourney+" does")

	files = dtCopied(t)
	dtEdit(t, files, acJourney, "| Type    |", "| Kind    |")
	dtEdit(t, files, acCode, "Numbers on classes", "Labels on classes")
	files[dtViews+acContainer] = files[dtViews+acContainer][:strings.Index(files[dtViews+acContainer], "| #")]
	journey := dtLine(t, files, acJourney, "| #   | Description          | Kind")
	table := dtLine(t, files, acCode, "| #")
	out, code = docTemplate(t, files)
	sees(t, out, code, 1,
		journey+": callout table columns #, Description, Kind, Why must be #, Description, Type, Why as "+dtTemplates+acJourney+" has",
		table+`: callout table needs the caption "Numbers on classes and letters on relationships match the callout table" above it as `+dtTemplates+acCode+" has",
		dtViews+acContainer+": the opening needs a callout table with columns #, Description, Why as "+dtTemplates+acContainer+" has")

	files = dtShipped(t)
	dtEdit(t, files, acContainer, "C4Container\n    title Product", "C4Context\n    title Product")
	dtEdit(t, files, acComponent, "C4Component\n    title Legend", "C4Container\n    title Legend")
	dtEdit(t, files, acCode, "```mermaid\nclassDiagram\n    class Record", "```mermaid\nflowchart TB\n    class Record")
	dtEdit(t, files, acJourney, "flowchart TB", "flowchart LR")
	out, code = docTemplate(t, files)
	sees(t, out, code, 1,
		dtLine(t, files, acContainer, "```mermaid")+": diagram type C4Context must be C4Container as "+dtShippedName(acContainer)+" has",
		dtViews+acComponent+": the opening needs a C4Component legend as "+dtShippedName(acComponent)+" has",
		": legend type C4Container must be C4Component as "+dtShippedName(acComponent)+" has",
		dtLine(t, files, acCode, "```mermaid")+": diagram type flowchart TB must be classDiagram as "+dtShippedName(acCode)+" has",
		dtLine(t, files, acJourney, "```mermaid")+": legend type flowchart LR must be flowchart TB as "+dtShippedName(acJourney)+" has")
}

func Test0023_17(t *testing.T) {
	t.Parallel()
	files := dtCopied(t)
	files[dtViews+acContainer] += "\n## Setup\n\nHow a consumer runs the app.\n"
	files[dtViews+acJourney] += "\n## Notes\n\nWhat the journey leaves out.\n"
	dtEdit(t, files, acComponent, "match the callout table.", "match the callout table. <!-- kept in step with the diagram -->")
	files[dtViews+"notes.md"] = "# Notes\n\n## Open threads\n\nWhat the views leave out.\n"
	files[dtViews+"archive/"+acCode] = "# Old code\n\n## Before\n\nThe first layout.\n"
	out, code := docTemplate(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "doc-template", "14")

	files = dtShipped(t)
	files[dtViews+acContainer] += "\n## Setup\n\nHow a consumer runs the app.\n"
	out, code = docTemplate(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	dtEdit(t, files, acJourney, "## Journey", "## Steps")
	dtEdit(t, files, acCode, "Numbers on classes", "Labels on classes")
	out, code = docTemplate(t, files)
	sees(t, out, code, 1,
		dtViews+acJourney+": sections must open with `## Legend`, `## Journey` as "+dtShippedName(acJourney)+" does",
		`: callout table needs the caption "Numbers on classes and letters on relationships match the callout table" above it as `+dtShippedName(acCode)+" has")
}

func Test0023_18(t *testing.T) {
	t.Parallel()
	files := dtShipped(t)
	dtEdit(t, files, acContext, "    UpdateElementStyle(user,",
		"    SystemDb_Ext(db, \"4 Records\", \"Keeps records.\")\n    UpdateElementStyle(db, $fontColor=\"#FFFFFF\", $bgColor=\"#64748b\",  $borderColor=\"#475569\")\n    UpdateLayoutConfig($c4ShapeInRow=\"3\")\n    Lay_R(user, product)\n    UpdateElementStyle(user,")
	dtEdit(t, files, acCode, "    save --> Record : A", "    save --> Record : A\n    Store : +save()\n    <<interface>> Store\n    style Store "+paletteClass)
	out, code := docTemplate(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	dtEdit(t, files, acContext, "UpdateElementStyle(product, $bgColor=\"#1d4ed8\"", "UpdateElementStyle(product, $bgColor=\"#ff0000\"")
	dtEdit(t, files, acContainer, "    UpdateElementStyle(app, "+paletteContainer+"\n", "")
	dtEdit(t, files, acComponent, "Container_Boundary(app,", "System_Boundary(app,")
	dtEdit(t, files, acComponent, "    Rel(user, entry, \"A\")", "    Rel(user, entry, \"A\")\n    UpdateRelStyle(user, entry, $textColor=\"red\")")
	dtEdit(t, files, acCode, "    save --> Record : A", "    save --> Record : A\n    classDef hot fill:#ff0000")
	out, code = docTemplate(t, files)
	sees(t, out, code, 1,
		"`UpdateElementStyle(product, $bgColor=\"#ff0000\", $borderColor=\"#1e3a8a\", $fontColor=\"#ffffff\")` must be `UpdateElementStyle(product, "+paletteSystem+"` as "+dtShippedName(acContext)+" styles each System",
		"Container app needs `UpdateElementStyle(app, "+paletteContainer+"` as "+dtShippedName(acContainer)+" styles each Container",
		dtLine(t, files, acComponent, "    System_Boundary(app,")+": System_Boundary app is a kind "+dtShippedName(acComponent)+" does not use; use Person, Container_Boundary, Component, System_Ext",
		"`UpdateRelStyle(user, entry, $textColor=\"red\")` is a style "+dtShippedName(acComponent)+" does not use",
		"`classDef hot fill:#ff0000` is a style "+dtShippedName(acCode)+" does not use")
	for _, styled := range []string{"Person user", "Component entry", "class save", "class Record"} {
		if strings.Contains(read(out), styled) {
			t.Errorf("%s keeps its palette and must not fail, got:\n%s", styled, out)
		}
	}
}
