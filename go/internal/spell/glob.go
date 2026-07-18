package spell

// cspell:ignore segs

import "strings"

// IgnoreMatcher matches cspell.json ignorePaths entries against
// slash-separated relative paths with gitignore-like semantics: a bare name
// (no '/') matches any path segment at any depth, a pattern with '/' matches
// from the root with ** crossing directories, and a pattern that matches a
// directory ignores everything beneath it. path/filepath.Match cannot do **,
// hence this matcher.
type IgnoreMatcher struct {
	bare  []string   // segment patterns, matched against every path segment
	paths [][]string // anchored patterns, split into segments
}

// NewIgnoreMatcher compiles ignorePaths entries into a matcher.
func NewIgnoreMatcher(entries []string) *IgnoreMatcher {
	m := &IgnoreMatcher{}
	for _, e := range entries {
		e = strings.TrimSpace(e)
		e = strings.TrimSuffix(e, "/")
		anchored := strings.HasPrefix(e, "/")
		e = strings.TrimPrefix(e, "/")
		if e == "" {
			continue
		}
		if anchored || strings.Contains(e, "/") {
			m.paths = append(m.paths, strings.Split(e, "/"))
		} else {
			m.bare = append(m.bare, e)
		}
	}
	return m
}

// Matches reports whether the slash-separated relative path (file or
// directory) is ignored.
func (m *IgnoreMatcher) Matches(relPath string) bool {
	segs := strings.Split(relPath, "/")
	if m.matchesBare(segs) {
		return true
	}
	for _, pat := range m.paths {
		if matchSegments(pat, segs) {
			return true
		}
	}
	return false
}

// matchesBare reports whether any bare (segment-only) pattern matches any
// path segment.
func (m *IgnoreMatcher) matchesBare(segs []string) bool {
	for _, pat := range m.bare {
		for _, seg := range segs {
			if matchSegment(pat, seg) {
				return true
			}
		}
	}
	return false
}

// matchSegments matches pattern segments against path segments; "**" spans
// any number of segments, and an exhausted pattern matches (the path is the
// matched directory or lives beneath it).
func matchSegments(pat, path []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			return matchDoubleStar(pat, path)
		}
		if len(path) == 0 || !matchSegment(pat[0], path[0]) {
			return false
		}
		pat, path = pat[1:], path[1:]
	}
	return true
}

// matchDoubleStar matches a pattern whose head is "**" by letting it span any
// number of path segments, including none; a trailing "**" matches everything.
func matchDoubleStar(pat, path []string) bool {
	if len(pat) == 1 {
		return true
	}
	for i := 0; i <= len(path); i++ {
		if matchSegments(pat[1:], path[i:]) {
			return true
		}
	}
	return false
}

// matchSegment matches one glob segment ('*', '?', '[…]' classes) against
// one path segment; '*' and '?' never cross a '/' because segments contain
// none.
func matchSegment(pattern, s string) bool {
	return matchRunes([]rune(pattern), []rune(s))
}

func matchRunes(pat, s []rune) bool {
	for len(pat) > 0 {
		if pat[0] == '*' {
			return matchStar(pat, s)
		}
		var ok bool
		pat, s, ok = matchOne(pat, s)
		if !ok {
			return false
		}
	}
	return len(s) == 0
}

// matchStar matches a pattern whose head is '*' by letting it absorb any
// number of runes, including none.
func matchStar(pat, s []rune) bool {
	for skip := 0; skip <= len(s); skip++ {
		if matchRunes(pat[1:], s[skip:]) {
			return true
		}
	}
	return false
}

// matchOne consumes one rune of s against the non-'*' pattern head ('?', a
// '[…]' class, or a literal), returning both remainders and whether it
// matched; an empty s never matches.
func matchOne(pat, s []rune) (restPat, restS []rune, ok bool) {
	if len(s) == 0 {
		return nil, nil, false
	}
	switch pat[0] {
	case '?':
		return pat[1:], s[1:], true
	case '[':
		rest, matched := matchClass(pat, s)
		return rest, s[1:], matched
	default:
		return pat[1:], s[1:], pat[0] == s[0]
	}
}

// matchClass matches s[0] against the character class opening at pat[0]=='[';
// it returns the pattern remainder past ']' and whether s[0] matched. A
// malformed class (no closing bracket) matches a literal '['.
func matchClass(pat, s []rune) (rest []rune, ok bool) {
	end := classEnd(pat)
	if end < 0 {
		return matchLiteralBracket(pat, s)
	}
	if len(s) == 0 || !classMatches(pat[1:end], s[0]) {
		return nil, false
	}
	return pat[end+1:], true
}

// classEnd returns the index of the ']' closing the class at pat[0]=='[', or
// -1 if none; a ']' at index 1 is a class member, not the closer.
func classEnd(pat []rune) int {
	for i := 1; i < len(pat); i++ {
		if pat[i] == ']' && i > 1 {
			return i
		}
	}
	return -1
}

// matchLiteralBracket handles a malformed class (no closing ']'): the '[' is
// matched as a literal character, consuming only itself from the pattern.
func matchLiteralBracket(pat, s []rune) (rest []rune, ok bool) {
	if len(s) > 0 && s[0] == '[' {
		return pat[1:], true
	}
	return nil, false
}

// classMatches reports whether r is matched by the class body (the runes
// between the brackets), honoring a leading '!' or '^' negation.
func classMatches(class []rune, r rune) bool {
	class, negate := stripNegation(class)
	for i := 0; i < len(class); {
		matched, next := classMemberMatch(class, i, r)
		if matched {
			return !negate
		}
		i = next
	}
	return negate
}

// stripNegation strips a leading '!' or '^' from a class body, reporting
// whether the class is negated.
func stripNegation(class []rune) ([]rune, bool) {
	if len(class) > 0 && (class[0] == '!' || class[0] == '^') {
		return class[1:], true
	}
	return class, false
}

// classMemberMatch checks the class member starting at i (an a-b range when a
// '-' with a bound follows, else a single rune) against r, returning whether
// it matched and the index just past the member.
func classMemberMatch(class []rune, i int, r rune) (matched bool, next int) {
	if i+2 < len(class) && class[i+1] == '-' {
		return class[i] <= r && r <= class[i+2], i + 3
	}
	return class[i] == r, i + 1
}
