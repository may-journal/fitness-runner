package check

import (
	"fmt"
	"regexp"
	"strings"
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

// syntaxError is a file YAML cannot read, and the line where reading
// stopped.
type syntaxError struct {
	line int
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
)

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
	return strings.TrimSpace(m[1]), strings.TrimSpace(text[len(m[0]):]), true
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
	return text[1:end], strings.TrimSpace(after[1:]), true
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
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasPrefix(rest, "#") {
		return p.nested(owner, keyed)
	}
	if rest[0] == '&' {
		return p.anchored(owner, rest, keyed)
	}
	if read, ok := scalarReaders[rest[0]]; ok {
		return read(p, owner, rest)
	}
	return p.plain(owner, rest), nil
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

// plain reads a plain scalar and the deeper lines that continue it.
func (p *parser) plain(owner *srcLine, rest string) *node {
	parts := []string{cutComment(rest)}
	for l := p.peek(); l != nil && l.indent > owner.indent; l = p.peek() {
		parts = append(parts, cutComment(l.text))
		p.pos++
	}
	return &node{kind: scalarNode, line: owner.num, text: strings.Join(parts, " ")}
}

// cutComment drops a ` #` comment from a plain scalar.
func cutComment(text string) string {
	if i := strings.Index(text, " #"); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text)
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
// its brackets span.
func (p *parser) flow(owner *srcLine, rest string) (*node, error) {
	text := rest
	depth, err := bracketDepth(owner, text)
	for ; depth > 0 && err == nil; depth, err = bracketDepth(owner, text) {
		if p.pos >= len(p.lines) {
			return nil, fault(owner, fmt.Sprintf("the %c opened on this line never closes", rest[0]))
		}
		text += " " + strings.TrimSpace(p.lines[p.pos].text)
		p.pos++
	}
	return &node{kind: flowNode, line: owner.num, text: text}, err
}

// bracketDepth counts the brackets text leaves open, outside quotes.
func bracketDepth(owner *srcLine, text string) (int, error) {
	depth := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		quote = quoteState(quote, text[i])
		depth += bracketStep(quote, text[i])
		if depth < 0 {
			return 0, fault(owner, "a bracket closes here that never opened")
		}
	}
	return depth, nil
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
