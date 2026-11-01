package swift

import "github.com/may-journal/fitness-runner/go/internal/comments"

// blankSwift returns src with its comments and string literals blanked.
func blankSwift(src string) string {
	code, _ := blanked(src, false)
	return code
}

// blankComments returns src with only its comments blanked, so a word a
// string literal holds, such as a command name, still reads.
func blankComments(src string) string {
	code, _ := blanked(src, true)
	return code
}

// blanked copies src with every comment, and every string literal unless
// keepStrings is set, turned to spaces, and lists each literal it passes,
// inner ones in an interpolation before the literal holding them.
// Newlines stay, so offsets and line numbers still match.
func blanked(src string, keepStrings bool) (string, []Literal) {
	out := []byte(src)
	blank := func(start, end int) {
		for i := start; i < end; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	var literals []Literal
	comments.WalkSwift(src, comments.SwiftVisitor{
		Comment: blank,
		String: func(start, end int, text string) {
			literals = append(literals, Literal{Start: start, End: end, Text: text})
			if !keepStrings {
				blank(start, end)
			}
		},
	})
	return string(out), literals
}
