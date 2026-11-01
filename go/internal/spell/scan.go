package spell

// cspell:ignore anrvtbf pell

import "strings"

// scanExact masks the simple default ignore patterns without a regexp, each
// exactly as its case-insensitive cspell regexp would match:
//
//	\[[0-9a-f]{7,}\]                          bracketed hash
//	\b0x[0-9a-f_]+n?\b                        hex literal
//	#[0-9a-f]{3,8}\b                          hex color
//	\bU\+[0-9a-f]{4,5}(?:-[0-9a-f]{4,5})?     Unicode reference
//	\\(?:[anrvtbf]|[xu][0-9a-f]+)             string escape
//
// A string escape is masked since cspell's splitter treats \n in "\nconst"
// as a word break.
func scanExact(line string, masked []bool) {
	scanAt(line, masked, '[', bracketHashEnd)
	scanAt(line, masked, '0', hexLiteralEnd)
	scanAt(line, masked, '#', hexColorEnd)
	scanAt(line, masked, 'U', unicodeRefEnd)
	scanAt(line, masked, 'u', unicodeRefEnd)
	scanAt(line, masked, '\\', escapeEnd)
}

// scanAt tries matchEnd at each occurrence of first, resuming after each
// match as FindAll does.
func scanAt(line string, masked []bool, first byte, matchEnd func(line string, i int) int) {
	for i := 0; i < len(line); {
		j := strings.IndexByte(line[i:], first)
		if j < 0 {
			return
		}
		i += j
		if end := matchEnd(line, i); end > i {
			mark(masked, i, end)
			i = end
		} else {
			i++
		}
	}
}

// bracketHashEnd matches \[[0-9a-f]{7,}\] at i.
func bracketHashEnd(line string, i int) int {
	n := runLen(line, i+1, isHexDigit)
	if n >= 7 && byteAt(line, i+1+n) == ']' {
		return i + 2 + n
	}
	return i
}

// hexLiteralEnd matches \b0x[0-9a-f_]+n?\b at i. The run and its n are
// word bytes, so only its full length can end at a word boundary.
func hexLiteralEnd(line string, i int) int {
	if wordByteAt(line, i-1) || lower(byteAt(line, i+1)) != 'x' {
		return i
	}
	n := runLen(line, i+2, func(c byte) bool { return c == '_' || isHexDigit(c) })
	if n == 0 {
		return i
	}
	return hexLiteralBoundary(line, i, i+2+n)
}

// hexLiteralBoundary ends a hex literal at i past an n suffix or at end,
// whichever a word boundary follows, else at i.
func hexLiteralBoundary(line string, i, end int) int {
	if lower(byteAt(line, end)) == 'n' && !wordByteAt(line, end+1) {
		return end + 1
	}
	if wordByteAt(line, end) {
		return i
	}
	return end
}

// hexColorEnd matches #[0-9a-f]{3,8}\b at i. A shorter prefix of the run
// is followed by a hex digit, never a boundary, so only the full run can
// match.
func hexColorEnd(line string, i int) int {
	n := runLen(line, i+1, isHexDigit)
	if n < 3 || n > 8 || wordByteAt(line, i+1+n) {
		return i
	}
	return i + 1 + n
}

// unicodeRefEnd returns the end of the Unicode reference starting at i, for example U+1F600 or U+1F600-1F64F.
func unicodeRefEnd(line string, i int) int {
	if wordByteAt(line, i-1) || byteAt(line, i+1) != '+' {
		return i
	}
	n := runLen(line, i+2, isHexDigit)
	if n < 4 {
		return i
	}
	end := i + 2 + min(n, 5)
	return end + rangeTailLen(line, end)
}

// rangeTailLen returns the length of a -[0-9a-f]{4,5} range end at i, or 0.
func rangeTailLen(line string, i int) int {
	if byteAt(line, i) != '-' {
		return 0
	}
	if m := runLen(line, i+1, isHexDigit); m >= 4 {
		return 1 + min(m, 5)
	}
	return 0
}

// escapeEnd matches \\(?:[anrvtbf]|[xu][0-9a-f]+) at i.
func escapeEnd(line string, i int) int {
	c := lower(byteAt(line, i+1))
	switch {
	case strings.IndexByte("anrvtbf", c) >= 0:
		return i + 2
	case c == 'x' || c == 'u':
		if n := runLen(line, i+2, isHexDigit); n > 0 {
			return i + 2 + n
		}
	}
	return i
}

// indexPell returns the offset of the first "pell" in text in any case, or
// -1. None of p, e and l folds to a non-ASCII rune.
func indexPell(text string) int {
	for i := 0; i+4 <= len(text); i++ {
		if lower(text[i]) == 'p' && strings.EqualFold(text[i:i+4], "pell") {
			return i
		}
	}
	return -1
}

// hasHashRun reports whether line holds 25 bytes in a row that a hash
// pattern's body can match: letters, digits, and - _ / + % . — or any
// non-ASCII byte, since (?i) folds k and s to non-ASCII runes.
func hasHashRun(line string) bool {
	return hasRun(line, 25, func(c byte) bool {
		return c >= 0x80 || isWordByte(c) || strings.IndexByte("-/+%.", c) >= 0
	})
}

// isUUIDByte reports whether c can appear in a UUID match: [0-9a-fx-].
func isUUIDByte(c byte) bool {
	return c == '-' || lower(c) == 'x' || isHexDigit(c)
}

// hasRun reports whether line holds n bytes in a row that keep holds for.
func hasRun(line string, n int, keep func(byte) bool) bool {
	run := 0
	for i := 0; i < len(line); i++ {
		if !keep(line[i]) {
			run = 0
			continue
		}
		if run++; run >= n {
			return true
		}
	}
	return false
}

// runLen counts the bytes from i on that keep holds for.
func runLen(line string, i int, keep func(byte) bool) int {
	n := 0
	for i+n < len(line) && keep(line[i+n]) {
		n++
	}
	return n
}

// byteAt returns line[i], or 0 outside the line.
func byteAt(line string, i int) byte {
	if i < 0 || i >= len(line) {
		return 0
	}
	return line[i]
}

// wordByteAt reports whether line[i] is an ASCII word byte, the only kind
// \b counts; outside the line it is not.
func wordByteAt(line string, i int) bool {
	return isWordByte(byteAt(line, i))
}

func isWordByte(c byte) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= lower(c) && lower(c) <= 'z'
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= lower(c) && lower(c) <= 'f'
}

// lower folds an ASCII letter to lowercase. Other bytes may change too, but
// never into a letter, so comparing the result with a letter stays exact.
func lower(c byte) byte {
	return c | 0x20
}
