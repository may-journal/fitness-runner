package main

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// block is one prose unit: a paragraph (isList false, text set) or a list
// (isList true, items set), tagged with the section it falls under.
type block struct {
	section     int
	sectionName string
	isList      bool
	text        string
	items       []string
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

// parse splits markdown into paragraph and list blocks, dropping HTML
// comments and front matter and skipping fenced code, tables, and headings.
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
	b.item(t)
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

// breakLine handles the lines that end a block without contributing prose —
// headings (which also open a new section), blanks, and table rows — and
// reports whether the line was one.
func (b *builder) breakLine(t string) bool {
	switch {
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

// item adds one content line: a list marker closes any open paragraph and
// starts or extends the list; anything else closes any open list and extends
// the paragraph.
func (b *builder) item(t string) {
	if m := listItemRe.FindStringSubmatch(t); m != nil {
		b.flushPara()
		b.items = append(b.items, m[1])
		return
	}
	b.flushList()
	b.para = append(b.para, strings.TrimPrefix(t, "> "))
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
		section: b.section, sectionName: b.sectionName, isList: true, items: b.items,
	})
	b.items = nil
}
