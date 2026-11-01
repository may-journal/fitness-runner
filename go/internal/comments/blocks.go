package comments

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Kind says where a comment block sits.
type Kind string

const (
	// Doc is a block on lines of its own, such as one above a declaration.
	Doc Kind = "doc"
	// Inline is a comment after code on its line.
	Inline Kind = "inline"
)

// Block is one comment block, its 1-based first and last lines, and its text without markers.
type Block struct {
	Path string
	Line int
	End  int
	Lang string
	Kind Kind
	Text string
	// InBody marks a comment in a function body or after code, as Place sets it.
	InBody bool
	// Next is the trimmed code line after an own-line comment, as Place sets it.
	Next string
}

// note is one comment as written, markers included, before adjacent notes join
// into blocks; Code marks a comment a toolchain reads.
type note struct {
	line, end, col int
	text           string
	inline, code   bool
}

// langs maps a file extension to its language name.
var langs = map[string]string{
	".go": "go", ".swift": "swift",
	".ts": "typescript", ".tsx": "typescript", ".mts": "typescript", ".cts": "typescript",
	".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".java": "java", ".kt": "kotlin", ".kts": "kotlin", ".rs": "rust",
	".sh": "shell", ".bash": "shell", ".zsh": "shell", ".yml": "yaml", ".yaml": "yaml",
}

// lexers holds the lexer settings of each C-style language.
var lexers = map[string]Lexer{
	"typescript": {Templates: true},
	"javascript": {Templates: true},
	"java":       {},
	"kotlin":     {Nested: true},
	"rust":       {Nested: true, Lifetimes: true, RawStrings: true},
}

// readers reads each language's comments.
var readers = map[string]func(src []byte) []note{
	"go":         goNotes,
	"swift":      swiftNotes,
	"shell":      shellNotes,
	"yaml":       yamlNotes,
	"typescript": cReader(lexers["typescript"]),
	"javascript": cReader(lexers["javascript"]),
	"java":       cReader(lexers["java"]),
	"kotlin":     cReader(lexers["kotlin"]),
	"rust":       cReader(lexers["rust"]),
}

// shebang names a shell interpreter on a file's first line, which makes a
// file with no extension a shell script.
var shebang = regexp.MustCompile(`^#!.*\b(sh|bash|zsh)\b`)

// Lang names the language of the file at rel holding src, or "" when no
// reader covers it.
func Lang(rel string, src []byte) string {
	ext := path.Ext(rel)
	if ext == "" && shebang.Match(src) {
		return "shell"
	}
	return langs[ext]
}

// Files returns the comment blocks of paths under root in path order, skipping generated and unread files.
func Files(root string, paths []string) ([]Block, error) {
	var blocks []Block
	for _, rel := range walkfs.WithoutGenerated(root, paths) {
		if !covered(rel) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return blocks, err
		}
		blocks = append(blocks, Source(rel, src)...)
	}
	return blocks, nil
}

// covered reports a path whose extension a reader covers, or one with
// none, which a shebang may make a shell script.
func covered(rel string) bool {
	ext := path.Ext(rel)
	return ext == "" || langs[ext] != ""
}

// Source returns the comment blocks of one file. It skips tool directives
// and a license header.
func Source(rel string, src []byte) []Block {
	lang := Lang(rel, src)
	read, ok := readers[lang]
	if !ok {
		return nil
	}
	blocks := withoutLicense(join(read(src)))
	for i := range blocks {
		blocks[i].Path, blocks[i].Lang = rel, lang
	}
	return blocks
}

// lines splits src into lines without their endings.
func lines(src []byte) []string {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	return strings.Split(text, "\n")
}

// at is the note for the comment from byte from to to of line n.
func at(n int, line string, from, to int) note {
	return note{line: n, end: n, col: from, text: line[from:to], inline: codeBefore(line, from)}
}

// codeBefore reports code before byte at on its line.
func codeBefore(line string, at int) bool {
	return strings.TrimSpace(line[:at]) != ""
}

// cReader reads C-style source through a lexer with lex's settings.
func cReader(lex Lexer) func(src []byte) []note {
	return func(src []byte) []note { return cNotes(lines(src), lex) }
}

// cNotes reads the comments of C-style lines through lex.
func cNotes(src []string, lex Lexer) []note {
	r := &cRead{}
	for i, line := range src {
		for _, sp := range lex.Line(line) {
			r.span(i+1, line, sp)
		}
	}
	if r.open != nil {
		r.notes = append(r.notes, *r.open)
	}
	return r.notes
}

// cRead collects notes span by span, holding a block comment open over
// lines.
type cRead struct {
	notes []note
	open  *note
}

func (r *cRead) span(n int, line string, sp Span) {
	switch sp.Part {
	case LineComment:
		r.notes = append(r.notes, at(n, line, sp.From, sp.To))
	case BlockComment:
		r.block(n, line, sp)
	}
}

func (r *cRead) block(n int, line string, sp Span) {
	if r.open == nil {
		first := at(n, line, sp.From, sp.To)
		r.open = &first
	} else {
		r.open.text += "\n" + line[sp.From:sp.To]
		r.open.end = n
	}
	if !sp.Open {
		r.notes, r.open = append(r.notes, *r.open), nil
	}
}

// offsets turns byte offsets of src into notes, through the offset each
// line starts at.
type offsets struct {
	src    string
	starts []int
}

func newOffsets(src string) offsets {
	starts := []int{0}
	for i := range len(src) {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return offsets{src: src, starts: starts}
}

// note is the note for the comment from byte start to end.
func (o offsets) note(start, end int) note {
	line := sort.SearchInts(o.starts, start+1)
	lineStart := o.starts[line-1]
	text := o.src[start:end]
	return note{line: line, end: line + strings.Count(text, "\n"), col: start - lineStart, text: text, inline: codeBefore(o.src[lineStart:], start-lineStart)}
}

// swiftNotes reads the comments of Swift source through WalkSwift.
func swiftNotes(src []byte) []note {
	offs := newOffsets(string(src))
	var notes []note
	WalkSwift(offs.src, SwiftVisitor{
		Comment: func(start, end int) { notes = append(notes, offs.note(start, end)) },
		String:  func(int, int, string) {},
	})
	return notes
}

// join drops directives and joins line comments on adjacent lines of
// their own into one block. A block comment or an inline comment stands
// alone.
func join(notes []note) []Block {
	var blocks []Block
	var prev note
	for _, n := range notes {
		if directive(n.text) {
			continue
		}
		if continues(prev, n) {
			last := &blocks[len(blocks)-1]
			last.Text += "\n" + strip(n.text)
			last.End = n.end
		} else {
			blocks = append(blocks, Block{Line: n.line, End: n.end, Kind: kind(n), Text: strip(n.text)})
		}
		prev = n
	}
	return blocks
}

// continues reports a note that carries on the block prev ended on the
// line before.
func continues(prev, n note) bool {
	return n.line == prev.end+1 && ownLine(prev) && ownLine(n)
}

// ownLine reports a // or # comment on a line of its own.
func ownLine(n note) bool {
	return !n.inline && (strings.HasPrefix(n.text, "//") || strings.HasPrefix(n.text, "#"))
}

func kind(n note) Kind {
	if n.inline {
		return Inline
	}
	return Doc
}

// goDirective matches a Go directive such as //go:build, //line, //export or //extern.
var goDirective = regexp.MustCompile(`^//([a-z0-9]+:[a-z0-9]|line |export |extern )`)

// toolDirectives open a comment that speaks to a tool, not a reader.
// Each is matched in the case and form the tool reads, so prose that
// only starts with a tool name stays.
var toolDirectives = []string{
	"cspell:", "spell-checker:", "swiftlint:", "eslint-disable", "eslint-enable",
	"prettier-ignore", "jscpd:", "shellcheck disable=", "shellcheck enable=", "shellcheck source=",
	"shellcheck source-path=", "shellcheck shell=", "shellcheck external-sources=",
	"@ts-", "+build", "yaml-language-server:",
}

// directive reports a comment that is a tool directive or a shebang.
func directive(text string) bool {
	if goDirective.MatchString(text) || strings.HasPrefix(text, "#!") {
		return true
	}
	words := strings.TrimSpace(strings.Split(strip(text), "\n")[0])
	for _, d := range toolDirectives {
		if strings.HasPrefix(words, d) {
			return true
		}
	}
	return false
}

// licenseWords mark a file's leading block as a license header.
var licenseWords = regexp.MustCompile(`(?i)\b(copyright|spdx-license-identifier|licensed under)\b`)

// withoutLicense drops a license header: a file's first block, on lines
// of its own, naming a copyright or license.
func withoutLicense(blocks []Block) []Block {
	if len(blocks) > 0 && blocks[0].Kind == Doc && licenseWords.MatchString(blocks[0].Text) {
		return blocks[1:]
	}
	return blocks
}

// strip removes a comment's markers: // or # with one space after it, the
// extra / or ! of a doc comment, and a block's /* */ and leading stars.
func strip(text string) string {
	if body, ok := strings.CutPrefix(text, "/*"); ok {
		return stripBlock(strings.TrimSuffix(body, "*/"))
	}
	body, ok := strings.CutPrefix(text, "//")
	if ok {
		body = strings.TrimLeft(body, "/!")
	} else {
		body = strings.TrimPrefix(text, "#")
	}
	return strings.TrimRight(strings.TrimPrefix(body, " "), " \t\r")
}

// stripBlock trims each line of a block comment's body, drops its leading
// stars, and drops blank first and last lines.
func stripBlock(body string) string {
	rows := strings.Split(strings.TrimLeft(body, "*!"), "\n")
	for i, row := range rows {
		rows[i] = strings.TrimSpace(row)
		if i > 0 {
			rows[i] = strings.TrimSpace(strings.TrimPrefix(rows[i], "*"))
		}
	}
	return strings.Trim(strings.Join(rows, "\n"), "\n")
}
