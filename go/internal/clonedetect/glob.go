package clonedetect

import "strings"

// MatchGlob reports whether a minimatch-style pattern matches the
// slash-separated relative path. Supported syntax: "**" as a whole segment
// matches any number of segments including none; "*" matches any run of
// characters within a segment — including a leading dot (dot:true
// semantics); "?" matches exactly one byte. No braces or character
// classes: the jscpd ignore set needs none.
func MatchGlob(pattern, path string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchSegments(pat, path []string) bool {
	if len(pat) == 0 {
		return len(path) == 0
	}
	if pat[0] == "**" {
		return matchDoublestar(pat, path)
	}
	if len(path) == 0 || !matchSeg(pat[0], path[0]) {
		return false
	}
	return matchSegments(pat[1:], path[1:])
}

// matchDoublestar resolves a leading "**" pattern segment: it matches zero
// path segments (dropping the "**") or consumes one segment and retries
// the same pattern.
func matchDoublestar(pat, path []string) bool {
	if matchSegments(pat[1:], path) {
		return true
	}
	if len(path) == 0 {
		return false
	}
	return matchSegments(pat, path[1:])
}

// matchSeg is classic iterative wildcard matching of one path segment,
// backtracking to the most recent "*" on mismatch.
func matchSeg(pat, s string) bool {
	pi, si := 0, 0
	star, mark := -1, 0
	for si < len(s) {
		switch {
		case segConsumes(pat, pi, s[si]):
			pi++
			si++
		case segAtStar(pat, pi):
			star, mark = pi, si
			pi++
		case star >= 0:
			pi = star + 1
			mark++
			si = mark
		default:
			return false
		}
	}
	return tailStars(pat, pi)
}

// segConsumes reports whether the pattern byte at pi matches subject byte
// c: '?' matches any byte, otherwise the bytes must be equal.
func segConsumes(pat string, pi int, c byte) bool {
	return pi < len(pat) && (pat[pi] == '?' || pat[pi] == c)
}

// segAtStar reports whether the pattern byte at pi is the '*' wildcard.
func segAtStar(pat string, pi int) bool {
	return pi < len(pat) && pat[pi] == '*'
}

// tailStars reports whether the pattern from pi onward is all '*' — each
// matches the empty remainder, so an exhausted subject still matches.
func tailStars(pat string, pi int) bool {
	for pi < len(pat) && pat[pi] == '*' {
		pi++
	}
	return pi == len(pat)
}
