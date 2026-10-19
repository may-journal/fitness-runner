package spell

// cspell:ignore pell

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// ignorePatterns are the RE2 ports of the cspell default ignoreRegExpList
// (Urls, Email, CommitHash, hash forms, UUID) that stay regexps. The
// lookahead-dependent Base64/PublicKey/SshRsa patterns are deliberately not
// replicated; commit-hash "must contain a digit" is enforced in code below.
// The simpler patterns (hex forms, Unicode refs, escapes) are exact scanners
// in scan.go.
//
// Each pattern pairs with mayMatch, an exact test that holds on every line
// the regexp can match, so a line that fails it skips the regexp and the
// mask never changes. Under (?i), Go folds k and s to the Kelvin sign and
// long s, so the run tests count every non-ASCII byte as a possible member.
type ignorePattern struct {
	re       *regexp.Regexp
	mayMatch func(line string) bool
}

var ignorePatterns = []ignorePattern{
	{regexp.MustCompile(`(?i)(?:https?|ftp)://[^\s"]+`), func(line string) bool {
		return strings.Contains(line, "://")
	}},
	{regexp.MustCompile(`(?i)<?\b[\w.+-]{1,128}@\w{1,63}(?:\.\w{1,63}){1,4}\b>?`), func(line string) bool {
		return strings.IndexByte(line, '@') >= 0
	}},
	{regexp.MustCompile(`(?i)\bsha\d+-[a-z0-9+/]{25,}={0,3}`), hasHashRun},
	{regexp.MustCompile(`(?i)(?:\b(?:sha\d+|md5|base64|crypt|bcrypt|scrypt|security-token|assertion)[-,:$=]|#code/)[-\w/+%.]{25,}={0,3}`), hasHashRun},
	{regexp.MustCompile(`(?i)\b[0-9a-fx]{8}-[0-9a-fx]{4}-[0-9a-fx]{4}-[0-9a-fx]{4}-[0-9a-fx]{12}\b`), func(line string) bool {
		return hasRun(line, 36, isUUIDByte)
	}},
}

// hexRun is cspell's CommitHash pattern minus its lookahead: the "contains a
// digit" requirement is checked in code.
var hexRun = regexp.MustCompile(`(?i)\b(?:0x)?[0-9a-f]{7,}\b`)

// maskPatterns marks every byte matched by the default ignore patterns.
// No ignore pattern matches a newline, so each line is matched alone.
func maskPatterns(text string, masked []bool) {
	offset := 0
	for line := range strings.SplitSeq(text, "\n") {
		maskPatternLine(line, masked[offset:offset+len(line)])
		offset += len(line) + 1
	}
}

func maskPatternLine(line string, masked []bool) {
	for _, pattern := range ignorePatterns {
		if pattern.mayMatch(line) {
			maskMatches(pattern.re, line, masked)
		}
	}
	scanExact(line, masked)
	maskHexRuns(line, masked)
}

func maskMatches(re *regexp.Regexp, text string, masked []bool) {
	for _, m := range re.FindAllStringIndex(text, -1) {
		mark(masked, m[0], m[1])
	}
}

// maskHexRuns masks runs of seven or more hex digits holding a digit; a
// line without such a run skips the regexp.
func maskHexRuns(text string, masked []bool) {
	if !hasRun(text, 7, isHexDigit) || !strings.ContainsAny(text, "0123456789") {
		return
	}
	for _, m := range hexRun.FindAllStringIndex(text, -1) {
		if strings.ContainsAny(text[m[0]:m[1]], "0123456789") {
			mark(masked, m[0], m[1])
		}
	}
}

// directiveRe finds a cspell in-document directive prefix (cspell:,
// spell-checker:, …); what follows decides the directive.
var directiveRe = regexp.MustCompile(`(?i)\bc?spell(?:-?checker)?::?[ \t]*`)

// maskDirectives handles in-document cspell directives line by line: ignore
// lists feed the returned word set, disable-line/disable-next/disable…enable
// mask regions, and every recognized directive masks its own text. Unknown
// text after a bare "cspell:" is left alone, exactly like cspell.
//
// Every directive holds "pell" in some case (Go folds the s of spell to a
// long s too), so only lines holding it are matched, and a text without it
// returns at once. lines are the text's line offsets.
func maskDirectives(text string, masked []bool, lines []int) map[string]struct{} {
	at := indexPell(text)
	if at < 0 {
		return nil
	}
	d := &directiveMasker{
		text:       text,
		masked:     masked,
		lines:      lines,
		words:      make(map[string]struct{}),
		blockStart: -1,
	}
	for at >= 0 {
		n := sort.SearchInts(lines, at+1) - 1
		d.maskLine(n)
		_, next := d.lineSpan(n)
		if at = indexPell(text[next:]); at >= 0 {
			at += next
		}
	}
	if d.blockStart >= 0 {
		mark(masked, d.blockStart, len(text))
	}
	return d.words
}

// directiveMasker carries the state of one maskDirectives pass: the text and
// mask being written, the text's line offsets, the collected ignore words,
// and the start of an open disable…enable block (-1 when none is open).
type directiveMasker struct {
	text       string
	masked     []bool
	lines      []int
	words      map[string]struct{}
	blockStart int
}

// directiveAt locates one recognized directive: its line index, the line's
// byte range, and the byte offsets of the directive prefix (dirStart) and of
// the text following it (restStart).
type directiveAt struct {
	n, lineStart, lineEnd, dirStart, restStart int
}

// lineSpan returns the [start, end) byte range of line n, where end includes
// the trailing newline (or is the end of text for the last line).
func (d *directiveMasker) lineSpan(n int) (start, end int) {
	end = len(d.text)
	if n+1 < len(d.lines) {
		end = d.lines[n+1]
	}
	return d.lines[n], end
}

// maskLine finds a directive prefix on line n and applies the directive that
// follows it; lines without one are untouched.
func (d *directiveMasker) maskLine(n int) {
	lineStart, lineEnd := d.lineSpan(n)
	loc := directiveRe.FindStringIndex(d.text[lineStart:lineEnd])
	if loc == nil {
		return
	}
	dir := directiveAt{
		n:         n,
		lineStart: lineStart,
		lineEnd:   lineEnd,
		dirStart:  lineStart + loc[0],
		restStart: lineStart + loc[1],
	}
	rest := strings.ToLower(d.text[dir.restStart:lineEnd])
	if !d.applyDisable(rest, dir) {
		d.applyEnableOrIgnore(rest, dir)
	}
}

// applyDisable handles the disable family (disable-line, disable-next,
// disable), reporting whether rest named one of them.
func (d *directiveMasker) applyDisable(rest string, dir directiveAt) bool {
	switch {
	case keyword(rest, "disable-line"):
		mark(d.masked, dir.lineStart, dir.lineEnd)
	case keyword(rest, "disable-next"):
		d.maskNextLine(dir)
	case keyword(rest, "disable"):
		if d.blockStart < 0 {
			d.blockStart = dir.dirStart
		}
	default:
		return false
	}
	return true
}

// applyEnableOrIgnore handles enable (closing an open block or masking just
// itself) and the ignore/word directives that also collect words.
func (d *directiveMasker) applyEnableOrIgnore(rest string, dir directiveAt) {
	switch {
	case keyword(rest, "enable"):
		d.closeBlock(dir)
	case strings.HasPrefix(rest, "ignore") || strings.HasPrefix(rest, "word"):
		addIgnoreWords(d.text[dir.restStart:dir.lineEnd], d.words)
		mark(d.masked, dir.dirStart, dir.lineEnd)
	}
}

// maskNextLine masks a disable-next directive from its start through the end
// of the following line, when one exists.
func (d *directiveMasker) maskNextLine(dir directiveAt) {
	mark(d.masked, dir.dirStart, dir.lineEnd)
	if dir.n+1 < len(d.lines) {
		_, nextEnd := d.lineSpan(dir.n + 1)
		mark(d.masked, d.lines[dir.n+1], nextEnd)
	}
}

// closeBlock ends an open disable…enable block by masking the whole block, or
// masks just the enable keyword when no block is open, exactly like cspell.
func (d *directiveMasker) closeBlock(dir directiveAt) {
	if d.blockStart >= 0 {
		mark(d.masked, d.blockStart, dir.lineEnd)
		d.blockStart = -1
		return
	}
	mark(d.masked, dir.dirStart, dir.restStart+len("enable"))
}

// keyword reports whether rest starts with kw at a word boundary, mirroring
// the \b in cspell's guard regexps ("disable-next-line" matches disable-next,
// "disabled" matches nothing).
func keyword(rest, kw string) bool {
	if !strings.HasPrefix(rest, kw) {
		return false
	}
	if len(rest) == len(kw) {
		return true
	}
	next := rune(rest[len(kw)])
	return !unicode.IsLetter(next) && !unicode.IsDigit(next)
}

// addIgnoreWords parses "cspell:ignore word word…" (and word/words variants)
// into the document ignore set; other keywords such as ignoreRegExp only
// mask.
func addIgnoreWords(rest string, words map[string]struct{}) {
	fields := strings.FieldsFunc(rest, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	if len(fields) == 0 {
		return
	}
	switch strings.ToLower(fields[0]) {
	case "ignore", "word", "words":
		for _, w := range fields[1:] {
			words[foldWord(w)] = struct{}{}
		}
	}
}

// mark sets masked over [start, end).
func mark(masked []bool, start, end int) {
	for i := start; i < end && i < len(masked); i++ {
		masked[i] = true
	}
}
