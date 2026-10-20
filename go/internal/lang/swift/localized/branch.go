package localized

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
)

// branchStart returns where the first argument of a call begins when the
// expression before i makes a literal a branch of it: after a ternary's
// ? or :, or a ?? . ok is false for a literal that is no branch, or whose
// expression is not a call's first argument.
func branchStart(code string, i int) (int, bool) {
	if i == 0 || !strings.ContainsRune("?:", rune(code[i-1])) {
		return 0, false
	}
	open := unmatchedOpen(code, i-1)
	if open < 0 {
		return 0, false
	}
	label := labeled.FindStringSubmatch(code[open+1:])
	return open + 1, label == nil || label[1] == "verbatim"
}

// bracketDelta is how a byte changes bracket depth, read backward: a
// closing bracket opens a level and an opening one closes it.
var bracketDelta = map[byte]int{')': 1, ']': 1, '}': 1, '(': -1, '[': -1, '{': -1}

// unmatchedOpen returns the offset of the parenthesis that opens the
// call holding end, or -1 when a comma, bracket or brace comes first.
func unmatchedOpen(code string, end int) int {
	depth := 0
	for j := end; j >= 0; j-- {
		depth += bracketDelta[code[j]]
		if open, done := openFound(code[j], depth, j); done {
			return open
		}
	}
	return -1
}

// openFound judges one byte at offset j while scanning back at depth.
func openFound(c byte, depth, j int) (int, bool) {
	switch {
	case depth < 0:
		return parenAt(c, j), true
	case c == ',' && depth == 0:
		return -1, true
	}
	return 0, false
}

// parenAt returns j for a parenthesis and -1 for another bracket.
func parenAt(c byte, j int) int {
	if c == '(' {
		return j
	}
	return -1
}

// labeled matches an argument that starts with a label, such as
// systemImage: or verbatim:, and captures the label. A ternary's colon
// follows an expression, so only a name straight after the parenthesis is
// a label. Only a verbatim: argument is shown as words.
var labeled = regexp.MustCompile(`^\s*([\p{L}_][\p{L}\p{N}_]*)[ \t]*:`)

// anchors returns, for each literal, the offset whose context decides
// what shows it: the literal's own start, or for a branch of a ternary
// or ?? inside another literal's interpolation, the start of that
// outermost literal, since the words appear where it does. Another
// literal inside an interpolation, such as a dictionary key, gets -1: it
// is data, not words the view shows.
func anchors(src string, literals []swift.Literal) []int {
	at := make([]int, len(literals))
	host := -1
	for i, lit := range literals {
		at[i] = lit.Start
		if host >= 0 && lit.Start < literals[host].End {
			at[i] = -1
			if branchBefore(src, lit.Start) {
				at[i] = literals[host].Start
			}
		} else {
			host = i
		}
	}
	return at
}

// branchBefore reports a ? or ternary : just before start in src.
func branchBefore(src string, start int) bool {
	i := skipSpaceBack(src, start)
	return i > 0 && (src[i-1] == '?' || src[i-1] == ':' && !labelColon(src, i) && ternaryColon(src, i))
}

// ternaryColon reports a colon just before i that has a ? before it in
// the same brackets. A dictionary literal's colon has none.
func ternaryColon(src string, i int) bool {
	depth := 0
	for j := i - 2; j >= 0 && depth >= 0; j-- {
		depth += bracketDelta[src[j]]
		if src[j] == '?' && depth == 0 {
			return true
		}
	}
	return false
}

// labelColon reports a colon just before i that follows an argument label
// after a parenthesis or comma, rather than a ternary's else branch.
func labelColon(code string, i int) bool {
	label, at := nameBack(code, skipSpaceBack(code, i-1))
	before := skipSpaceBack(code, at)
	return label != "" && before > 0 && strings.ContainsRune("(,", rune(code[before-1]))
}

// isData reports a test literal used as plain data rather than as a label
// to find: one assigned with =, or passed under an argument label, such as
// caption: "Photo". A comparison, a subscript and a positional argument
// still read as labels.
func isData(src string, start int) bool {
	i := skipSpaceBack(src, start)
	if i == 0 {
		return false
	}
	switch src[i-1] {
	case '=':
		return i < 2 || !strings.ContainsRune("=!<>", rune(src[i-2]))
	case ':':
		return !branchBefore(src, start)
	}
	return false
}
