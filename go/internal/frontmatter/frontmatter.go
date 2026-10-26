// Package frontmatter reads a Markdown file's leading front matter for the
// fitness keys, `fitnessFunctions` and `relatedConfigurations`. No check
// reads either key. `fitness strip-front-matter` deletes them with Strip,
// and the no-fitness-front-matter check fails a file that still holds one,
// found with HasFitnessKeys.
//
// A block is a leading `---` block, or the same block inside an HTML
// comment that opens on the first line, the form a PR template carries.
// A byte order mark may come first, and lines may end in CRLF. A leading
// `---` followed by prose is a horizontal rule, never front matter.
package frontmatter

import (
	"regexp"
	"slices"
	"strings"
)

// FitnessKeys are the front matter keys only fitness ever read. Both go
// from every block; any other key belongs to a tool that may still need it,
// such as GitHub's issue template fields, and stays.
var FitnessKeys = []string{"fitnessFunctions", "relatedConfigurations"}

var (
	// plainFront is a leading --- block and the blank lines after it. The
	// block is empty or opens on a non-blank line, so a rule followed by
	// a blank line never matches.
	plainFront = regexp.MustCompile(`\A---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n(?:\r?\n)*`)
	// commentFront is the same block inside an HTML comment that opens on
	// the first line, the form a PR template carries.
	commentFront = regexp.MustCompile(`\A<!--\r?\n---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n-->\r?\n(?:\r?\n)*`)
)

// byteOrderMark is the UTF-8 BOM some editors put before a file's first line.
const byteOrderMark = string(rune(0xFEFF))

// keyLine is a top-level front matter entry: a name, then a colon.
var keyLine = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\s*:`)

// block is a file's leading front matter: where the match ends, where the
// lines between its --- markers start and end, and its entries.
type block struct {
	end, start, stop int
	entries          []entry
}

// leading finds content's front matter in either form. It reports false
// when content opens with neither, or with a rule rather than front matter.
func leading(content string) (block, bool) {
	for _, re := range []*regexp.Regexp{plainFront, commentFront} {
		m := re.FindStringSubmatchIndex(content)
		if m == nil {
			continue
		}
		start, stop := innerSpan(m)
		entries, ok := frontEntries(content[start:stop])
		return block{end: m[1], start: start, stop: stop, entries: entries}, ok
	}
	return block{}, false
}

// HasFitnessKeys reports whether content's leading front matter holds
// either fitness key.
func HasFitnessKeys(content string) bool {
	b, ok := leading(strings.TrimPrefix(content, byteOrderMark))
	_, _, fitness := sortEntries(b.entries)
	return ok && fitness
}

// Strip returns content with the fitness keys gone from its front matter.
// A block left with no key, or empty to begin with, goes whole. When other
// keys stay, it returns the block without the fitness keys and names the
// keys that stay. Content without fitness keys, or whose leading --- opens
// a rule rather than front matter, comes back unchanged with no names. A
// BOM before the block stays at the top of the body.
func Strip(content string) (string, []string) {
	bom := ""
	if strings.HasPrefix(content, byteOrderMark) {
		bom, content = byteOrderMark, strings.TrimPrefix(content, byteOrderMark)
	}
	b, ok := leading(content)
	kept, others, fitness := sortEntries(b.entries)
	switch {
	case !ok, !fitness && len(others) > 0:
		return bom + content, nil
	case len(others) == 0:
		return bom + content[b.end:], nil
	}
	return bom + content[:b.start] + kept + content[b.stop:], others
}

// innerSpan is where the lines between a matched block's --- markers start
// and end; an empty block has none, so both sit at the block's start.
func innerSpan(m []int) (int, int) {
	if m[2] < 0 {
		return m[0], m[0]
	}
	return m[2], m[3]
}

// sortEntries joins the text of every entry that stays, names the other
// tools' keys among them, and reports whether a fitness key was there.
func sortEntries(entries []entry) (kept string, others []string, fitness bool) {
	for _, e := range entries {
		if slices.Contains(FitnessKeys, e.name) {
			fitness = true
			continue
		}
		kept += e.text
		if e.name != "" {
			others = append(others, e.name)
		}
	}
	return kept, others, fitness
}

// entry is one top-level front matter key and the text it spans, its own
// line through its last continuation line, line endings included.
type entry struct {
	name string
	text string
}

// frontEntries splits the lines between a block's --- markers into its
// top-level entries in order. Indented lines, list items, comments and
// blank lines belong to the entry above; those before the first key form an
// entry with no name. It reports false when a line is neither an entry nor
// a continuation, as with prose between two horizontal rules.
func frontEntries(inner string) ([]entry, bool) {
	var entries []entry
	for _, line := range strings.SplitAfter(inner, "\n") {
		var ok bool
		if entries, ok = addLine(entries, line); !ok {
			return nil, false
		}
	}
	return entries, true
}

// addLine adds one line to entries: a key opens a new entry, and any other
// line joins the entry above. It reports false for a line that can be
// neither.
func addLine(entries []entry, line string) ([]entry, bool) {
	bare := strings.TrimRight(line, "\r\n")
	switch {
	case keyLine.MatchString(bare):
		name, _, _ := strings.Cut(bare, ":")
		return append(entries, entry{name: strings.TrimSpace(name), text: line}), true
	case !continuesEntry(bare):
		return entries, false
	case len(entries) == 0:
		return append(entries, entry{text: line}), true
	}
	entries[len(entries)-1].text += line
	return entries, true
}

// continuesEntry reports whether a line belongs to the entry above it: it
// is blank, indented, a list item or a comment.
func continuesEntry(line string) bool {
	return line == "" || strings.ContainsAny(line[:1], " \t-#")
}
