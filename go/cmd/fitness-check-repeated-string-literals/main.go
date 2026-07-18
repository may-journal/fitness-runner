// Command fitness-check-repeated-string-literals fails when any string
// literal value appears three or more times across the repo's JS/TS source
// files — the Go port of the repeated-string-literals check. A hand lexer
// extracts single- and double-quoted literals, skipping line/block comments,
// regex literals, template literals, and module specifiers (the string after
// import/require/from); idiomatic tokens (buffer encodings, stdio modes,
// typeof results, directive prologues) are never flagged. Each duplicated
// value earns one error listing its locations, most-repeated first.
//
// One deviation from the TypeScript original: the allow list comes from
// .fitnessrc.json via internal/conf, so any config load error — including a
// legacy .fitnessrc.js/.ts, which this binary cannot evaluate — degrades to
// an empty allow list instead of reading the configured baseline.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// sourceExtensions are the source file extensions scanned — the TS
// SOURCE_FILE_EXTENSIONS set. Test/spec/bench files are excluded separately.
var sourceExtensions = []string{".cts", ".mts", ".ts", ".cjs", ".js", ".mjs", ".tsx"}

// minLength: string literals shorter than this many runes are ignored as
// noise (single chars, tiny tokens).
const minLength = 3

// minOccurrences: a value must appear at least this many times (repo-wide)
// to be flagged.
const minOccurrences = 3

// maxLocations caps the locations listed per duplicated value to keep an
// error line readable.
const maxLocations = 5

// fixtureFileRe matches a test/spec/bench source file (fixtures there
// repeat strings intentionally).
var fixtureFileRe = regexp.MustCompile(`\.(?:bench|spec|test)\.(?:c|m)?[jt]sx?$`)

// idiomaticValues are JS/Node tokens where the literal IS the clearest
// spelling — extracting a constant would hurt readability. Closed sets only:
// buffer encodings, child_process stdio modes, and typeof results. Anything
// domain-specific is still flagged.
var idiomaticValues = map[string]bool{
	// Buffer/stream encodings (BufferEncoding)
	"ascii":     true,
	"base64":    true,
	"base64url": true,
	"binary":    true,
	"hex":       true,
	"latin1":    true,
	"ucs2":      true,
	"utf8":      true,
	"utf-8":     true,
	"utf16le":   true,
	// child_process stdio modes
	"inherit":    true,
	"ignore":     true,
	"overlapped": true,
	"pipe":       true,
	// typeof results
	"bigint":    true,
	"boolean":   true,
	"function":  true,
	"number":    true,
	"object":    true,
	"string":    true,
	"symbol":    true,
	"undefined": true,
	// language directives (a directive prologue cannot be replaced by a constant)
	"use strict": true,
}

// moduleWords are words that, immediately before a string, mark it a module
// specifier to skip (import/require/from).
var moduleWords = map[string]bool{"from": true, "import": true, "require": true}

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "repeated-string-literals"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := findSourceFiles(root)
	scanned := make([]scannedFile, 0, len(files))
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		scanned = append(scanned, scannedFile{file: file, literals: scanStringLiterals(string(content))})
	}
	errs := findDuplicates(scanned, allowedValues(root))
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// findSourceFiles returns sorted relative paths of scanned source files
// (test/spec/bench excluded).
func findSourceFiles(root string) []string {
	var out []string
	for _, f := range walkfs.FilesByExt(root, sourceExtensions...) {
		if !fixtureFileRe.MatchString(f) {
			out = append(out, f)
		}
	}
	return out
}

// allowedValues returns the exact values allowed via .fitnessrc.json
// `repeatedStringLiterals.allow` (project baseline). Any load error — a
// legacy JS/TS config or malformed JSON — degrades to no allow list.
func allowedValues(root string) map[string]bool {
	cfg, err := conf.Load(root)
	if err != nil || cfg == nil {
		return nil
	}
	allow := make(map[string]bool, len(cfg.RepeatedStringLiterals.Allow))
	for _, v := range cfg.RepeatedStringLiterals.Allow {
		allow[v] = true
	}
	return allow
}

// scannedFile pairs one source file with the literals scanned from it.
type scannedFile struct {
	file     string
	literals []literal
}

// literal is a scanned string literal: its text and 1-based line.
type literal struct {
	line  int
	value string
}

// location is one occurrence of a duplicated value.
type location struct {
	file string
	line int
}

// groupByValue groups scanned literals by value into their locations across
// all files, remembering first-seen value order for deterministic iteration.
func groupByValue(scanned []scannedFile) (map[string][]location, []string) {
	byValue := make(map[string][]location)
	var order []string
	for _, sf := range scanned {
		for _, lit := range sf.literals {
			if _, seen := byValue[lit.value]; !seen {
				order = append(order, lit.value)
			}
			byValue[lit.value] = append(byValue[lit.value], location{file: sf.file, line: lit.line})
		}
	}
	return byValue, order
}

// formatDuplicate formats one duplicated value and its locations into an
// error message.
func formatDuplicate(value string, locs []location) string {
	shown := make([]string, 0, maxLocations)
	for _, l := range locs[:min(len(locs), maxLocations)] {
		shown = append(shown, fmt.Sprintf("%s:%d", l.file, l.line))
	}
	where := strings.Join(shown, ", ")
	if more := len(locs) - len(shown); more > 0 {
		where = fmt.Sprintf("%s, +%d more", where, more)
	}
	return fmt.Sprintf("\"%s\" appears %d times (%s) — extract a shared constant", value, len(locs), where)
}

// findDuplicates returns one error message per non-allowed value that occurs
// >= minOccurrences times across all files, most-repeated first, then
// alphabetical by value for stable output.
func findDuplicates(scanned []scannedFile, allow map[string]bool) []string {
	byValue, order := groupByValue(scanned)
	var duplicated []string
	for _, value := range order {
		if len(byValue[value]) >= minOccurrences && !allow[value] {
			duplicated = append(duplicated, value)
		}
	}
	sort.Slice(duplicated, func(a, b int) bool {
		if len(byValue[duplicated[a]]) != len(byValue[duplicated[b]]) {
			return len(byValue[duplicated[a]]) > len(byValue[duplicated[b]])
		}
		return duplicated[a] < duplicated[b]
	})
	out := make([]string, 0, len(duplicated))
	for _, value := range duplicated {
		out = append(out, formatDuplicate(value, byValue[value]))
	}
	return out
}

// cursor is the mutable scan position and accumulated state of the hand lexer.
type cursor struct {
	content  string
	i        int
	lastWord string // last completed identifier — used to detect module specifiers
	line     int
	out      []literal
	prev     byte   // last significant byte (0 at start) — used for regex-vs-division
	word     string // identifier currently being read
}

// scanStringLiterals extracts single- and double-quoted string literals from
// source, skipping line/block comments, regex literals, and template
// literals, and dropping module specifiers (the string after
// import/require/from). Template literals are out of scope in v1.
func scanStringLiterals(content string) []literal {
	cur := &cursor{content: content, line: 1}
	for cur.i < len(content) {
		cur.step()
	}
	return cur.out
}

// isDivision reports whether a `/` following prev is a division operator,
// not the start of a regex literal.
func isDivision(prev byte) bool {
	if isWordChar(prev) {
		return true
	}
	switch prev {
	case ')', ']', '}', '\'', '"', '`':
		return true
	}
	return false
}

// isASCIILetter reports an ASCII letter (a…z, either case).
func isASCIILetter(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

// isWordChar reports whether b can be part of an identifier/keyword word.
func isWordChar(b byte) bool {
	return isASCIILetter(b) || b >= '0' && b <= '9' || b == '_' || b == '$'
}

// isSpace reports inline whitespace (newlines are handled separately for
// line counting).
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r'
}

// isFlag reports a regex flag byte (a…z, either case).
func isFlag(b byte) bool {
	return isASCIILetter(b)
}

// at reports whether the source at the cursor starts with token.
func (cur *cursor) at(token string) bool {
	return strings.HasPrefix(cur.content[cur.i:], token)
}

// endWord finalizes the in-progress word into lastWord.
func (cur *cursor) endWord() {
	if cur.word != "" {
		cur.lastWord = cur.word
		cur.word = ""
	}
}

// skipLineComment advances past a `//` line comment (the newline is handled
// on the next step).
func (cur *cursor) skipLineComment() {
	cur.i += 2
	for cur.i < len(cur.content) && cur.content[cur.i] != '\n' {
		cur.i++
	}
}

// skipBlockComment advances past a block comment (slash-star … star-slash),
// counting the newlines inside it.
func (cur *cursor) skipBlockComment() {
	cur.i += 2
	for cur.i < len(cur.content) && !cur.at("*/") {
		if cur.content[cur.i] == '\n' {
			cur.line++
		}
		cur.i++
	}
	cur.i += 2
}

// classAfter tracks character-class state across one regex-body byte: `[`
// enters a class, `]` leaves it, anything else keeps the current state.
func classAfter(inClass bool, ch byte) bool {
	if ch == '[' {
		return true
	}
	if ch == ']' {
		return false
	}
	return inClass
}

// scanRegexChar consumes one byte of a regex body, updating the char-class
// state; reports done at the closing `/` (consumed) or at a newline (left
// unconsumed — the unterminated-body bail).
func (cur *cursor) scanRegexChar(inClass *bool) (done bool) {
	ch := cur.content[cur.i]
	if ch == '\\' {
		cur.i += 2
		return false
	}
	if ch == '\n' {
		return true // unterminated — bail safely
	}
	cur.i++
	if ch == '/' && !*inClass {
		return true
	}
	*inClass = classAfter(*inClass, ch)
	return false
}

// scanRegexBody advances through a regex body up to and including the
// closing `/` (honoring char classes; an unterminated body bails at the
// newline).
func (cur *cursor) scanRegexBody() {
	inClass := false
	for cur.i < len(cur.content) {
		if cur.scanRegexChar(&inClass) {
			return
		}
	}
}

// skipRegex advances past a regex literal and its flags.
func (cur *cursor) skipRegex() {
	cur.i++ // opening /
	cur.scanRegexBody()
	for cur.i < len(cur.content) && isFlag(cur.content[cur.i]) {
		cur.i++
	}
	cur.prev = '/'
	cur.lastWord = ""
}

// recordLiteral records a scanned literal unless it's a module specifier,
// below the length floor, or idiomatic.
func (cur *cursor) recordLiteral(value string, line int, isModule bool) {
	if isModule || utf8.RuneCountInString(value) < minLength || idiomaticValues[value] {
		return
	}
	cur.out = append(cur.out, literal{line: line, value: value})
}

// readEscape consumes a backslash escape inside a string literal, appending
// the escaped byte to value raw (the lexer never decodes escape sequences; a
// trailing backslash at EOF appends nothing).
func (cur *cursor) readEscape(value *strings.Builder) {
	if cur.i+1 < len(cur.content) {
		value.WriteByte(cur.content[cur.i+1])
	}
	cur.i += 2
}

// readString reads a single- or double-quoted string literal, honoring
// escapes, and records it.
func (cur *cursor) readString() {
	cur.endWord()
	quote := cur.content[cur.i]
	startLine := cur.line
	isModule := moduleWords[cur.lastWord]
	cur.i++
	var value strings.Builder
	for cur.i < len(cur.content) && cur.content[cur.i] != quote {
		if cur.content[cur.i] == '\\' {
			cur.readEscape(&value)
			continue
		}
		if cur.content[cur.i] == '\n' {
			cur.line++
		}
		value.WriteByte(cur.content[cur.i])
		cur.i++
	}
	cur.i++ // closing quote
	cur.prev = quote
	cur.lastWord = ""
	cur.recordLiteral(value.String(), startLine, isModule)
}

// skipTemplate skips a template literal (out of scope in v1), counting the
// newlines inside it.
func (cur *cursor) skipTemplate() {
	cur.endWord()
	cur.i++
	for cur.i < len(cur.content) && cur.content[cur.i] != '`' {
		if cur.content[cur.i] == '\\' {
			cur.i += 2
			continue
		}
		if cur.content[cur.i] == '\n' {
			cur.line++
		}
		cur.i++
	}
	cur.i++
	cur.prev = '`'
	cur.lastWord = ""
}

// stepSlash handles a `/`: line comment, block comment, regex literal, or
// division operator.
func (cur *cursor) stepSlash() {
	if cur.at("//") {
		cur.skipLineComment()
		return
	}
	if cur.at("/*") {
		cur.skipBlockComment()
		return
	}
	if !isDivision(cur.prev) {
		cur.skipRegex()
		return
	}
	cur.endWord()
	cur.prev = '/'
	cur.i++
}

// stepText handles a non-quote, non-slash byte: newline, whitespace, word
// char, or other delimiter.
func (cur *cursor) stepText(ch byte) {
	if ch == '\n' {
		cur.endWord()
		cur.line++
		cur.i++
		return
	}
	if isSpace(ch) {
		cur.endWord()
		cur.i++
		return
	}
	if isWordChar(ch) {
		cur.word += string(ch)
		cur.prev = ch
		cur.i++
		return
	}
	cur.endWord()
	cur.prev = ch
	cur.i++
}

// step advances the cursor by one token, dispatching on the current byte.
func (cur *cursor) step() {
	ch := cur.content[cur.i]
	switch ch {
	case '/':
		cur.stepSlash()
	case '\'', '"':
		cur.readString()
	case '`':
		cur.skipTemplate()
	default:
		cur.stepText(ch)
	}
}
