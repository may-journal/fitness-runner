package swift

import (
	"regexp"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/lang"
)

// activity matches the call that opens a named XCTest activity, up to the
// colon before its name.
var activity = regexp.MustCompile(`\bXCTContext\s*\.\s*runActivity\s*\(\s*named\s*:`)

// activities lists the cases in the body of the function whose signature
// ends at from: each XCTContext.runActivity(named:) given a literal name.
func activities(file, code string, from int, literals []Literal) []lang.Case {
	open := strings.IndexByte(code[from:], '{')
	if open < 0 {
		return nil
	}
	open += from
	end := bodyEnd(code, open)
	var out []lang.Case
	for _, m := range activity.FindAllStringIndex(code[open:end], -1) {
		if l, ok := literalAt(code, open+m[1], literals); ok {
			out = append(out, lang.NewCase(l.Text, file, l.Line))
		}
	}
	return out
}

// literalAt returns the next literal from offset at, when only white space
// comes before it. The blanked code shows each literal as spaces.
func literalAt(code string, at int, literals []Literal) (Literal, bool) {
	i := sort.Search(len(literals), func(i int) bool { return literals[i].Start >= at })
	if i == len(literals) || strings.TrimSpace(code[at:literals[i].Start]) != "" {
		return Literal{}, false
	}
	return literals[i], true
}

// bodyEnd returns the offset of the brace closing the one open at open, or
// the code's end.
func bodyEnd(code string, open int) int {
	depth := 0
	for i := open; i < len(code); i++ {
		switch code[i] {
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return len(code)
}
