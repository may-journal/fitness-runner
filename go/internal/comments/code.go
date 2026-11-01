package comments

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// Uncommented is src without its comments, each line trimmed and blank
// lines dropped, so versions that differ only in comments read the same.
// Spacing inside a line, and line breaks between code, still count.
// Directives, shebangs, and comments a toolchain reads stay.
//
// Markdown loses its HTML comments. It reports false when nothing covers
// rel.
func Uncommented(rel string, src []byte) (string, bool) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	if path.Ext(rel) == ".md" {
		return trimLines(WithoutHTMLComments(text)), true
	}
	read, ok := readers[Lang(rel, []byte(text))]
	if !ok {
		return "", false
	}
	return trimLines(without(text, read([]byte(text)))), true
}

// without is text with each note cut out, except directives and notes a
// toolchain reads.
func without(text string, notes []note) string {
	starts := newOffsets(text).starts
	var code strings.Builder
	last := 0
	for _, n := range notes {
		start := starts[n.line-1] + n.col
		if n.code || directive(n.text) || start < last {
			continue
		}
		code.WriteString(text[last:start])
		last = start + len(n.text)
	}
	return code.String() + text[last:]
}

// trimLines trims each line of text and drops the blank ones.
func trimLines(text string) string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// mdFence opens or closes a fenced code block in Markdown.
var mdFence = regexp.MustCompile("^\\s*(```|~~~)")

// mdComment matches an inline code span, kept whole, or an HTML comment.
var mdComment = regexp.MustCompile("(?s)`+[^`]*`+|<!--.*?-->")

// WithoutHTMLComments is Markdown without the HTML comments a reader never
// sees. A comment shown in a fenced block or a code span stays.
func WithoutHTMLComments(md string) string {
	var out strings.Builder
	for i, part := range fenceSplit(md) {
		if i%2 == 0 {
			part = mdComment.ReplaceAllStringFunc(part, codeSpan)
		}
		out.WriteString(part)
	}
	return out.String()
}

// codeSpan keeps a code span and drops an HTML comment.
func codeSpan(m string) string {
	if strings.HasPrefix(m, "<!--") {
		return ""
	}
	return m
}

// fenceSplit splits md into runs that take turns, prose first, then a
// fenced block with its fence lines.
func fenceSplit(md string) []string {
	parts := []string{""}
	fenced := false
	for _, line := range strings.SplitAfter(md, "\n") {
		last := len(parts) - 1
		if !mdFence.MatchString(line) {
			parts[last] += line
			continue
		}
		if fenced {
			parts[last] += line
			parts = append(parts, "")
		} else {
			parts = append(parts, line)
		}
		fenced = !fenced
	}
	return parts
}

// CommentOnly reports whether file holds the same code at base as in the
// scope being judged. From is its path at base, before any rename.
func CommentOnly(root, base, from, file string) bool {
	before, ok := gitx.Show(root, base, from)
	if !ok {
		return false
	}
	after, ok := gitx.Judged(root, file)
	if !ok {
		return false
	}
	was, ok := Uncommented(from, []byte(before))
	is, _ := Uncommented(file, []byte(after))
	return ok && was == is
}
