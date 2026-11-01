package swift

import "sort"

// Literal is one string literal in Swift source. Start and End are its
// byte offsets, delimiters and raw-string hashes included, and Line is the
// line it opens on. Source is the literal as written, and Text holds its
// words: the characters between the delimiters, without its
// interpolations and with escapes read.
type Literal struct {
	Start, End, Line int
	Source, Text     string
}

// Literals returns src with its comments and string literals blanked, and
// every literal in source order. A literal inside another's interpolation
// is listed too, after the literal holding it opens.
func Literals(src string) (code string, literals []Literal) {
	code, literals = blanked(src, false)
	sort.SliceStable(literals, func(i, j int) bool { return literals[i].Start < literals[j].Start })
	line, at := 1, 0
	for i := range literals {
		for ; at < literals[i].Start; at++ {
			if src[at] == '\n' {
				line++
			}
		}
		literals[i].Line = line
		literals[i].Source = src[literals[i].Start:literals[i].End]
	}
	return code, literals
}
