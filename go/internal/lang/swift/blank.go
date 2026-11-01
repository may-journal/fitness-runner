package swift

import "github.com/may-journal/fitness-runner/go/internal/comments"

// blankSwift returns a code view with its string literals blanked too.
func blankSwift(code string) string {
	out, _ := blanked(code)
	return out
}

// blanked copies a code view with its string literals turned to spaces, and
// lists each literal, keeping newlines. The walker still steps over the
// directives a code view keeps, so a quote in one opens no string.
func blanked(code string) (string, []Literal) {
	out := []byte(code)
	var literals []Literal
	comments.WalkSwift(code, comments.SwiftVisitor{
		Comment: func(int, int) {},
		String: func(start, end int, text string) {
			literals = append(literals, Literal{Start: start, End: end, Text: text})
			for i := start; i < end; i++ {
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
		},
	})
	return string(out), literals
}
