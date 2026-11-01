package check

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// block is one paragraph (text) or list (isList, items) within a section.
type block struct {
	section     int
	sectionName string
	isList      bool
	text        string
	items       []string
	// depths holds each item's nesting level, 0 for top-level items.
	depths []int
}

// words returns the block's masked prose word count.
func (b block) words() int {
	if !b.isList {
		return mdx.WordCount(mdx.MaskInline(b.text))
	}
	n := 0
	for _, it := range b.items {
		n += mdx.WordCount(mdx.MaskInline(it))
	}
	return n
}

// parse splits markdown into paragraph and list blocks, skipping comments,
// front matter, fenced code, tables, and headings.
func parse(content string) []block {
	content = htmlCommentRe.ReplaceAllString(content, " ")
	b := &builder{}
	for _, ln := range strings.Split(mdx.StripFrontMatter(content), "\n") {
		b.feed(ln)
	}
	b.flush()
	return b.blocks
}

// builder folds markdown lines into blocks, one section at a time.
type builder struct {
	section     int
	sectionName string
	blocks      []block
	para        []string
	items       []string
	depths      []int
	indents     []int
	inFence     bool
}

// feed consumes one raw markdown line.
func (b *builder) feed(raw string) {
	t := strings.TrimSpace(raw)
	if b.fenceToggle(t) || b.inFence {
		return
	}
	if b.breakLine(t) {
		return
	}
	b.item(raw, t)
}

// fenceToggle flips fence state and closes the open block on a ``` line,
// reporting whether the line was a fence marker.
func (b *builder) fenceToggle(t string) bool {
	if strings.HasPrefix(t, "```") {
		b.inFence = !b.inFence
		b.flush()
		return true
	}
	return false
}

// breakLine reports a line with no prose, ending a block at a heading, blank
// or table row and skipping a lone dash.
func (b *builder) breakLine(t string) bool {
	switch {
	case mdx.DashOnly(t):
	case strings.HasPrefix(t, "#"):
		b.flush()
		b.section++
		b.sectionName = strings.TrimSpace(strings.TrimLeft(t, "# "))
	case t == "", strings.HasPrefix(t, "|"):
		b.flush()
	default:
		return false
	}
	return true
}

// item adds a content line to the open list or paragraph, closing the other
// kind of block first.
func (b *builder) item(raw, t string) {
	if text, ok := mdx.ListItem(t); ok {
		b.flushPara()
		b.items = append(b.items, text)
		b.depths = append(b.depths, b.depth(len(raw)-len(strings.TrimLeft(raw, " \t"))))
		return
	}
	b.flushList()
	b.para = append(b.para, strings.TrimPrefix(t, "> "))
}

// depth returns an item's nesting level, opening a level on a deeper indent
// and closing back on a shallower one.
func (b *builder) depth(indent int) int {
	for len(b.indents) > 0 && b.indents[len(b.indents)-1] > indent {
		b.indents = b.indents[:len(b.indents)-1]
	}
	if len(b.indents) == 0 || b.indents[len(b.indents)-1] < indent {
		b.indents = append(b.indents, indent)
	}
	return len(b.indents) - 1
}

// flush closes both the open paragraph and the open list.
func (b *builder) flush() {
	b.flushPara()
	b.flushList()
}

// flushPara emits the buffered paragraph, if any.
func (b *builder) flushPara() {
	if len(b.para) == 0 {
		return
	}
	b.blocks = append(b.blocks, block{
		section: b.section, sectionName: b.sectionName, text: strings.Join(b.para, " "),
	})
	b.para = nil
}

// flushList emits the buffered list, if any.
func (b *builder) flushList() {
	if len(b.items) == 0 {
		return
	}
	b.blocks = append(b.blocks, block{
		section: b.section, sectionName: b.sectionName, isList: true, items: b.items, depths: b.depths,
	})
	b.items, b.depths, b.indents = nil, nil, nil
}
