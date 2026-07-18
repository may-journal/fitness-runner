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
	for _, pat := range m.bare {
		for _, seg := range segs {
			if matchSegment(pat, seg) {
				return true
			}
		}
	}
	for _, pat := range m.paths {
		if matchSegments(pat, segs) {
			return true
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
		if len(path) == 0 || !matchSegment(pat[0], path[0]) {
			return false
		}
		pat, path = pat[1:], path[1:]
	}
	return true
}

// matchSegment matches one glob segment ('*', '?', '[…]' classes) against
// one path segment; '*' and '?' never cross a '/' because segments contain
// none.
func matchSegment(pattern, s string) bool {
	return matchRunes([]rune(pattern), []rune(s))
}

func matchRunes(pat, s []rune) bool {
	for len(pat) > 0 {
		switch pat[0] {
		case '*':
			for skip := 0; skip <= len(s); skip++ {
				if matchRunes(pat[1:], s[skip:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			pat, s = pat[1:], s[1:]
		case '[':
			rest, ok := matchClass(pat, s)
			if !ok {
				return false
			}
			pat, s = rest, s[1:]
		default:
			if len(s) == 0 || pat[0] != s[0] {
				return false
			}
			pat, s = pat[1:], s[1:]
		}
	}
	return len(s) == 0
}

// matchClass matches s[0] against the character class opening at pat[0]=='[';
// it returns the pattern remainder past ']' and whether s[0] matched. A
// malformed class (no closing bracket) matches a literal '['.
func matchClass(pat, s []rune) (rest []rune, ok bool) {
	end := -1
	for i := 1; i < len(pat); i++ {
		if pat[i] == ']' && i > 1 {
			end = i
			break
		}
	}
	if end < 0 {
		if len(s) > 0 && s[0] == '[' {
			return pat[1:], true
		}
		return nil, false
	}
	if len(s) == 0 {
		return nil, false
	}
	class, negate := pat[1:end], false
	if len(class) > 0 && (class[0] == '!' || class[0] == '^') {
		negate, class = true, class[1:]
	}
	matched := false
	for i := 0; i < len(class); i++ {
		if i+2 < len(class) && class[i+1] == '-' {
			if class[i] <= s[0] && s[0] <= class[i+2] {
				matched = true
			}
			i += 2
		} else if class[i] == s[0] {
			matched = true
		}
	}
	if matched == negate {
		return nil, false
	}
	return pat[end+1:], true
}
