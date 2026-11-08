package check

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// This file reads the block-style YAML of workflow files, keeping the line each value starts on, and keeps flow values as text.

// kind is what a node holds.
type kind int

const (
	scalarNode kind = iota
	mapNode
	listNode
	flowNode
)

// node is one YAML value and the line it starts on.
type node struct {
	kind kind
	line int
	// text is a scalar's or a flow value's text.
	text string
	// entries are a mapping's keys and values, in file order.
	entries []entry
	// items are a list's values, in file order.
	items []*node
}

// entry is one key of a mapping, its line, and its value.
type entry struct {
	key  string
	line int
	val  *node
}

// get is the value under key in a mapping, or nil.
func (n *node) get(key string) *node {
	if n == nil {
		return nil
	}
	for _, e := range n.entries {
		if e.key == key {
			return e.val
		}
	}
	return nil
}

// syntaxError is a file YAML cannot read, and the line and column where
// reading stopped. col is 0 when the whole line is at fault.
type syntaxError struct {
	line int
	col  int
	msg  string
}

func (e *syntaxError) Error() string { return fmt.Sprintf("line %d: %s", e.line, e.msg) }

// srcLine is one line of the file: its number, its indent in spaces, and
// the text after that indent.
type srcLine struct {
	num    int
	indent int
	text   string
}

// parser walks the lines of one file.
type parser struct {
	lines   []srcLine
	pos     int
	anchors map[string]*node
	// err is a fault peek found, such as a tab indent, which ends the read.
	err *syntaxError
}

var (
	// blockHeaderRe matches a block scalar's header, such as `|` or `>-`.
	blockHeaderRe = regexp.MustCompile(`^[|>][-+0-9]*$`)
	// keyRe matches a plain key and its colon, which a space or the line's
	// end follows.
	keyRe = regexp.MustCompile("^([^\\s\\[\\]{}#&*!|>%@`\"'](?:[^:]|:\\S)*?)\\s*:(?:\\s|$)")
	// doubleEscapeRe matches a backslash escape in a double-quoted string.
	doubleEscapeRe = regexp.MustCompile(`\\.`)
	// singleEscapeRe matches a doubled quote in a single-quoted string.
	singleEscapeRe = regexp.MustCompile(`''`)
	// docMarkRe matches a line that starts or ends a YAML document.
	docMarkRe = regexp.MustCompile(`^(---|\.\.\.)(\s.*)?$`)
	// mapColonRe matches a colon that a blank or the end follows, which
	// starts a mapping value and so ends a plain scalar.
	mapColonRe = regexp.MustCompile(`:(\s|$)`)
	// keyLineIndicatorRe matches a value that opens a list, a key or a
	// mapping value, which YAML forbids on its key's line.
	keyLineIndicatorRe = regexp.MustCompile(`^[-?:](\s|$)`)
)

// reservedStarts are characters that cannot start a plain scalar: flow
// collections own `,]}`, directives own `%`, and YAML reserves `@` and the backtick.
const reservedStarts = ",]}%@`"

// parseYAML reads one YAML document. An error is a *syntaxError naming the
// line.
func parseYAML(content string) (*node, error) {
	p := &parser{anchors: map[string]*node{}}
	for i, raw := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		text := strings.TrimLeft(raw, " ")
		p.lines = append(p.lines, srcLine{num: i + 1, indent: len(raw) - len(text), text: text})
	}
	root, err := p.document()
	if p.err != nil {
		return nil, p.err
	}
	return root, err
}

// document reads the root node and fails any line left after it.
func (p *parser) document() (*node, error) {
	p.skipMarks()
	first := p.peek()
	if first == nil {
		return &node{kind: mapNode, line: 1}, nil
	}
	root, err := p.collection(first)
	if err != nil {
		return nil, err
	}
	p.skipMarks()
	if l := p.peek(); l != nil {
		return nil, fault(l, "a line here is outside the document's indentation")
	}
	return root, nil
}

// skipMarks moves past `---` and `...` lines at the top level.
func (p *parser) skipMarks() {
	for l := p.peek(); l != nil && l.indent == 0 && docMarkRe.MatchString(l.text); l = p.peek() {
		p.pos++
	}
}

// fault is a syntax error at l.
func fault(l *srcLine, msg string) error {
	return &syntaxError{line: l.num, msg: msg}
}

// faultAt is a syntax error at column col of l.
func faultAt(l *srcLine, col int, msg string) error {
	return &syntaxError{line: l.num, col: col, msg: msg}
}

// blanks are the characters YAML trims around a value: spaces and tabs.
const blanks = " \t"

// colOf is the 1-based column, in characters, where text starts on l.
// text is a trimmed piece of l's text, so its last match is the piece.
func colOf(l *srcLine, text string) int {
	i := strings.LastIndex(l.text, text)
	if i < 0 {
		i = 0
	}
	return l.indent + utf8.RuneCountInString(l.text[:i]) + 1
}

// peek is the next line that holds YAML, past blank and comment lines, or
// nil at the end. A line indented with a tab ends the read with a fault.
func (p *parser) peek() *srcLine {
	for ; p.pos < len(p.lines); p.pos++ {
		if l := &p.lines[p.pos]; holdsYAML(l.text) {
			return p.spaced(l)
		}
	}
	return nil
}

// holdsYAML reports whether a line holds more than blanks or a comment. The
// code view keeps directives such as `# cspell:ignore`, so a # still reads.
func holdsYAML(text string) bool {
	t := strings.TrimSpace(text)
	return t != "" && !strings.HasPrefix(t, "#")
}

// spaced is l when spaces indent it. A tab indent records a fault and
// ends the read.
func (p *parser) spaced(l *srcLine) *srcLine {
	if !strings.HasPrefix(l.text, "\t") {
		return l
	}
	if p.err == nil {
		p.err = &syntaxError{line: l.num, msg: "a tab indents this line; YAML indents with spaces"}
	}
	return nil
}

// isItem reports whether text starts a list item.
func isItem(text string) bool {
	return text == "-" || strings.HasPrefix(text, "- ")
}

// collection reads the mapping or list that starts at l.
func (p *parser) collection(l *srcLine) (*node, error) {
	if isItem(l.text) {
		return p.list(l.indent)
	}
	return p.mapping(l.indent)
}

// nested reads a value on the lines after a key or dash at indent: a same-indent list, a deeper collection, or empty.
func (p *parser) nested(owner *srcLine, keyed bool) (*node, error) {
	if l := p.peek(); opens(l, owner, keyed) {
		return p.collection(l)
	}
	return &node{kind: scalarNode, line: owner.num}, nil
}

// opens reports whether l starts the value of owner's key or dash.
func opens(l, owner *srcLine, keyed bool) bool {
	if l == nil {
		return false
	}
	return l.indent > owner.indent || keyed && l.indent == owner.indent && isItem(l.text)
}

// mapping reads the keys written at indent.
func (p *parser) mapping(indent int) (*node, error) {
	n := &node{kind: mapNode, line: p.peek().num}
	for l := p.peek(); l != nil && l.indent >= indent; l = p.peek() {
		e, err := p.mapEntry(l, indent, n)
		if err != nil {
			return nil, err
		}
		n.entries = append(n.entries, e)
	}
	return n, nil
}

// mapEntry reads one `key: value` line at indent and the value it opens.
func (p *parser) mapEntry(l *srcLine, indent int, m *node) (entry, error) {
	if l.indent > indent {
		return entry{}, fault(l, "this line is indented deeper than the key above it")
	}
	key, rest, ok := splitKey(l.text)
	if !ok {
		return entry{}, fault(l, "expected `key: value` here")
	}
	if m.get(key) != nil {
		return entry{}, fault(l, fmt.Sprintf("the key %q appears twice in one mapping", key))
	}
	p.pos++
	val, err := p.value(l, rest, true)
	return entry{key: key, line: l.num, val: val}, err
}

// list reads the items written at indent.
func (p *parser) list(indent int) (*node, error) {
	n := &node{kind: listNode, line: p.peek().num}
	for l := p.peek(); inList(l, indent); l = p.peek() {
		item, err := p.listItem(l, indent)
		if err != nil {
			return nil, err
		}
		n.items = append(n.items, item)
	}
	return n, nil
}

// inList reports whether l belongs to the list at indent: an item at that
// indent, or a stray deeper line listItem faults.
func inList(l *srcLine, indent int) bool {
	return l != nil && (l.indent > indent || l.indent == indent && isItem(l.text))
}

// listItem reads the item at l, which must sit at the list's indent.
func (p *parser) listItem(l *srcLine, indent int) (*node, error) {
	if l.indent > indent {
		return nil, fault(l, "this line is indented deeper than the list item above it")
	}
	return p.item(l)
}

// item reads one list item, moving the indent past the dash when it opens a mapping or list on the dash's line.
func (p *parser) item(l *srcLine) (*node, error) {
	rest := strings.TrimLeft(strings.TrimPrefix(l.text, "-"), " ")
	if _, _, keyed := splitKey(rest); keyed || isItem(rest) {
		l.indent += len(l.text) - len(rest)
		l.text = rest
		return p.collection(l)
	}
	p.pos++
	return p.value(l, rest, false)
}

// splitKey splits `key: rest` at the first colon followed by a space or
// the line's end, reading a quoted key whole.
func splitKey(text string) (key, rest string, ok bool) {
	if strings.IndexAny(text, `"'`) == 0 {
		return quotedKey(text)
	}
	m := keyRe.FindStringSubmatch(text)
	if m == nil || notKey(text, m[1]) {
		return "", "", false
	}
	return strings.Trim(m[1], blanks), strings.Trim(text[len(m[0]):], blanks), true
}

// notKey reports whether a match of keyRe is a list item or a comment
// rather than a key.
func notKey(text, key string) bool {
	return isItem(text) || strings.Contains(key, " #")
}

// quotedKey reads a key written in quotes.
func quotedKey(text string) (key, rest string, ok bool) {
	end := closingQuote(text)
	if end < 0 {
		return "", "", false
	}
	after := strings.TrimLeft(text[end+1:], " ")
	if !strings.HasPrefix(after, ":") || len(after) > 1 && after[1] != ' ' {
		return "", "", false
	}
	return text[1:end], strings.Trim(after[1:], blanks), true
}

// closingQuote is the index of the quote that closes the one text starts
// with, or -1. A double-quoted string escapes with a backslash, a
// single-quoted one by doubling the quote.
func closingQuote(text string) int {
	q, escape := text[0], singleEscapeRe
	if q == '"' {
		escape = doubleEscapeRe
	}
	blanked := escape.ReplaceAllString(text[1:], "\x00\x00")
	if i := strings.IndexByte(blanked, q); i >= 0 {
		return i + 1
	}
	return -1
}

// scalarReaders reads a value by its first character.
var scalarReaders = map[byte]func(p *parser, owner *srcLine, rest string) (*node, error){
	'|':  (*parser).blockScalar,
	'>':  (*parser).blockScalar,
	'"':  (*parser).quoted,
	'\'': (*parser).quoted,
	'[':  (*parser).flow,
	'{':  (*parser).flow,
	'*':  (*parser).alias,
}

// value reads the value after a key or a dash on owner's line. keyed tells
// a key's value, which may be a list at the key's own indent.
func (p *parser) value(owner *srcLine, rest string, keyed bool) (*node, error) {
	rest = strings.Trim(rest, blanks)
	if rest == "" || strings.HasPrefix(rest, "#") {
		return p.nested(owner, keyed)
	}
	if rest[0] == '&' {
		return p.anchored(owner, rest, keyed)
	}
	if read, ok := scalarReaders[rest[0]]; ok {
		return read(p, owner, rest)
	}
	return p.plain(owner, rest, keyed)
}

// anchored reads `&name value` and remembers the value under name.
func (p *parser) anchored(owner *srcLine, rest string, keyed bool) (*node, error) {
	name, val, _ := strings.Cut(rest[1:], " ")
	n, err := p.value(owner, val, keyed)
	p.anchors[name] = n
	return n, err
}

// alias reads `*name`, the value an anchor named earlier.
func (p *parser) alias(owner *srcLine, rest string) (*node, error) {
	name, _, _ := strings.Cut(rest[1:], " ")
	if n, ok := p.anchors[name]; ok {
		return n, nil
	}
	return nil, fault(owner, fmt.Sprintf("the alias *%s names no anchor above it", name))
}

// plain reads a plain scalar and the deeper lines that continue it,
// failing one GitHub's YAML parser rejects. keyed tells a key's value.
func (p *parser) plain(owner *srcLine, rest string, keyed bool) (*node, error) {
	if err := plainStart(owner, rest, keyed); err != nil {
		return nil, err
	}
	first := cutComment(rest)
	if err := plainColon(owner, first); err != nil {
		return nil, err
	}
	parts, err := p.plainLines(owner)
	text := strings.Join(append([]string{first}, parts...), " ")
	return &node{kind: scalarNode, line: owner.num, text: text}, err
}

// plainLines reads the lines deeper than owner that continue its plain
// scalar.
func (p *parser) plainLines(owner *srcLine) ([]string, error) {
	var parts []string
	for l := p.peek(); l != nil && l.indent > owner.indent; l = p.peek() {
		part := cutComment(l.text)
		if err := plainColon(l, part); err != nil {
			return nil, err
		}
		parts = append(parts, part)
		p.pos++
	}
	return parts, nil
}

// openers name what `- `, `? ` and `: ` open, which YAML forbids on a
// key's line.
var openers = map[byte]string{'-': "list item", '?': "key", ':': "mapping value"}

// plainStart fails a plain scalar text on l that starts with a character
// YAML reserves. On a key's line, it also fails one that opens a list
// item, key or mapping value.
func plainStart(l *srcLine, text string, keyed bool) error {
	if strings.IndexByte(reservedStarts, text[0]) >= 0 {
		return faultAt(l, colOf(l, text), fmt.Sprintf("a plain value cannot start with %q, which YAML reserves; quote the value", text[0]))
	}
	if keyed && keyLineIndicatorRe.MatchString(text) {
		return faultAt(l, colOf(l, text), fmt.Sprintf("%q opens a %s on its key's line, which YAML does not allow; "+
			"move it onto its own line, indented under the key", text[0], openers[text[0]]))
	}
	return nil
}

// plainColon fails plain scalar text on l that holds a colon a blank or
// the end follows. YAML reads that colon as a mapping value, which a plain
// value cannot hold.
func plainColon(l *srcLine, text string) error {
	loc := mapColonRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	col := colOf(l, text) + utf8.RuneCountInString(text[:loc[0]])
	if loc[1] == len(text) && loc[1]-loc[0] == 1 {
		return faultAt(l, col, "a plain value ends in `:`, which YAML reads as a mapping it does not allow here; quote the value")
	}
	return faultAt(l, col, "a `: ` inside a plain value starts a mapping, which YAML does not allow here; quote the value")
}

// cutComment drops a ` #` comment from a plain scalar.
func cutComment(text string) string {
	if i := strings.Index(text, " #"); i >= 0 {
		text = text[:i]
	}
	return strings.Trim(text, blanks)
}

// quoted reads a quoted scalar, which may run over several lines.
func (p *parser) quoted(owner *srcLine, rest string) (*node, error) {
	text := rest
	for closingQuote(text) < 0 {
		if p.pos >= len(p.lines) {
			return nil, fault(owner, "the quote opened on this line never closes")
		}
		text += " " + strings.TrimSpace(p.lines[p.pos].text)
		p.pos++
	}
	end := closingQuote(text)
	if after := strings.TrimSpace(text[end+1:]); after != "" && !strings.HasPrefix(after, "#") {
		return nil, fault(owner, "text follows the closing quote")
	}
	return &node{kind: scalarNode, line: owner.num, text: text[1:end]}, nil
}

// blockScalar reads a `|` or `>` scalar: every following line indented
// deeper than its owner, blank lines included.
func (p *parser) blockScalar(owner *srcLine, rest string) (*node, error) {
	if !blockHeaderRe.MatchString(cutComment(rest)) {
		return nil, fault(owner, fmt.Sprintf("%q is not a block scalar header such as | or >-", rest))
	}
	var body []string
	for ; p.pos < len(p.lines) && inBlock(p.lines[p.pos], owner.indent); p.pos++ {
		body = append(body, p.lines[p.pos].text)
	}
	return &node{kind: scalarNode, line: owner.num, text: strings.Join(body, "\n")}, nil
}

// inBlock reports whether l belongs to a block scalar whose owner sits at
// indent: it is blank or indented deeper.
func inBlock(l srcLine, indent int) bool {
	return strings.TrimSpace(l.text) == "" || l.indent > indent
}

// flow reads a `[...]` or `{...}` value as its text, over as many lines as
// its brackets span. Text other than a comment after the closing bracket
// fails, as GitHub fails it.
func (p *parser) flow(owner *srcLine, rest string) (*node, error) {
	text, last := rest, owner
	end, err := flowEnd(owner, text)
	for ; end < 0 && err == nil; end, err = flowEnd(owner, text) {
		if p.pos >= len(p.lines) {
			return nil, fault(owner, fmt.Sprintf("the %c opened on this line never closes", rest[0]))
		}
		last = &p.lines[p.pos]
		text += " " + strings.Trim(last.text, blanks)
		p.pos++
	}
	if err == nil {
		err = flowTail(last, text[end+1:])
	}
	return &node{kind: flowNode, line: owner.num, text: text[:end+1]}, err
}

// flowTail fails text other than a comment after a flow value's closing
// bracket on l.
func flowTail(l *srcLine, tail string) error {
	after := strings.Trim(tail, blanks)
	if after == "" || strings.HasPrefix(after, "#") {
		return nil
	}
	return faultAt(l, colOf(l, after), "text follows the closing bracket of this flow value; remove it, or quote the whole value")
}

// flowEnd is the index of the bracket that closes the one text opens, outside quotes, or -1 while it stays open.
func flowEnd(owner *srcLine, text string) (int, error) {
	depth := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		quote = quoteState(quote, text[i])
		depth += bracketStep(quote, text[i])
		if depth < 0 {
			return 0, fault(owner, "a bracket closes here that never opened")
		}
		if depth == 0 {
			return i, nil
		}
	}
	return -1, nil
}

// quoteState is the quote open after reading c.
func quoteState(open, c byte) byte {
	if open == 0 && (c == '"' || c == '\'') {
		return c
	}
	if open == c {
		return 0
	}
	return open
}

// bracketStep is how c changes the bracket depth outside quotes.
func bracketStep(quote, c byte) int {
	if quote != 0 {
		return 0
	}
	return strings.Count("[{", string(c)) - strings.Count("]}", string(c))
}
