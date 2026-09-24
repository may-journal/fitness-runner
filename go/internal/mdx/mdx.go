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

// Heading is one ATX markdown heading found outside fenced code.
type Heading struct {
	// Level is the number of leading '#' (1-6).
	Level int
	// Text is the trimmed heading text after the marker.
	Text string
	// Line is the 1-based line number of the heading.
	Line int
}

var atxHeading = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)

// Headings returns every ATX heading in document order, skipping any line
// inside a fenced code block so a "## foo" in a code sample is not counted as
// a heading.
func Headings(content string) []Heading {
	var out []Heading
	for i, line := range blankFenced(content) {
		if hm := atxHeading.FindStringSubmatch(strings.TrimRight(line, "\r")); hm != nil {
			out = append(out, Heading{
				Level: len(hm[1]),
				Text:  strings.TrimSpace(hm[2]),
				Line:  i + 1,
			})
		}
	}
	return out
}

// TextLine is a source line paired with its 1-based line number.
type TextLine struct {
	Num  int
	Text string
}

// NonFencedLines returns one TextLine per line of content, with any line
// inside a fenced code block emptied out (its Text set to "") so line-oriented
// scans skip fenced content while line numbers stay accurate. A trailing CR is
// trimmed.
func NonFencedLines(content string) []TextLine {
	blanked := blankFenced(content)
	out := make([]TextLine, len(blanked))
	for i, line := range blanked {
		out[i] = TextLine{Num: i + 1, Text: strings.TrimRight(line, "\r")}
	}
	return out
}

// blankFenced returns content's lines with every line inside a fenced code
// block replaced by "" (line indices preserved). Fence detection mirrors
// Fences: a block closes at the first line whose trimmed start begins with the
// same marker.
func blankFenced(content string) []string {
	lines := strings.Split(content, "\n")
	inFence := false
	marker := ""
	for i, line := range lines {
		if inFence {
			if strings.HasPrefix(strings.TrimLeft(line, " \t"), marker) {
				inFence = false
			}
			lines[i] = ""
			continue
		}
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			inFence, marker = true, m[1]
			lines[i] = ""
		}
	}
	return lines
}

// --- Prose extraction ---
//
// Prose and WordCount reduce a markdown document to its prose and
// count its words under one frozen masking spec, shared by the readability
// smoke detector (text-readability) and the volume budget (prose-budget).

var (
	htmlCommentRe   = regexp.MustCompile(`(?s)<!--.*?-->`)
	listItemRe      = regexp.MustCompile(`^(?:[-*+]|\d+\.)\s+(.*)$`)
	checkboxRe      = regexp.MustCompile(`^\[[xX ]\]\s*`)
	codeSpanRe      = regexp.MustCompile("`[^`]+`")
	linkRe          = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	urlRe           = regexp.MustCompile(`https?://\S+`)
	versionRe       = regexp.MustCompile(`\bv?\d+(?:\.\d+)+\b`)
	terminalRe      = regexp.MustCompile(`[.!?:;]$`)
	sentenceSplitRe = regexp.MustCompile(`[.!?]+(?:\s+|$)`)
)

// abbrevPairs neutralize common mid-sentence periods before splitting.
var abbrevPairs = [][2]string{
	{"e.g.", "eg"}, {"E.g.", "eg"}, {"i.e.", "ie"}, {"I.e.", "ie"},
	{"etc.", "etc"}, {"vs.", "vs"}, {"cf.", "cf"},
}

// Sentences splits text into its sentences after abbreviation protection,
// dropping parts with no alphanumeric content. Feed it Prose output (or one
// masked paragraph); a text with no boundary yields one sentence.
func Sentences(text string) []string {
	text = protectAbbrev(text)
	var out []string
	for _, part := range sentenceSplitRe.Split(text, -1) {
		if strings.IndexFunc(part, IsAlnum) >= 0 {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

// protectAbbrev rewrites known abbreviations so their periods stop looking
// like sentence boundaries.
func protectAbbrev(text string) string {
	for _, p := range abbrevPairs {
		text = strings.ReplaceAll(text, p[0], p[1])
	}
	return text
}

// Prose extracts the prose from raw markdown under the frozen masking
// spec: HTML comments and front matter dropped; fenced code, headings, and
// tables skipped; every block end becomes a sentence boundary; inline code
// spans become a placeholder word; links keep their text; URLs and version
// tokens are dropped.
func Prose(content string) string {
	content = htmlCommentRe.ReplaceAllString(content, " ")
	e := &proseExtractor{}
	for _, ln := range strings.Split(StripFrontMatter(content), "\n") {
		e.line(ln)
	}
	e.flush()
	return MaskInline(strings.Join(e.units, " "))
}

// StripFrontMatter returns content with a leading --- front matter block
// removed, if present.
func StripFrontMatter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return content
}

// WordCount returns the number of prose words in text: whitespace-separated
// tokens carrying at least one alphanumeric character. Pair it with Prose to
// count the words in a markdown document.
func WordCount(text string) int {
	n := 0
	for _, tok := range strings.Fields(text) {
		if AlnumLen(tok) > 0 {
			n++
		}
	}
	return n
}

// AlnumLen counts the ASCII alphanumeric characters in one token.
func AlnumLen(tok string) int {
	n := 0
	for _, r := range tok {
		if IsAlnum(r) {
			n++
		}
	}
	return n
}

// IsAlnum reports whether r is an ASCII letter or digit.
func IsAlnum(r rune) bool {
	lower := r | 0x20
	return (lower >= 'a' && lower <= 'z') || (r >= '0' && r <= '9')
}

// MaskInline applies the inline masks to prose: inline code spans become a
// short placeholder word, links keep their text, and URLs and version tokens
// are dropped. Apply it to one paragraph or list item before counting words.
func MaskInline(s string) string {
	s = codeSpanRe.ReplaceAllString(s, "code")
	s = linkRe.ReplaceAllString(s, "$1")
	s = urlRe.ReplaceAllString(s, " ")
	s = versionRe.ReplaceAllString(s, " ")
	return s
}

// proseExtractor folds markdown lines into prose units; each finished unit
// gets terminal punctuation so block ends read as sentence boundaries.
type proseExtractor struct {
	units   []string
	buf     []string
	inFence bool
}

// line consumes one raw markdown line.
func (e *proseExtractor) line(ln string) {
	t := strings.TrimSpace(ln)
	if strings.HasPrefix(t, "```") {
		e.inFence = !e.inFence
		e.flush()
		return
	}
	if e.inFence || isProseBreak(t, ln) {
		e.flush()
		return
	}
	e.consume(t)
}

// isProseBreak reports whether the line ends the current prose unit without
// contributing text: blank lines, table rows, and headings.
func isProseBreak(trimmed, orig string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "|") || strings.HasPrefix(orig, "#")
}

// consume adds one content line to the current unit; a new list item first
// closes the previous unit, and blockquote/checkbox markers are stripped.
func (e *proseExtractor) consume(t string) {
	if m := listItemRe.FindStringSubmatch(t); m != nil {
		e.flush()
		t = m[1]
	}
	t = strings.TrimPrefix(t, "> ")
	t = checkboxRe.ReplaceAllString(t, "")
	e.buf = append(e.buf, t)
}

// flush closes the current unit, adding terminal punctuation when missing.
func (e *proseExtractor) flush() {
	joined := strings.TrimSpace(strings.Join(e.buf, " "))
	e.buf = nil
	if joined == "" {
		return
	}
	if !terminalRe.MatchString(joined) {
		joined += "."
	}
	e.units = append(e.units, joined)
}
