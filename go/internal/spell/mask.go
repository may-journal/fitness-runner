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
	words := make(map[string]struct{})
	lines := lineOffsets(text)
	blockStart := -1
	for n := 0; n < len(lines); n++ {
		lineStart, lineEnd := lines[n], len(text)
		if n+1 < len(lines) {
			lineEnd = lines[n+1]
		}
		loc := directiveRe.FindStringIndex(text[lineStart:lineEnd])
		if loc == nil {
			continue
		}
		dirStart := lineStart + loc[0]
		rest := strings.ToLower(text[lineStart+loc[1] : lineEnd])
		switch {
		case keyword(rest, "disable-line"):
			mark(masked, lineStart, lineEnd)
		case keyword(rest, "disable-next"):
			mark(masked, dirStart, lineEnd)
			if n+1 < len(lines) {
				nextEnd := len(text)
				if n+2 < len(lines) {
					nextEnd = lines[n+2]
				}
				mark(masked, lines[n+1], nextEnd)
			}
		case keyword(rest, "disable"):
			if blockStart < 0 {
				blockStart = dirStart
			}
		case keyword(rest, "enable"):
			if blockStart >= 0 {
				mark(masked, blockStart, lineEnd)
				blockStart = -1
			} else {
				mark(masked, dirStart, lineStart+loc[1]+len("enable"))
			}
		case strings.HasPrefix(rest, "ignore") || strings.HasPrefix(rest, "word"):
			addIgnoreWords(text[lineStart+loc[1]:lineEnd], words)
			mark(masked, dirStart, lineEnd)
		}
	}
	if blockStart >= 0 {
		mark(masked, blockStart, len(text))
	}
	return words
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
