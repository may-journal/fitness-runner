// Package spell is the native engine behind the cspell fitness check: the
// soul of cspell — no unknown words, and the project dictionary always wins —
// without the cspell engine underneath.
//
// The pipeline mirrors what cspell observably does: in-document directives
// (cspell:ignore, cspell:disable/enable, …) and the default ignore patterns
// (URLs, emails, hex runs, hashes, …) mask regions of the text; the rest
// splits into extended tokens (letters, digits, ., +, -, _, quotes) whose
// compound dictionary entries win whole (prettier-plugin-packagejson), then
// into words on non-letter boundaries and camelCase transitions with
// English-suffix handling for ALL-CAPS runs (URLs, APIs). Words shorter than
// four runes are never flagged; lookup is case-insensitive against the
// embedded base dictionaries plus any added project words, tolerating a
// trailing possessive ('s).
package spell

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// minWordLength mirrors cspell's default: shorter words are never flagged.
const minWordLength = 4

// Issue is one unknown word at a 1-based line and column (rune columns).
type Issue struct {
	Word string
	Line int
	Col  int
}

// Checker holds the dictionary a text is checked against. Words are stored
// lowercased with curly apostrophes normalized to ASCII; base, when set, is
// an extra dictionary consulted in place (the embedded wordlists in
// production — see NewEmbeddedChecker).
type Checker struct {
	words map[string]struct{}
	base  baseDict
}

// baseDict is the lookup side of a base dictionary: Lookup receives
// already-folded words and reports membership.
type baseDict interface {
	Lookup(word string) bool
}

// NewChecker builds a Checker over the union of the given word sets.
func NewChecker(sets ...map[string]struct{}) *Checker {
	size := 0
	for _, set := range sets {
		size += len(set)
	}
	words := make(map[string]struct{}, size)
	for _, set := range sets {
		for w := range set {
			words[w] = struct{}{}
		}
	}
	return &Checker{words: words}
}

// AddWords adds words (e.g. the project cspell.json words list) to the
// dictionary, normalized like every other entry.
func (c *Checker) AddWords(words []string) {
	for _, w := range words {
		w = foldWord(strings.TrimSpace(w))
		if w != "" {
			c.words[w] = struct{}{}
		}
	}
}

// CheckText spell-checks one document and returns its unknown words in
// document order.
func (c *Checker) CheckText(text string) []Issue {
	text = strings.TrimPrefix(text, "\ufeff")
	masked := make([]bool, len(text))
	docWords := maskDirectives(text, masked)
	maskPatterns(text, masked)
	lines := lineOffsets(text)
	var issues []Issue
	for _, ext := range extendedSpans(text, masked) {
		if c.knownCompound(text[ext.start:ext.end], docWords) {
			continue
		}
		for _, tok := range tokenSpans(text[ext.start:ext.end], masked[ext.start:ext.end]) {
			tok = span{tok.start + ext.start, tok.end + ext.start}
			issues = append(issues, c.checkToken(text, tok, docWords, lines)...)
		}
	}
	return issues
}

// checkToken judges one word token: known as a whole (or as an ALL-CAPS word
// with an English suffix), or judged sub-word by sub-word after camelCase
// splitting.
func (c *Checker) checkToken(text string, tok span, docWords map[string]struct{}, lines []int) []Issue {
	token := text[tok.start:tok.end]
	if c.tokenExempt(token, docWords) {
		return nil
	}
	var issues []Issue
	for _, sub := range splitSubWords(token) {
		word := token[sub.start:sub.end]
		if c.subWordExempt(word, docWords) {
			continue
		}
		line, col := position(text, lines, tok.start+sub.start)
		issues = append(issues, Issue{Word: word, Line: line, Col: col})
	}
	return issues
}

// tokenExempt reports whether a whole token needs no sub-word judging: too
// short to flag, an ALL-CAPS+suffix form, or known outright.
func (c *Checker) tokenExempt(token string, docWords map[string]struct{}) bool {
	return utf8.RuneCountInString(token) < minWordLength ||
		c.capsSuffixOk(token, docWords) || c.known(token, docWords)
}

// subWordExempt reports whether a sub-word escapes flagging: too short to
// flag, known, an ALL-CAPS+suffix form, or a single repeated rune.
func (c *Checker) subWordExempt(word string, docWords map[string]struct{}) bool {
	return utf8.RuneCountInString(word) < minWordLength || c.known(word, docWords) ||
		c.capsSuffixOk(word, docWords) || isRepeatedRune(word)
}

// known reports whether word (any case, possibly possessive) is in the
// dictionary or the document's inline ignore words.
func (c *Checker) known(word string, docWords map[string]struct{}) bool {
	folded := foldWord(word)
	for _, w := range []string{folded, strings.TrimSuffix(folded, "'s"), strings.TrimSuffix(folded, "'")} {
		if c.hasWord(w, docWords) {
			return true
		}
	}
	return false
}

// hasWord reports whether one folded word form is in any dictionary: the
// checker's own words, the document's, or the base dictionary.
func (c *Checker) hasWord(w string, docWords map[string]struct{}) bool {
	if _, ok := c.words[w]; ok {
		return true
	}
	if _, ok := docWords[w]; ok {
		return true
	}
	return c.base != nil && c.base.Lookup(w)
}

// knownCompound reports whether an extended token, trimmed of surrounding
// quotes and punctuation, is a dictionary compound entry
// (prettier-plugin-packagejson, package.json, …).
func (c *Checker) knownCompound(token string, docWords map[string]struct{}) bool {
	trimmed := strings.Trim(foldWord(token), "`'.+-")
	return trimmed != "" && c.known(trimmed, docWords)
}

// capsSuffixRe is cspell's ALL-CAPS-with-English-suffix exception: URLs and
// APIs pass when their base is known (or too short to judge).
var capsSuffixRe = regexp.MustCompile(`^((?:\p{Lu}\p{M}?){2,})['’]?(?:s|ing|ies|es|ings|ize|ed|ning)$`)

// capsSuffixOk reports whether word is an ALL-CAPS run plus a common English
// suffix whose base is known or shorter than the flaggable minimum.
func (c *Checker) capsSuffixOk(word string, docWords map[string]struct{}) bool {
	m := capsSuffixRe.FindStringSubmatch(word)
	if m == nil {
		return false
	}
	return c.known(m[1], docWords) || utf8.RuneCountInString(m[1]) < minWordLength
}

// isRepeatedRune mirrors cspell's repeated-character filter: xxxx or XXXX is
// never flagged.
func isRepeatedRune(word string) bool {
	runes := []rune(strings.ToLower(word))
	for _, r := range runes[1:] {
		if r != runes[0] {
			return false
		}
	}
	return len(runes) >= minWordLength
}

// foldWord lowercases a word and normalizes curly apostrophes, the shared
// normalization for dictionary entries and lookups.
func foldWord(word string) string {
	return strings.ToLower(strings.ReplaceAll(word, "’", "'"))
}

// span is a half-open byte range into the checked text.
type span struct {
	start, end int
}

// runScanner accumulates half-open spans of consecutive run bytes: extend
// grows (or opens) the current run, end closes any open one.
type runScanner struct {
	spans []span
	start int // -1 when no run is open
}

// extend continues the open run through byte i, opening one there if needed.
func (s *runScanner) extend(i int) {
	if s.start < 0 {
		s.start = i
	}
}

// end closes any open run just before byte at; without one it is a no-op.
func (s *runScanner) end(at int) {
	if s.start >= 0 {
		s.spans = append(s.spans, span{s.start, at})
		s.start = -1
	}
}

// extendedSpans finds cspell's "possible words": maximal unmasked runs of
// letters, digits, and joining punctuation (. + - _ and quotes) — the shape
// compound dictionary entries take.
func extendedSpans(text string, masked []bool) []span {
	sc := runScanner{start: -1}
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !masked[i] && isExtendedChar(r) {
			sc.extend(i)
		} else {
			sc.end(i)
		}
		i += size
	}
	sc.end(len(text))
	return sc.spans
}

func isExtendedChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) ||
		strings.ContainsRune("_`.+-'’", r)
}

// tokenSpans extracts word tokens: runs of letters (plus combining marks)
// with internal apostrophes, skipping masked bytes — cspell's word regexp as
// a scanner.
func tokenSpans(text string, masked []bool) []span {
	sc := runScanner{start: -1}
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch {
		case unmaskedLetter(masked, i, r):
			sc.extend(i)
		case sc.start >= 0 && tokenExtends(text, masked, i, r, size):
			// marks and internal apostrophes extend a started token:
			// don't and CSS's stay single tokens
		default:
			sc.end(i)
		}
		i += size
	}
	sc.end(len(text))
	return sc.spans
}

// unmaskedLetter reports whether the rune r at byte i is an unmasked letter,
// the only thing that can start (or plainly continue) a word token.
func unmaskedLetter(masked []bool, i int, r rune) bool {
	return !masked[i] && unicode.IsLetter(r)
}

// tokenExtends reports whether an already-started token continues through the
// non-letter rune r (of byte width size) at byte i: unmasked combining marks
// always extend, an unmasked apostrophe extends when an unmasked letter
// follows.
func tokenExtends(text string, masked []bool, i int, r rune, size int) bool {
	if masked[i] {
		return false
	}
	return unicode.IsMark(r) || (isApostrophe(r) && letterFollows(text, masked, i+size))
}

func isApostrophe(r rune) bool {
	return r == '\'' || r == '’'
}

// letterFollows reports whether an unmasked letter starts at byte offset i.
func letterFollows(text string, masked []bool, i int) bool {
	if i >= len(text) || masked[i] {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[i:])
	return unicode.IsLetter(r)
}

// camelSuffixes suppress the Upper-run camel break before an English suffix,
// keeping LSTMs or URLs whole instead of splitting off "Ms"/"Ls".
var camelSuffixes = map[string]bool{
	// cspell:ignore ings ning — suffix fragments, not words
	"s": true, "ing": true, "ies": true, "es": true, "ings": true, "ed": true, "ning": true,
}

// splitSubWords splits a token at camelCase boundaries, mirroring cspell's
// break rules: before an Upper that follows a lower (myWord -> my Word) and
// before the last Upper of an Upper run followed by a lower (HTMLElement ->
// HTML Element) unless what follows is just an English suffix. Byte spans
// are relative to the token.
func splitSubWords(token string) []span {
	runes := []rune(token)
	offs := runeOffsets(runes)
	var out []span
	start := 0
	for k := 1; k < len(runes); k++ {
		if camelBreakAt(runes, k) {
			out = append(out, span{offs[start], offs[k]})
			start = k
		}
	}
	return append(out, span{offs[start], offs[len(runes)]})
}

// runeOffsets returns the byte offset of every rune boundary in runes,
// including the final end-of-token offset (so it has len(runes)+1 entries).
func runeOffsets(runes []rune) []int {
	offs := make([]int, len(runes)+1)
	for i, r := range runes {
		offs[i+1] = offs[i] + utf8.RuneLen(r)
	}
	return offs
}

// camelBreakAt reports whether cspell breaks a token before runes[k] (k >= 1):
// at an Upper following a lower, or at the Upper ending an Upper run.
func camelBreakAt(runes []rune, k int) bool {
	if !isUpper(runes[k]) {
		return false
	}
	return unicode.IsLower(runes[k-1]) || upperRunEnd(runes, k)
}

// upperRunEnd reports whether runes[k] is the last Upper of an Upper run
// followed by a lower that is more than an English suffix (HTMLElement breaks
// before Element; LSTMs keeps Ms attached).
func upperRunEnd(runes []rune, k int) bool {
	return isUpper(runes[k-1]) && k+1 < len(runes) && unicode.IsLower(runes[k+1]) &&
		!camelSuffixes[lowerRun(runes[k+1:])]
}

// lowerRun returns the maximal leading run of lowercase letters.
func lowerRun(runes []rune) string {
	for i, r := range runes {
		if !unicode.IsLower(r) {
			return string(runes[:i])
		}
	}
	return string(runes)
}

func isUpper(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsTitle(r)
}

// lineOffsets returns the byte offset of each line start.
func lineOffsets(text string) []int {
	out := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			out = append(out, i+1)
		}
	}
	return out
}

// position converts a byte offset to a 1-based line and rune column.
func position(text string, lines []int, offset int) (line, col int) {
	i := sort.Search(len(lines), func(n int) bool { return lines[n] > offset }) - 1
	return i + 1, utf8.RuneCountInString(text[lines[i]:offset]) + 1
}
