package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Rule IDs of the comment slop rules.
const (
	ruleCommentedOut = "commented-out-code"
	ruleRestates     = "restates-code"
	ruleDecoration   = "decoration"
	ruleNarration    = "change-narration"
	ruleHedging      = "hedging"
)

// maxRestateWords is the most words a comment may hold for `restates-code` to judge it.
const maxRestateWords = 5

// slopRule is one slop rule: its ID, the fix it names, and what it finds in a comment and its masked prose.
type slopRule struct {
	id, fix string
	find    func(b comments.Block, prose string) string
}

// slopRules are the slop rules in the order a comment's errors list them.
var slopRules = []slopRule{
	{ruleCommentedOut, "delete it, since git keeps old code", func(b comments.Block, _ string) string { return commentedOut(b) }},
	{ruleRestates, "delete it or say why", func(b comments.Block, _ string) string { return restates(b) }},
	{ruleDecoration, "delete it", func(b comments.Block, _ string) string { return decorated(b.Text) }},
	{ruleNarration, "say what holds today, since git keeps history", func(_ comments.Block, prose string) string {
		return matched(narration, prose, "narrates a change")
	}},
	{ruleHedging, "drop the word or state the fact", func(_ comments.Block, prose string) string {
		return matched(hedge, prose, "hedges")
	}},
}

// slopErrors reports each slop rule a comment breaks, reading prose from its Markdown blocks.
func slopErrors(name string, b comments.Block, blocks []block) []string {
	var errs []string
	prose := proseOf(blocks)
	for _, rule := range slopRules {
		if why := rule.find(b, prose); why != "" {
			errs = append(errs, fmt.Sprintf("%s: %s: %s; %s: %q", name, rule.id, why, rule.fix, snippet(b.Text)))
		}
	}
	return errs
}

// matched quotes the first match of re in prose that no participle follows, followed by what, or returns "".
func matched(re *regexp.Regexp, prose, what string) string {
	for _, m := range re.FindAllStringSubmatchIndex(prose, -1) {
		if !participle.MatchString(prose[m[1]:]) {
			return fmt.Sprintf("%q %s", prose[m[0]:m[1]], what)
		}
	}
	return ""
}

// participle opens text with a past participle, as in "previously stored" or "actually written", which describes today.
var participle = regexp.MustCompile(`^\s+[a-z]{3,}(ed|en)\b`)

// proseOf joins a comment's paragraphs and list items with code spans and quoted literals masked.
func proseOf(blocks []block) string {
	var parts []string
	for _, bl := range blocks {
		if bl.isList {
			parts = append(parts, bl.items...)
		} else {
			parts = append(parts, bl.text)
		}
	}
	return masked(strings.Join(parts, "\n"))
}

// singleQuoted is a one-line single-quoted literal, opened after a space so an apostrophe opens none.
var singleQuoted = regexp.MustCompile(`(^|[\s(])'[^'\n]*'`)

// masked is text with each code span, link and quoted literal replaced by one word.
func masked(text string) string {
	return mdx.MaskInline(singleQuoted.ReplaceAllString(quoted.ReplaceAllString(text, "quote"), "${1}quote"))
}

// narration matches words that tell how code changed rather than how it works.
var narration = regexp.MustCompile(`(?i)\b(previously|we changed|(it|this|that|we|they|which)\s+used\s+to|used\s+to\s+be)\b`)

// hedge matches filler that softens a statement without adding a fact.
var hedge = regexp.MustCompile(`(?i:\b(basically|simply|essentially|obviously|actually)\b)|\bI think\b`)

// banner matches a run of five or more of one decoration character.
var banner = regexp.MustCompile(`={5,}|-{5,}|\*{5,}|#{5,}`)

// decorated names the emoji, box drawing or banner in text's judged rows outside quotes and code spans, or returns "".
func decorated(text string) string {
	text = masked(strings.Join(judgedRows(text), "\n"))
	if why := symbol(text); why != "" {
		return why
	}
	if m := banner.FindString(text); m != "" {
		return fmt.Sprintf("a comment holds the banner %q", m)
	}
	return ""
}

// symbol names the first emoji or box-drawing character in text, or returns "".
func symbol(text string) string {
	for _, r := range text {
		if inRanges(r, pictographs) {
			return fmt.Sprintf("a comment holds the emoji %q", string(r))
		}
		if inRanges(r, boxDrawing) {
			return fmt.Sprintf("a comment holds the box-drawing character %q", string(r))
		}
	}
	return ""
}

// runeRange is an inclusive range of code points.
type runeRange struct{ lo, hi rune }

// pictographs are the emoji, dingbat and emoji-styled symbol code points; arrows such as → and ➔ stay prose.
var pictographs = []runeRange{
	{0x1F000, 0x1FAFF}, {0x2600, 0x2793}, {0x2B50, 0x2B50}, {0x2B55, 0x2B55},
	{0x203C, 0x203C}, {0x2049, 0x2049}, {0x2139, 0x2139}, {0x231A, 0x231B}, {0x2328, 0x2328},
	{0x23CF, 0x23CF}, {0x23E9, 0x23F3}, {0x23F8, 0x23FA}, {0x24C2, 0x24C2}, {0x3030, 0x3030},
	{0x303D, 0x303D}, {0x3297, 0x3297}, {0x3299, 0x3299}, {0xFE0F, 0xFE0F},
}

// boxDrawing is the Unicode box-drawing block.
var boxDrawing = []runeRange{{0x2500, 0x257F}}

// inRanges reports r inside one of ranges.
func inRanges(r rune, ranges []runeRange) bool {
	for _, rr := range ranges {
		if rr.lo <= r && r <= rr.hi {
			return true
		}
	}
	return false
}

// codeLine matches a line shaped like a statement, call, assignment, declaration or lone brace.
var codeLine = regexp.MustCompile(`^(` +
	`[})\]]+[;,)]*|` +
	`.*[{;]|` +
	`[\w.]+(\[[^\]]*\])?\s*(:=|[-+*/|&]?=)\s*\S.*|` +
	`[\w.]+(\+\+|--);?|` +
	`[\w.]+\(.*\)[;,]?|` +
	`(func|var|let|const|import|defer|fn|def|guard|struct|class|enum)\s+\S.*[=({:"].*|` +
	`return(\s+[\w.]+(\(.*\))?)?;?` +
	`)$`)

// keywords are code words that neither count toward prose nor break a run of prose words.
var keywords = map[string]bool{
	"if": true, "else": true, "for": true, "return": true, "range": true, "func": true, "var": true,
	"let": true, "const": true, "case": true, "go": true, "defer": true, "await": true, "async": true, "import": true,
}

// readsAsProse reports a line holding three plain words in a row, or two pairs of them.
func readsAsProse(row string) bool {
	pairs := 0
	for _, n := range wordRuns(row) {
		if n >= 3 {
			return true
		}
		if n == 2 {
			pairs++
		}
	}
	return pairs >= 2
}

// wordRuns returns the lengths of a line's runs of plain words outside quotes; keywords neither count nor break a run.
func wordRuns(row string) []int {
	runs := []int{0}
	for _, w := range strings.Fields(masked(row)) {
		w = strings.TrimRight(w, ",.;:!?")
		switch {
		case keywords[strings.ToLower(w)]:
		case plainWord(w):
			runs[len(runs)-1]++
		default:
			runs = append(runs, 0)
		}
	}
	return runs
}

// plainWord reports a word of letters, hyphens and apostrophes that starts with a letter.
func plainWord(w string) bool {
	return w != "" && unicode.IsLetter([]rune(w)[0]) &&
		strings.IndexFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && r != '-' && r != '\'' }) < 0
}

// commentedOut says how many lines of b look like code when at least half do, or returns "".
func commentedOut(b comments.Block) string {
	rows := judgedRows(b.Text)
	code := codeRows(b.Lang, rows)
	if code == 0 || 2*code < len(rows) {
		return ""
	}
	return fmt.Sprintf("%d of %d lines are code", code, len(rows))
}

// codeRows counts the rows that look like code in lang.
func codeRows(lang string, rows []string) int {
	code := 0
	for _, row := range rows {
		if looksLikeCode(lang, row) {
			code++
		}
	}
	return code
}

// judgedRows returns a comment's unindented lines outside fences, since an indented or fenced line is an example.
func judgedRows(text string) []string {
	var rows []string
	fenced := false
	for _, row := range strings.Split(text, "\n") {
		fenced = fenced != strings.HasPrefix(strings.TrimSpace(row), "```")
		if judgedRow(row, fenced) {
			rows = append(rows, row)
		}
	}
	return rows
}

// judgedRow reports a line with text, outside a fence, that is neither indented nor a fence marker.
func judgedRow(row string, fenced bool) bool {
	return !fenced && strings.TrimSpace(row) != "" && strings.TrimLeft(row, " \t") == row && !strings.HasPrefix(row, "```")
}

// looksLikeCode reports a line shaped like code and not like a sentence, parsing Go only when no pattern matches.
func looksLikeCode(lang, row string) bool {
	if readsAsProse(row) {
		return false
	}
	return codeLine.MatchString(strings.TrimSpace(row)) || lang == "go" && parsesAsGo(row)
}

// parsesAsGo reports a line that parses as Go statements holding code punctuation, labels excluded.
func parsesAsGo(row string) bool {
	if !strings.ContainsAny(row, "(){}[]=:;") {
		return false
	}
	file, err := parser.ParseFile(token.NewFileSet(), "", "package p\nfunc _() {\n"+row+"\n}\n", parser.SkipObjectResolution)
	return err == nil && !labeled(file)
}

// labeled reports a labeled statement in file, as a `Note:` lead-in parses.
func labeled(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		_, ok := n.(*ast.LabeledStmt)
		found = found || ok
		return !found
	})
	return found
}

// restates says why a one-line body comment of few words repeats the next code line, or returns "".
func restates(b comments.Block) string {
	if !oneLiner(b) {
		return ""
	}
	words := wordsOf(b.Text)
	if len(words) > maxRestateWords || !allIn(words, codeWords(b.Next)) {
		return ""
	}
	return "every word repeats the next code line"
}

// oneLiner reports a one-line comment on its own line above code in a function body, where no doc convention applies.
func oneLiner(b comments.Block) bool {
	return b.Kind == comments.Doc && b.InBody && b.Next != "" && !strings.Contains(b.Text, "\n")
}

// allIn reports words, at least one, each an article or a word of code.
func allIn(words []string, code map[string]bool) bool {
	for _, w := range words {
		if !code[stem(w)] && !articles[w] {
			return false
		}
	}
	return len(words) > 0
}

// articles are words a restating comment adds without saying more.
var articles = map[string]bool{"the": true, "a": true, "an": true}

// operatorWords name what an operator does, so `increment i` restates `i++`.
var operatorWords = map[string][]string{
	"++": {"increment"}, "--": {"decrement"}, "+=": {"add", "increment"}, "-=": {"subtract", "decrement"},
}

// wordsOf returns the lower-case words of a comment.
func wordsOf(text string) []string {
	words := names(text)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return words
}

// codeWords returns the stemmed names of a code line with their camelCase and snake_case parts, and its operators named.
func codeWords(code string) map[string]bool {
	words := operatorNames(code)
	for _, name := range names(code) {
		for _, part := range append(nameParts(name), name) {
			words[stem(strings.ToLower(part))] = true
		}
	}
	return words
}

// operatorNames returns the words that name each operator in code.
func operatorNames(code string) map[string]bool {
	words := map[string]bool{}
	for op, names := range operatorWords {
		for _, name := range names {
			words[name] = words[name] || strings.Contains(code, op)
		}
	}
	return words
}

// nameParts splits a name at each underscore and each lower-to-upper case change.
func nameParts(name string) []string {
	var parts []string
	for _, snake := range strings.Split(name, "_") {
		parts = append(parts, camelParts(snake)...)
	}
	return parts
}

// camelParts splits a name at each lower-to-upper case change.
func camelParts(name string) []string {
	var parts []string
	start := 0
	runes := []rune(name)
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) && unicode.IsLower(runes[i-1]) {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
	}
	return append(parts, string(runes[start:]))
}

// stem drops a plural or third-person `s`, so `returns` meets `return`.
func stem(w string) string {
	if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
		return strings.TrimSuffix(w, "s")
	}
	return w
}
