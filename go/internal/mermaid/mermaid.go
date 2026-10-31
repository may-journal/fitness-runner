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
//   - ExtractCallouts returns a diagram body's callouts in document order,
//     duplicates preserved, each marked as a node or a connector callout.
//     Nodes carry numbers; connectors carry letters (A–Z, then AA–ZZ, then
//     AAA…). ParseDoc sorts them into the DiagramBlock fields.
//   - IsCalloutHeader reports whether split header cells mark a numbered
//     callout table.
//   - PairDiagramsWithTables associates each diagram with callouts with the
//     callout table that follows it; legend diagrams (no callouts) are
//     invisible to pairing, and tables before any such diagram are orphans.
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
//
//fitness:container cli
//fitness:component kit Checks share it as one engine, so no check keeps its own copy.
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
	// Numbers are the node callout numbers referenced, in document order
	// (may contain duplicates).
	Numbers []int
	// Letters are the connector callout letters that follow the doubling,
	// in document order (may contain duplicates).
	Letters []string
	// EdgeNumbers are connector callouts written as numbers, which the
	// letter rule forbids, in document order.
	EdgeNumbers []int
	// BadLetters are whole connector labels of capital letters that break
	// the doubling, such as AB, in document order.
	BadLetters []string
}

// Kind reports DiagramKind.
func (*DiagramBlock) Kind() Kind { return DiagramKind }

// HasCallouts reports whether the diagram carries any node or connector
// callout. A diagram without one is a legend or a plain diagram. Labels
// that break the doubling do not count, since a label like OK is prose.
func (b *DiagramBlock) HasCallouts() bool {
	return len(b.Numbers)+len(b.Letters)+len(b.EdgeNumbers) > 0
}

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
	// Letters are the first-column connector letters that follow the
	// doubling, in order (may contain duplicates).
	Letters []string
	// BadLetters are first-column capital-letter cells that break the
	// doubling, such as AB, in order.
	BadLetters []string
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

// Callout is one callout label read from a diagram.
type Callout struct {
	// Label is the callout as written: digits such as "5", or capital
	// letters such as "A".
	Label string
	// Edge reports a connector callout; false means a node callout.
	Edge bool
	// Offset is the label's byte offset in the diagram body, so a fix can
	// rewrite exactly the label the check read.
	Offset int
}

// side says what a pattern match labels: nothing, a node, or a connector.
type side int

const (
	sideNone side = iota
	sideNode
	sideEdge
)

// calloutPattern is one JS callout regex split into its RE2-safe core and
// the (?=…) lookahead emulated as a follower predicate (nil when the
// original had none). The greedy \d+ and [A-Z]+ never backtrack usefully —
// every follower class excludes digits and letters — so the core's match
// extent is exactly the JS engine's. side classifies the match by where it
// starts, and whole, for letter patterns, reports a label that ends right
// after the letters.
type calloutPattern struct {
	core    *regexp.Regexp
	follows func(line string, end int) bool
	side    func(line string, start int) side
	whole   func(line string, end int) bool
}

// calloutPatterns locate a callout as the leading number or letters of a
// label. The label's absolute position dedupes matches that overlap across
// patterns. Numbers mark nodes; numbers on connectors are read too, so the
// check can reject them. Letters mark connectors only.
var calloutPatterns = []calloutPattern{
	{regexp.MustCompile(`["'](\d+)`), spaceOrQuoteFollows, quotedSide(sideNode), nil},                  // quoted label: "1 Check", "6"
	{regexp.MustCompile(`[([{>][` + jsWS + `]*(\d+)`), spaceFollows, always(sideNode), nil},            // node-shape label: [2 Screen], ((1 Persona)), [(4 System)]
	{regexp.MustCompile(`:[` + jsWS + `]*(\d+)`), spaceOrEndFollows, colonSide(sideNode), nil},         // edge/message label: A --> B : 11
	{regexp.MustCompile(`\|[` + jsWS + `]*(\d+)`), spaceOrPipeFollows, pipeSide, nil},                  // flowchart pipe label: -->|5|, -->|6 calls the api|
	{regexp.MustCompile(`--[` + jsWS + `]*(\d+)[` + jsWS + `]*--`), nil, inlineSide, nil},              // inline link label: A -- 5 --> B
	{regexp.MustCompile(`["']([A-Z]+)`), spaceOrQuoteFollows, quotedSide(sideNone), quoteAhead},        // quoted connector label: Rel(a, b, "A"), -->|"B"|
	{regexp.MustCompile(`:[` + jsWS + `]*([A-Z]+)`), spaceOrEndFollows, colonSide(sideNone), endAhead}, // message label: A ->> B : C
	{regexp.MustCompile(`\|[` + jsWS + `]*([A-Z]+)`), spaceOrPipeFollows, pipeSide, pipeAhead},         // flowchart pipe label: -->|D|
	{regexp.MustCompile(`--[` + jsWS + `]*([A-Z]+)[` + jsWS + `]*--`), nil, inlineSide, nil},           // inline link label: A -- E --> B
}

// relLine matches a C4 relationship call, whose quoted label is a
// connector's.
var relLine = regexp.MustCompile(`^[` + jsWS + `]*(?:Bi)?Rel\w*[` + jsWS + `]*\(`)

// arrow matches a link or message arrow: flowchart, sequence, class, and
// state forms.
var arrow = regexp.MustCompile(`--|->|==|\.\.|-\.|-\)`)

// always classifies every match the same way.
func always(s side) func(string, int) side {
	return func(string, int) side { return s }
}

// quotedSide classifies a quoted label: a connector's when it opens a pipe
// or inline link label or is a C4 Rel call's label, otherwise other.
func quotedSide(other side) func(string, int) side {
	return func(line string, start int) side {
		before := strings.TrimRightFunc(line[:start], isJSSpace)
		for _, opener := range []string{"|", "--", "-.", "=="} {
			if strings.HasSuffix(before, opener) {
				return sideEdge
			}
		}
		if relLine.MatchString(line) {
			return relSide(line, start)
		}
		return other
	}
}

// relSide classifies a quoted argument of a C4 Rel call. The first quoted
// argument is the connector's label; a later one, such as the technology
// in Rel(a, b, "A", "HTTPS"), labels nothing.
func relSide(line string, start int) side {
	if start == strings.IndexAny(line, `"'`) {
		return sideEdge
	}
	return sideNone
}

// arrowTail matches the end of a link arrow, such as -->, ---, ==>, -.->,
// --o, or --x, which a pipe label's opening pipe follows.
var arrowTail = regexp.MustCompile(`(?:[-=.]{2,}|[-=.]+[>ox])$`)

// pipeSide classifies a pipe label: a connector's when the pipe opens one
// right after a link arrow, as in A -->|D| B. A closing pipe starts no
// label, so in A -->|go| Y the node Y is no callout.
func pipeSide(line string, start int) side {
	if arrowTail.MatchString(strings.TrimRightFunc(line[:start], isJSSpace)) {
		return sideEdge
	}
	return sideNone
}

// colonSide classifies a colon label: a connector's when an arrow comes
// before it, as in A --> B : 5 or A ->> B: 5, otherwise other, as in a
// state description or a note.
func colonSide(other side) func(string, int) side {
	return func(line string, start int) side {
		if arrow.MatchString(line[:start]) {
			return sideEdge
		}
		return other
	}
}

// inlineSide classifies an inline link label. A match whose opening dashes
// continue a longer run, as in A --- B --- C, is a link, not a label.
func inlineSide(line string, start int) side {
	if start > 0 && line[start-1] == '-' {
		return sideNone
	}
	return sideEdge
}

// restAfterSpace returns line from end with leading JS whitespace trimmed.
func restAfterSpace(line string, end int) string {
	return trimStartJS(line[end:])
}

// quoteAhead reports a quote right after optional whitespace.
func quoteAhead(line string, end int) bool {
	rest := restAfterSpace(line, end)
	return strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, "'")
}

// endAhead reports nothing but whitespace to the end of the line.
func endAhead(line string, end int) bool {
	return restAfterSpace(line, end) == ""
}

// pipeAhead reports a pipe right after optional whitespace.
func pipeAhead(line string, end int) bool {
	return strings.HasPrefix(restAfterSpace(line, end), "|")
}

// IsLetterCallout reports whether label is a connector letter that follows
// the doubling: one capital letter repeated, as in A, ZZ, or AAA.
func IsLetterCallout(label string) bool {
	return isLetters(label) && strings.Count(label, label[:1]) == len(label)
}

// fileLetters appends a letter label to good when it follows the doubling,
// or to bad when it breaks it.
func fileLetters(label string, good, bad []string) ([]string, []string) {
	if IsLetterCallout(label) {
		return append(good, label), bad
	}
	return good, append(bad, label)
}

// isLetters reports a label of capital letters.
func isLetters(label string) bool {
	return label != "" && label[0] >= 'A' && label[0] <= 'Z'
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

// collectCalloutPositions records each callout's absolute label position for
// one (non-style) line by scanning it with every callout pattern.
func collectCalloutPositions(line string, offset int, byPosition map[int]Callout) {
	for _, p := range calloutPatterns {
		p.scan(line, offset, byPosition)
	}
}

// scan records the pattern's callouts on line into byPosition, keyed by each
// label's absolute position (line offset + in-line index). Matching mirrors
// JS matchAll: a successful match resumes after the consumed core (the
// lookahead consumes nothing); a rejected match abandons that start and
// scans again from the next character, exactly where the JS engine's scan
// would resume.
func (p calloutPattern) scan(line string, offset int, byPosition map[int]Callout) {
	pos := 0
	for {
		loc := p.core.FindStringSubmatchIndex(line[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		label := line[pos+loc[2] : pos+loc[3]]
		s := p.side(line, start)
		if s == sideNone || !p.accepts(line, label, end) {
			pos = start + 1
			continue
		}
		at := offset + pos + loc[2]
		byPosition[at] = Callout{Label: label, Edge: s == sideEdge, Offset: at}
		pos = end
	}
}

// accepts reports whether a core match ending at byte offset end passes the
// pattern's emulated lookahead. Letters that break the doubling pass only as
// a whole label, so prose such as `|API calls|` is not read as a callout.
func (p calloutPattern) accepts(line, label string, end int) bool {
	if p.follows != nil && !p.follows(line, end) {
		return false
	}
	return !p.needsWhole(label) || p.whole(line, end)
}

// needsWhole reports a letter pattern's label that breaks the doubling,
// which counts only as a whole label.
func (p calloutPattern) needsWhole(label string) bool {
	return p.whole != nil && isLetters(label) && !IsLetterCallout(label)
}

// ExtractCallouts returns the callouts referenced in a mermaid diagram body.
// Heuristic: the leading number or letters of each node/edge label — quoted
// (`"1 Check"`, `Rel(a, b, "A")`), inside a node shape (`[2 Screen]`,
// `((1 …))`), or as an edge label (`: B`, `|C|`, `-- D --`). Each callout is
// marked as a node's or a connector's. Styling lines are skipped so hex
// colors and stroke widths aren't mistaken for callouts. Callouts come back
// in document order, preserving duplicates at distinct positions, each with
// its byte offset in body.
func ExtractCallouts(body string) []Callout {
	byPosition := map[int]Callout{}
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
	callouts := make([]Callout, len(positions))
	for i, p := range positions {
		callouts[i] = byPosition[p]
	}
	return callouts
}

// sort files each callout into the block's node numbers, connector letters,
// connector numbers, or letters that break the doubling.
func (b *DiagramBlock) sort(callouts []Callout) {
	for _, c := range callouts {
		n, err := strconv.Atoi(c.Label)
		switch {
		case err != nil:
			b.Letters, b.BadLetters = fileLetters(c.Label, b.Letters, b.BadLetters)
		case c.Edge:
			b.EdgeNumbers = append(b.EdgeNumbers, n)
		default:
			b.Numbers = append(b.Numbers, n)
		}
	}
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

// firstCellLabel matches a callout first-column cell: a number or capital
// letters, e.g. "1", "[1]", "A", or "[AA]".
var firstCellLabel = regexp.MustCompile(`^\[?(\d+|[A-Z]+)\]?$`)

// RowLabel returns a callout row's first-column label, brackets stripped,
// or "" when the cell holds no number or capital letters.
func RowLabel(row []string) string {
	if len(row) == 0 {
		return ""
	}
	if m := firstCellLabel.FindStringSubmatch(row[0]); m != nil {
		return m[1]
	}
	return ""
}

// RowLabelOffset returns the byte offset in a raw table row line where its
// first-column label starts, past the outer pipe, the cell's padding, and
// an opening bracket. The label RowLabel reads from the split row starts
// there.
func RowLabelOffset(line string) int {
	rest := strings.TrimPrefix(trimStartJS(line), "|")
	rest = strings.TrimPrefix(trimStartJS(rest), "[")
	return len(line) - len(rest)
}

// readRows files each row's first-column label into the table's numbers,
// connector letters, or letters that break the doubling.
func (t *CalloutTableBlock) readRows() {
	t.Numbers = []int{}
	for _, row := range t.Rows {
		label := RowLabel(row)
		if n, err := strconv.Atoi(label); err == nil {
			t.Numbers = append(t.Numbers, n)
		} else if label != "" {
			t.Letters, t.BadLetters = fileLetters(label, t.Letters, t.BadLetters)
		}
	}
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
	block := &DiagramBlock{Body: joined, Line: start + 1}
	block.sort(ExtractCallouts(joined))
	return block, i + 1 // skip the closing fence
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
	table := &CalloutTableBlock{Header: header, Line: start + 1, Rows: rows}
	table.readRows()
	return table, j
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
// (those with no callouts) are exempt: they carry no callouts, so
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
// new pending diagram: a legend diagram (no callouts) is invisible to
// pairing and leaves pending unchanged; a diagram with callouts flushes any
// pending diagram as a table-less pair and becomes pending itself.
func pairDiagram(result *PairResult, pending, b *DiagramBlock) *DiagramBlock {
	if !b.HasCallouts() {
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

// RunDocCheck runs a per-document validator over every in-scope .md file
// under root that contains at least one mermaid diagram or callout table; files with
// neither are skipped and don't count toward filesChecked. Shared by the
// mermaid diagram + callout table checks so each supplies only its own
// validate rule. A file read error aborts the check (the caller crashes,
// matching the TS driver's thrown readFileSync error).
func RunDocCheck(root string, validate func(file, content string) []string) (errors []string, filesChecked int, err error) {
	for _, file := range walkfs.InScope(walkfs.FilesByExt(root, ".md")) {
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

// SplitArgs splits the arguments of a C4 macro call on commas outside
// quotes, each trimmed, as in `Person(user, "1 User, admin")`.
func SplitArgs(text string) []string {
	var parts []string
	quoted, start := false, 0
	for i, r := range text {
		if r == '"' {
			quoted = !quoted
		}
		if r == ',' && !quoted {
			parts, start = append(parts, strings.TrimSpace(text[start:i])), i+1
		}
	}
	return append(parts, strings.TrimSpace(text[start:]))
}
