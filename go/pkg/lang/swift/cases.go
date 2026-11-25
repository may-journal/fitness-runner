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

// signatureEnd matches the rest of a function's signature, up to the brace
// that opens its body, past no other brace or semicolon. With the check
// for a declaration keyword, a function with no body never takes the next
// member's.
var signatureEnd = regexp.MustCompile(`^[^{};]*?\{`)

// declaration matches a keyword that opens another declaration.
var declaration = regexp.MustCompile(`\b(?:func|var|let|class|struct|enum|protocol|extension|init|subscript)\b`)

// activities lists the cases in the body of the function whose signature
// ends at from: each XCTContext.runActivity(named:) given a literal name.
func activities(file, code string, from int, literals []Literal) []lang.Case {
	sig := signatureEnd.FindStringIndex(code[from:])
	if sig == nil || declaration.MatchString(code[from:from+sig[1]]) {
		return nil
	}
	open := from + sig[1] - 1
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
	if i == len(literals) || strings.TrimSpace(code[at:literals[i].Start]) != "" || interpolated(literals[i].Source) {
		return Literal{}, false
	}
	return literals[i], true
}

// interpolation matches an interpolation in a literal as written: \( in a
// plain string, or \#( in a raw one.
var interpolation = regexp.MustCompile(`\\#*\(`)

// interpolated reports a literal whose value is computed, so it names no
// one case.
func interpolated(source string) bool {
	return interpolation.MatchString(source)
}

// bodyEnd returns the offset of the brace closing the one open at open, or
// the code's end, counting braces as enclosing does.
func bodyEnd(code string, open int) int {
	stack := []int{-1}
	for i := open; i < len(code); i++ {
		if stack = braces(stack, code[i], i); len(stack) == 1 {
			return i
		}
	}
	return len(code)
}
