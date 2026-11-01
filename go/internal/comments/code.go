package comments

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// Uncommented is src without its comments, each line trimmed and blank lines
// dropped, so versions differing only in comments read the same. Directives,
// shebangs and toolchain comments stay, and it reports false when nothing
// covers rel.
func Uncommented(rel string, src []byte) (string, bool) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	if path.Ext(rel) == ".md" {
		return trimLines(WithoutHTMLComments(text)), true
	}
	read, ok := readers[Lang(rel, []byte(text))]
	if !ok {
		return "", false
	}
	return trimLines(without(text, read([]byte(text)), cut)), true
}

// CodeView blanks each comment in src, the file at rel, to spaces, keeping
// lines, columns and what a tool reads. A file no reader covers is its own.
func CodeView(rel string, src []byte) []byte {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	var view string
	if path.Ext(rel) == ".md" {
		view = withoutHTML(text, blankHTML)
	} else if read, ok := readers[Lang(rel, []byte(text))]; ok {
		view = without(text, read([]byte(text)), blank)
	} else {
		return src
	}
	return withReturns(src, view)
}

// withReturns puts back the carriage returns view lost when src's line
// endings were read as plain newlines.
func withReturns(src []byte, view string) []byte {
	if len(view) == len(src) {
		return []byte(view)
	}
	rows := strings.Split(string(src), "\n")
	out := strings.Split(view, "\n")
	for i, row := range rows[:len(rows)-1] {
		if strings.HasSuffix(row, "\r") {
			out[i] += "\r"
		}
	}
	return []byte(strings.Join(out, "\n"))
}

// cut drops a comment from a view.
func cut(string) string { return "" }

// blank turns a comment into spaces, keeping its line breaks.
func blank(text string) string {
	b := []byte(text)
	for i, c := range b {
		if c != '\n' {
			b[i] = ' '
		}
	}
	return string(b)
}

// without is text with fill applied to each note, except directives and
// notes a toolchain reads.
func without(text string, notes []note, fill func(string) string) string {
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
		code.WriteString(fill(text[start:last]))
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
	return withoutHTML(md, cut)
}

// withoutHTML is md with fill applied to each HTML comment a reader never
// sees.
func withoutHTML(md string, fill func(string) string) string {
	var out strings.Builder
	for i, part := range fenceSplit(md) {
		if i%2 == 0 {
			part = mdComment.ReplaceAllStringFunc(part, func(m string) string {
				if !strings.HasPrefix(m, "<!--") {
					return m
				}
				return fill(m)
			})
		}
		out.WriteString(part)
	}
	return out.String()
}

// blankHTML blanks an HTML comment in a code view, except a tool directive
// such as a duplicate-code marker.
func blankHTML(m string) string {
	if directive(htmlBody(m)) {
		return m
	}
	return blank(m)
}

// htmlBody is an HTML comment's text without its markers, read as a #
// comment so directive reads it as any other.
func htmlBody(m string) string {
	return "# " + strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(m, "<!--"), "-->"))
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
