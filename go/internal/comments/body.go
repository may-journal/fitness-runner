package comments

import (
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// measureBodies sets each placed block's body totals: its comment blocks and their words, and its code lines and tokens.
func measureBodies(blocks []Block, src []byte) {
	totals := commentTotals(blocks)
	if len(totals) == 0 {
		return
	}
	rows := lines(CodeView(blocks[0].Path, src))
	for _, t := range totals {
		t.Lines, t.Tokens = code(rows[t.From:min(t.To-1, len(rows))])
	}
	for i := range blocks {
		if t, ok := totals[blocks[i].Body.From]; ok {
			blocks[i].Body = *t
		}
	}
}

// commentTotals counts the comment blocks and words of each body among blocks, by its opening line.
func commentTotals(blocks []Block) map[int]*Body {
	totals := map[int]*Body{}
	for _, b := range blocks {
		if b.Body.From == 0 {
			continue
		}
		if totals[b.Body.From] == nil {
			totals[b.Body.From] = &Body{From: b.Body.From, To: b.Body.To}
		}
		totals[b.Body.From].Blocks++
		totals[b.Body.From].Words += proseWords(b.Text)
	}
	return totals
}

// code counts the lines of rows holding code and their tokens.
func code(rows []string) (lines, count int) {
	for _, row := range rows {
		if n := tokens(row); n > 0 {
			lines, count = lines+1, count+n
		}
	}
	return lines, count
}

// proseWords counts the words of a comment outside the code its Markdown fences, so a drawn diagram or example adds none.
func proseWords(text string) int {
	n, fenced := 0, false
	for _, line := range strings.Split(Markdown(text), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
		} else if !fenced {
			n += mdx.WordCount(line)
		}
	}
	return n
}

// tokens counts the code tokens of a row: each run of letters, digits and underscores, and each other visible character.
func tokens(row string) int {
	n := len(strings.FieldsFunc(row, func(r rune) bool { return !wordRune(r) }))
	for _, r := range row {
		if !wordRune(r) && !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// wordRune reports a letter, digit or underscore, which a name or number is made of.
func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// markLeads marks the first block of each function body among blocks, which the run judges.
func markLeads(blocks []Block) {
	type key struct {
		path string
		from int
	}
	seen := map[key]bool{}
	for i := range blocks {
		k := key{blocks[i].Path, blocks[i].Body.From}
		if k.from > 0 && !seen[k] {
			seen[k], blocks[i].Body.Lead = true, true
		}
	}
}
