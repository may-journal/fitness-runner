package requirements

import (
	"strings"
	"testing"
)

// calloutWhy runs `fitness-install -- mermaid-callout-why <args>` on
// happyRepo with files written over it.
func calloutWhy(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-callout-why"}, args...)...)
}

// calloutWhyDoc is a doc with a mermaid diagram followed by a table made of
// rows; the table's header sits on line 10.
func calloutWhyDoc(rows ...string) string {
	return "# Design\n\n```mermaid\nflowchart LR\n    a -->|1| b\n```\n\nThe callouts:\n\n" +
		strings.Join(rows, "\n") + "\n"
}

const (
	calloutWhyNoWhy = "| # | Description |\n| --- | --- |\n| 1 | an edge |"
	calloutWhyHead  = "| # | Description | Why |\n| --- | --- | --- |"
)

func Test0038_1(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/design.md": calloutWhyDoc(calloutWhyNoWhy)})
	sees(t, out, code, 1, `docs/design.md: callout table at line 10 is missing a "Why" column`)
}

func Test0038_2(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/design.md": calloutWhyDoc(calloutWhyHead, "| [2] | an edge |  |")})
	sees(t, out, code, 1, `docs/design.md: callout table row [2] has an empty "Why" cell (line 10)`)
}

func Test0038_3(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/design.md": calloutWhyDoc(calloutWhyHead, "| 3 | an edge |")})
	sees(t, out, code, 1, `docs/design.md: callout table row 3 has an empty "Why" cell (line 10)`)
}

func Test0038_4(t *testing.T) {
	t.Parallel()
	doc := calloutWhyDoc("| # | Description | WHY |\n| --- | --- | --- |", "| 1 | an edge | it carries the request |")
	out, code := calloutWhy(t, map[string]string{"docs/design.md": doc})
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0038_5(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/design.md": calloutWhyDoc(calloutWhyHead, "| n/a | an edge |  |")})
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0038_6(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/design.md": "# Design\n\n" + calloutWhyNoWhy + "\n"})
	sees(t, out, code, 1, `docs/design.md: callout table at line 3 is missing a "Why" column`)
}

func Test0038_7(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"docs/plain.md": "# Plain\n\n| Name | Value |\n| --- | --- |\n| a | b |\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-callout-why")
}

func Test0038_8(t *testing.T) {
	t.Parallel()
	out, code := calloutWhy(t, map[string]string{"body.md": "## Design\n\n" + calloutWhyNoWhy + "\n"}, "--body-file", "body.md")
	sees(t, out, code, 1, `(description): callout table at line 3 is missing a "Why" column`)
}
