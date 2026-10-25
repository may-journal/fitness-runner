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
	b := &blanker{src: src, out: []byte(src)}
	for b.i < len(b.src) {
		b.step()
	}
	sort.SliceStable(b.literals, func(i, j int) bool { return b.literals[i].Start < b.literals[j].Start })
	line, at := 1, 0
	for i := range b.literals {
		for ; at < b.literals[i].Start; at++ {
			if src[at] == '\n' {
				line++
			}
		}
		b.literals[i].Line = line
		b.literals[i].Source = src[b.literals[i].Start:b.literals[i].End]
	}
	return string(b.out), b.literals
}
