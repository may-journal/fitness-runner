// Package mermaid parses the mermaid diagrams and numbered callout tables
// out of a markdown document — the Go port of the TypeScript shared
// mermaid.ts plus its runMermaidDocCheck.ts driver. The six mermaid checks
// (mermaid-callouts, mermaid-callout-why, mermaid-diagram-prose,
// mermaid-diagram-table-gap, mermaid-legend, mermaid-level-bleed) build on
// this API:
//
//   - ParseDoc returns the document's blocks in order: *DiagramBlock (a
//     ```mermaid fenced block) and *CalloutTableBlock (a GFM table whose
//     first header cell is #/No/Callout/Ref). Discriminate with a type
//     switch or the Kind method (DiagramKind / TableKind).
//   - ExtractDiagramNumbers returns a diagram body's callout numbers in
//     document order, duplicates preserved; ParseDoc fills
//     DiagramBlock.Numbers with it.
//   - IsCalloutHeader reports whether split header cells mark a numbered
//     callout table.
//   - PairDiagramsWithTables associates each numbered diagram with the
//     callout table that follows it; legend diagrams (no numbers) are
//     invisible to pairing, and tables before any numbered diagram are
//     orphans.
//   - RunDocCheck walks every .md under root and runs a validator over each
//     file with at least one parsed block; block-free files are skipped and
//     do not count toward filesChecked.
//
// The TypeScript callout patterns use lookahead assertions, which RE2 lacks;
// each
// (?=…) is emulated by matching the core pattern and testing the following
// rune by hand, preserving JS matchAll resumption (a lookahead consumes
// nothing; a failed lookahead retries at the next character). Positions
// are byte offsets where the original used UTF-16 code units — the two are
// ordering-consistent (both strictly increase along the text), so
// dedupe-by-position and document ordering behave identically.
package mermaid

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Kind discriminates the two DocBlock implementations — the port of the
// TypeScript DIAGRAM_KIND/TABLE_KIND literals.
type Kind string

const (
	// DiagramKind marks a *DiagramBlock.
	DiagramKind Kind = "diagram"
	// TableKind marks a *CalloutTableBlock.
	TableKind Kind = "table"
)

// DocBlock is one parsed block: a *DiagramBlock or a *CalloutTableBlock.
// Type-switch on the concrete types for their fields.
type DocBlock interface {
	// Kind reports the block's discriminant.
	Kind() Kind
}

// DiagramBlock is a ```mermaid fenced block.
type DiagramBlock struct {
	// Body is the raw mermaid source, fences stripped.
	Body string
	// Line is the 1-based line of the opening fence.
	Line int
	// Numbers are the callout numbers referenced, in document order (may
	// contain duplicates).
	Numbers []int
}

// Kind reports DiagramKind.
func (*DiagramBlock) Kind() Kind { return DiagramKind }

// CalloutTableBlock is a GFM table whose first header cell marks it as a
// numbered callout table.
type CalloutTableBlock struct {
	// Header holds the trimmed header cells.
	Header []string
	// Line is the 1-based line of the header row.
	Line int
	// Numbers are the integers read from the first (`#`) column, in order
	// (may contain duplicates).
	Numbers []int
	// Rows holds the trimmed body-row cells.
	Rows [][]string
}

// Kind reports TableKind.
func (*CalloutTableBlock) Kind() Kind { return TableKind }

// jsWS is the JavaScript \s character class (ASCII whitespace incl. \v,
// Unicode spaces, line separators, and the BOM) as an RE2 class body, so the
// ported patterns match exactly what the JS regexes matched.
const jsWS = `\t\n\v\f\r \x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`

// isJSSpace reports whether r is in the JavaScript \s class — the rune-level
// twin of jsWS, used by the lookahead emulation and trim helpers.
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// trimStartJS trims leading JavaScript whitespace — String.prototype.trimStart.
func trimStartJS(s string) string { return strings.TrimLeftFunc(s, isJSSpace) }

// trimJS trims JavaScript whitespace from both ends — String.prototype.trim.
func trimJS(s string) string { return strings.TrimFunc(s, isJSSpace) }

// styleLine matches mermaid directive/style lines whose numbers (hex colors,
// stroke widths) are not callouts.
var styleLine = regexp.MustCompile(`^[` + jsWS + `]*(classDef|style|linkStyle|%%)`)

// calloutPattern is one JS callout regex split into its RE2-safe core and
// the (?=…) lookahead emulated as a follower predicate (nil when the
// original had none). The greedy \d+ never backtracks usefully — every
// follower class excludes digits — so the core's match extent is exactly the
// JS engine's.
type calloutPattern struct {
	core    *regexp.Regexp
	follows func(line string, end int) bool
}

// calloutPatterns locate a callout number as the leading integer of a label.
// The digit's absolute position dedupes matches that overlap across patterns.
var calloutPatterns = []calloutPattern{
	{regexp.MustCompile(`["'](\d+)`), spaceOrQuoteFollows},               // quoted label: "1 Check", "6"
	{regexp.MustCompile(`[([{>][` + jsWS + `]*(\d+)`), spaceFollows},     // node-shape label: [2 Screen], ((1 Persona)), [(4 System)]
	{regexp.MustCompile(`:[` + jsWS + `]*(\d+)`), spaceOrEndFollows},     // edge/message label: A --> B : 11
	{regexp.MustCompile(`\|[` + jsWS + `]*(\d+)`), spaceOrPipeFollows},   // flowchart pipe label: -->|5|, -->|6 calls the api|
	{regexp.MustCompile(`--[` + jsWS + `]*(\d+)[` + jsWS + `]*--`), nil}, // inline link label: A -- 5 --> B
}

// followRune returns the rune immediately after byte offset end — the
// subject of a lookahead — or ok=false at end of line.
func followRune(line string, end int) (rune, bool) {
	if end >= len(line) {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(line[end:])
	return r, true
}

// spaceOrQuoteFollows emulates (?=[\s"']).
func spaceOrQuoteFollows(line string, end int) bool {
	r, ok := followRune(line, end)
	return ok && (isJSSpace(r) || r == '"' || r == '\'')
}

// spaceFollows emulates (?=\s).
func spaceFollows(line string, end int) bool {
	r, ok := followRune(line, end)
	return ok && isJSSpace(r)
}

// spaceOrEndFollows emulates (?=\s|$).
func spaceOrEndFollows(line string, end int) bool {
	r, ok := followRune(line, end)
	return !ok || isJSSpace(r)
}

// spaceOrPipeFollows emulates (?=[\s|]).
func spaceOrPipeFollows(line string, end int) bool {
	r, ok := followRune(line, end)
	return ok && (isJSSpace(r) || r == '|')
}

// collectCalloutPositions records each callout number's absolute digit
// position for one (non-style) line by scanning it with every callout
// pattern.
func collectCalloutPositions(line string, offset int, byPosition map[int]int) {
	for _, p := range calloutPatterns {
		p.scan(line, offset, byPosition)
	}
}

// scan records the pattern's callout numbers on line into byPosition, keyed
// by each digit's absolute position (line offset + in-line index). Matching
// mirrors JS matchAll: a successful match resumes after the consumed core
// (the lookahead consumes nothing); a failed lookahead abandons that start
// and scans again from the next character, exactly where the JS engine's
// scan would resume.
func (p calloutPattern) scan(line string, offset int, byPosition map[int]int) {
	pos := 0
	for {
		loc := p.core.FindStringSubmatchIndex(line[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		if p.lookaheadFails(line, end) {
			pos = start + 1
			continue
		}
		if n, err := strconv.Atoi(line[pos+loc[2] : pos+loc[3]]); err == nil {
			byPosition[offset+pos+loc[2]] = n
		}
		pos = end
	}
}

// lookaheadFails reports whether the pattern's emulated lookahead rejects a
// core match ending at byte offset end; patterns without a lookahead never
// reject.
func (p calloutPattern) lookaheadFails(line string, end int) bool {
	return p.follows != nil && !p.follows(line, end)
}

// ExtractDiagramNumbers returns the callout numbers referenced in a mermaid
// diagram body. Heuristic: the leading integer of each node/edge label —
// quoted (`"1 Check"`), inside a node shape (`[2 Screen]`, `((1 …))`), or as
// an edge label (`: 11`, `|5|`). Styling lines are skipped so hex colors and
// stroke widths aren't mistaken for callouts. Numbers come back in document
// order, preserving duplicates at distinct positions.
func ExtractDiagramNumbers(body string) []int {
	byPosition := map[int]int{}
	offset := 0
	for _, line := range strings.Split(body, "\n") {
		if !styleLine.MatchString(line) {
			collectCalloutPositions(line, offset, byPosition)
		}
		offset += len(line) + 1 // +1 for the '\n' removed by Split
	}
	positions := make([]int, 0, len(byPosition))
	for p := range byPosition {
		positions = append(positions, p)
	}
	sort.Ints(positions)
	numbers := make([]int, len(positions))
	for i, p := range positions {
		numbers[i] = byPosition[p]
	}
	return numbers
}

// tableSeparator matches a GFM table separator row (`| --- | :--: |`).
var tableSeparator = regexp.MustCompile(
	`^[` + jsWS + `]*\|?[` + jsWS + `]*:?-+:?[` + jsWS + `]*(\|[` + jsWS + `]*:?-+:?[` + jsWS + `]*)*\|?[` + jsWS + `]*$`)

// splitRow splits a GFM table row into trimmed cell strings, dropping the
// outer pipes.
func splitRow(line string) []string {
	inner := trimJS(line)
	inner = strings.TrimPrefix(inner, "|")
	inner = strings.TrimSuffix(inner, "|")
	cells := strings.Split(inner, "|")
	for i, cell := range cells {
		cells[i] = trimJS(cell)
	}
	return cells
}

// calloutHeaderCell matches a first header cell that marks a callout table.
var calloutHeaderCell = regexp.MustCompile(`(?i)^(#|no\.?|callout|ref)$`)

// IsCalloutHeader reports whether a table's first header cell marks it as a
// numbered callout table.
func IsCalloutHeader(cells []string) bool {
	return len(cells) > 0 && calloutHeaderCell.MatchString(cells[0])
}

// firstCellNumber matches a numbered first-column cell, e.g. "1" or "[1]".
var firstCellNumber = regexp.MustCompile(`^\[?(\d+)\]?$`)

// tableNumbers reads the leading integer from a callout table's first-column
// cells.
func tableNumbers(rows [][]string) []int {
	numbers := []int{}
	for _, row := range rows {
		first := ""
		if len(row) > 0 {
			first = row[0]
		}
		if m := firstCellNumber.FindStringSubmatch(first); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				numbers = append(numbers, n)
			}
		}
	}
	return numbers
}

// fenceOpen matches a mermaid fence opener: three-plus backticks or tildes
// then the word "mermaid", case-insensitively.
var fenceOpen = regexp.MustCompile(`(?i)^[` + jsWS + "]*(`{3,}|~{3,})[" + jsWS + `]*mermaid\b`)

// parseDiagramBlock parses a mermaid fence opened at start and returns the
// block plus the index after its closing fence (a line whose trimmed start
// begins with the same marker).
func parseDiagramBlock(lines []string, start int, marker string) (*DiagramBlock, int) {
	var body []string
	i := start + 1
	for i < len(lines) && !strings.HasPrefix(trimStartJS(lines[i]), marker) {
		body = append(body, lines[i])
		i++
	}
	joined := strings.Join(body, "\n")
	return &DiagramBlock{
		Body:    joined,
		Line:    start + 1,
		Numbers: ExtractDiagramNumbers(joined),
	}, i + 1 // skip the closing fence
}

// parseTableBlock reads a GFM table at start (header at start, separator at
// start+1) and returns the block — nil when not a callout table — plus the
// index after the table's rows.
func parseTableBlock(lines []string, start int) (*CalloutTableBlock, int) {
	header := splitRow(lines[start])
	var rows [][]string
	j := start + 2
	for j < len(lines) && strings.Contains(lines[j], "|") && trimJS(lines[j]) != "" {
		rows = append(rows, splitRow(lines[j]))
		j++
	}
	if !IsCalloutHeader(header) {
		return nil, j
	}
	return &CalloutTableBlock{
		Header:  header,
		Line:    start + 1,
		Numbers: tableNumbers(rows),
		Rows:    rows,
	}, j
}

// isTableStart reports whether lines[i] opens a GFM table (a row followed by
// a separator row).
func isTableStart(lines []string, i int) bool {
	return strings.Contains(lines[i], "|") && i+1 < len(lines) && tableSeparator.MatchString(lines[i+1])
}

// ParseDoc parses a markdown document into ordered mermaid diagrams and
// numbered callout tables. Non-callout tables and prose are ignored.
func ParseDoc(content string) []DocBlock {
	lines := strings.Split(content, "\n")
	var blocks []DocBlock
	i := 0
	for i < len(lines) {
		if m := fenceOpen.FindStringSubmatch(lines[i]); m != nil {
			block, next := parseDiagramBlock(lines, i, m[1])
			blocks = append(blocks, block)
			i = next
			continue
		}
		if isTableStart(lines, i) {
			block, next := parseTableBlock(lines, i)
			if block != nil {
				blocks = append(blocks, block)
			}
			i = next
			continue
		}
		i++
	}
	return blocks
}

// DiagramTablePair is a mermaid diagram and the callout table that
// immediately follows it, if any.
type DiagramTablePair struct {
	Diagram *DiagramBlock
	// Table is nil when no callout table follows the diagram.
	Table *CalloutTableBlock
}

// PairResult is PairDiagramsWithTables' outcome.
type PairResult struct {
	// OrphanTables are callout tables with no preceding numbered diagram.
	OrphanTables []*CalloutTableBlock
	// Pairs holds each numbered diagram with its following table, in order.
	Pairs []DiagramTablePair
}

// PairDiagramsWithTables associates each numbered diagram with the callout
// table that follows it before the next numbered diagram. Legend diagrams
// (those with no callout numbers) are exempt: they carry no callouts, so
// they neither require a table nor consume the table that belongs to a
// preceding numbered diagram — the documented layout is diagram, legend,
// then table. Tables appearing before any diagram are orphans.
func PairDiagramsWithTables(blocks []DocBlock) PairResult {
	result := PairResult{OrphanTables: []*CalloutTableBlock{}, Pairs: []DiagramTablePair{}}
	var pending *DiagramBlock
	for _, block := range blocks {
		switch b := block.(type) {
		case *DiagramBlock:
			pending = pairDiagram(&result, pending, b)
		case *CalloutTableBlock:
			pending = pairTable(&result, pending, b)
		}
	}
	if pending != nil {
		result.Pairs = append(result.Pairs, DiagramTablePair{Diagram: pending})
	}
	return result
}

// pairDiagram folds a diagram block into the pairing state and returns the
// new pending diagram: a legend diagram (no callout numbers) is invisible to
// pairing and leaves pending unchanged; a numbered diagram flushes any
// pending diagram as a table-less pair and becomes pending itself.
func pairDiagram(result *PairResult, pending, b *DiagramBlock) *DiagramBlock {
	if len(b.Numbers) == 0 {
		return pending
	}
	if pending != nil {
		result.Pairs = append(result.Pairs, DiagramTablePair{Diagram: pending})
	}
	return b
}

// pairTable folds a callout table into the pairing state and returns the new
// pending diagram: the table completes the pending diagram's pair (clearing
// pending), or becomes an orphan when no numbered diagram precedes it.
func pairTable(result *PairResult, pending *DiagramBlock, b *CalloutTableBlock) *DiagramBlock {
	if pending == nil {
		result.OrphanTables = append(result.OrphanTables, b)
		return nil
	}
	result.Pairs = append(result.Pairs, DiagramTablePair{Diagram: pending, Table: b})
	return nil
}

// RunDocCheck runs a per-document validator over every .md file under root
// that contains at least one mermaid diagram or callout table; files with
// neither are skipped and don't count toward filesChecked. Shared by the
// mermaid diagram + callout table checks so each supplies only its own
// validate rule. A file read error aborts the check (the caller crashes,
// matching the TS driver's thrown readFileSync error).
func RunDocCheck(root string, validate func(file, content string) []string) (errors []string, filesChecked int, err error) {
	for _, file := range walkfs.FilesByExt(root, ".md") {
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if readErr != nil {
			return nil, 0, readErr
		}
		content := string(raw)
		if len(ParseDoc(content)) == 0 {
			continue
		}
		filesChecked++
		errors = append(errors, validate(file, content)...)
	}
	return errors, filesChecked, nil
}
