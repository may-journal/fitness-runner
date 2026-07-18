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
// lowercased with curly apostrophes normalized to ASCII.
type Checker struct {
	words map[string]struct{}
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
	if utf8.RuneCountInString(token) < minWordLength ||
		c.capsSuffixOk(token, docWords) || c.known(token, docWords) {
		return nil
	}
	var issues []Issue
	for _, sub := range splitSubWords(token) {
		word := token[sub.start:sub.end]
		if utf8.RuneCountInString(word) < minWordLength || c.known(word, docWords) ||
			c.capsSuffixOk(word, docWords) || isRepeatedRune(word) {
			continue
		}
		line, col := position(text, lines, tok.start+sub.start)
		issues = append(issues, Issue{Word: word, Line: line, Col: col})
	}
	return issues
}

// known reports whether word (any case, possibly possessive) is in the
// dictionary or the document's inline ignore words.
func (c *Checker) known(word string, docWords map[string]struct{}) bool {
	folded := foldWord(word)
	for _, w := range []string{folded, strings.TrimSuffix(folded, "'s"), strings.TrimSuffix(folded, "'")} {
		if _, ok := c.words[w]; ok {
			return true
		}
		if _, ok := docWords[w]; ok {
			return true
		}
	}
	return false
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

// extendedSpans finds cspell's "possible words": maximal unmasked runs of
// letters, digits, and joining punctuation (. + - _ and quotes) — the shape
// compound dictionary entries take.
func extendedSpans(text string, masked []bool) []span {
	var out []span
	start := -1
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !masked[i] && isExtendedChar(r) {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			out = append(out, span{start, i})
			start = -1
		}
		i += size
	}
	if start >= 0 {
		out = append(out, span{start, len(text)})
	}
	return out
}

func isExtendedChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) ||
		r == '_' || r == '`' || r == '.' || r == '+' || r == '-' || isApostrophe(r)
}

// tokenSpans extracts word tokens: runs of letters (plus combining marks)
// with internal apostrophes, skipping masked bytes — cspell's word regexp as
// a scanner.
func tokenSpans(text string, masked []bool) []span {
	var out []span
	start := -1
	end := func(at int) {
		if start >= 0 {
			out = append(out, span{start, at})
			start = -1
		}
	}
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch {
		case masked[i]:
			end(i)
		case unicode.IsLetter(r):
			if start < 0 {
				start = i
			}
		case start >= 0 && unicode.IsMark(r):
			// marks extend a started token
		case start >= 0 && isApostrophe(r) && letterFollows(text, masked, i+size):
			// internal apostrophe: don't and CSS's stay single tokens
		default:
			end(i)
		}
		i += size
	}
	end(len(text))
	return out
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
	"s": true, "ing": true, "ies": true, "es": true, "ings": true, "ed": true, "ning": true,
}

// splitSubWords splits a token at camelCase boundaries, mirroring cspell's
// break rules: before an Upper that follows a lower (myWord -> my Word) and
// before the last Upper of an Upper run followed by a lower (HTMLElement ->
// HTML Element) unless what follows is just an English suffix. Byte spans
// are relative to the token.
func splitSubWords(token string) []span {
	runes := []rune(token)
	offs := make([]int, len(runes)+1)
	for i, r := range runes {
		offs[i+1] = offs[i] + utf8.RuneLen(r)
	}
	var out []span
	start := 0
	for k := 1; k < len(runes); k++ {
		if !isUpper(runes[k]) {
			continue
		}
		lowerBefore := unicode.IsLower(runes[k-1])
		upperRunEnd := isUpper(runes[k-1]) && k+1 < len(runes) && unicode.IsLower(runes[k+1]) &&
			!camelSuffixes[lowerRun(runes[k+1:])]
		if lowerBefore || upperRunEnd {
			out = append(out, span{offs[start], offs[k]})
			start = k
		}
	}
	return append(out, span{offs[start], offs[len(runes)]})
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
