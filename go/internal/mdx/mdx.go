// Package mdx holds the small markdown parsing primitives the markdown and
// mermaid checks share: front matter extraction, fenced-block scanning, and
// GFM table row splitting. Semantics mirror the TypeScript helpers the
// checks were ported from.
package mdx

import (
	"regexp"
	"strings"
)

// FrontMatter returns the YAML front matter body (the lines between the
// leading "---" fence pair) and whether the document has one. The opening
// fence must be the very first line.
func FrontMatter(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			return strings.Join(lines[1:i], "\n"), true
		}
	}
	return "", false
}

// Fence is one fenced code block.
type Fence struct {
	// Info is the trimmed info string after the opening marker (e.g. "mermaid").
	Info string
	// Line is the 1-based line number of the opening fence.
	Line int
	// Body is the block content, fences excluded.
	Body string
}

var fenceOpen = regexp.MustCompile("^\\s*(`{3,}|~{3,})\\s*(.*)$")

// Fences returns every fenced code block in document order. A block closes
// at the first line whose trimmed start begins with the same marker; an
// unterminated block runs to the end of the document.
func Fences(content string) []Fence {
	lines := strings.Split(content, "\n")
	var out []Fence
	for i := 0; i < len(lines); i++ {
		m := fenceOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		marker := m[1]
		var body []string
		j := i + 1
		for ; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimLeft(lines[j], " \t"), marker) {
				break
			}
			body = append(body, lines[j])
		}
		out = append(out, Fence{
			Info: strings.TrimSpace(m[2]),
			Line: i + 1,
			Body: strings.Join(body, "\n"),
		})
		i = j
	}
	return out
}

// SplitTableRow splits a GFM table row into trimmed cells, dropping one
// leading and one trailing pipe.
func SplitTableRow(line string) []string {
	inner := strings.TrimSpace(line)
	inner = strings.TrimPrefix(inner, "|")
	inner = strings.TrimSuffix(inner, "|")
	cells := strings.Split(inner, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}
	return cells
}

var tableSeparator = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

// IsTableSeparator reports whether the line is a GFM table separator row.
func IsTableSeparator(line string) bool {
	return tableSeparator.MatchString(line)
}
