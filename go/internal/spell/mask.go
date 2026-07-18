package spell

// cspell:ignore anrvtbf

import (
	"regexp"
	"strings"
	"unicode"
)

// ignorePatterns are the RE2 ports of the cspell default ignoreRegExpList
// (Urls, Email, CommitHash, hex and hash forms, UUID, Unicode refs). The
// lookahead-dependent Base64/PublicKey/SshRsa patterns are deliberately not
// replicated; commit-hash "must contain a digit" is enforced in code below.
var ignorePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:https?|ftp)://[^\s"]+`),
	regexp.MustCompile(`(?i)<?\b[\w.+-]{1,128}@\w{1,63}(?:\.\w{1,63}){1,4}\b>?`),
	regexp.MustCompile(`(?i)\[[0-9a-f]{7,}\]`),
	regexp.MustCompile(`(?i)\b0x[0-9a-f_]+n?\b`),
	regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b`),
	regexp.MustCompile(`(?i)\bsha\d+-[a-z0-9+/]{25,}={0,3}`),
	regexp.MustCompile(`(?i)(?:\b(?:sha\d+|md5|base64|crypt|bcrypt|scrypt|security-token|assertion)[-,:$=]|#code/)[-\w/+%.]{25,}={0,3}`),
	regexp.MustCompile(`(?i)\bU\+[0-9a-f]{4,5}(?:-[0-9a-f]{4,5})?`),
	regexp.MustCompile(`(?i)\b[0-9a-fx]{8}-[0-9a-fx]{4}-[0-9a-fx]{4}-[0-9a-fx]{4}-[0-9a-fx]{12}\b`),
	// Escape sequences in string literals: cspell's word splitter treats \n in
	// "\nconst" as a break so "const" is judged alone; masking the escape gets
	// the same judgment.
	regexp.MustCompile(`(?i)\\(?:[anrvtbf]|[xu][0-9a-f]+)`),
}

// hexRun is cspell's CommitHash pattern minus its lookahead: the "contains a
// digit" requirement is checked in code.
var hexRun = regexp.MustCompile(`(?i)\b(?:0x)?[0-9a-f]{7,}\b`)

// maskPatterns marks every byte matched by the default ignore patterns.
func maskPatterns(text string, masked []bool) {
	for _, re := range ignorePatterns {
		for _, m := range re.FindAllStringIndex(text, -1) {
			mark(masked, m[0], m[1])
		}
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
func maskDirectives(text string, masked []bool) map[string]struct{} {
	d := &directiveMasker{
		text:       text,
		masked:     masked,
		lines:      lineOffsets(text),
		words:      make(map[string]struct{}),
		blockStart: -1,
	}
	for n := range d.lines {
		d.maskLine(n)
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
