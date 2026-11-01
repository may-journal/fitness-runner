package check

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// fixFlag makes the check convert each numbered connector callout to a
// letter before it judges, so `fitness mermaid-callouts --fix` converts a
// repo's docs.
const fixFlag = "--fix"

// fixHint ends the numbered connector error with the command that fixes it.
const fixHint = "run `fitness mermaid-callouts " + fixFlag + "`"

// edit replaces size bytes at offset with text.
type edit struct {
	offset int
	size   int
	text   string
}

// conversion is one connector callout the fix turned from a number into a
// letter.
type conversion struct {
	from string
	to   string
}

// fixDocs converts every doc the check reads when args ask for the fix. It
// returns a note for each diagram it left unchanged and each file it
// changed, then the count of changed files.
func fixDocs(root string, args []string) ([]string, error) {
	if !slices.Contains(args, fixFlag) {
		return nil, nil
	}
	var notes []string
	changed := 0
	for _, file := range walkfs.InScope(walkfs.FilesByExt(root, ".md")) {
		fileNotes, wrote, err := fixFile(root, file)
		if err != nil {
			return nil, err
		}
		notes = append(notes, fileNotes...)
		changed += wrote
	}
	return append(notes, fmt.Sprintf("%s changed %d %s", fixFlag, changed, plural(changed, "file"))), nil
}

// fixFile converts one doc in place. It returns its notes and 1 when it
// wrote the file, 0 when nothing changed.
func fixFile(root, file string) ([]string, int, error) {
	path := filepath.Join(root, filepath.FromSlash(file))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	fixed, conversions, notes := convertDoc(file, string(raw))
	if len(conversions) == 0 {
		return notes, 0, nil
	}
	notes = append(notes, fmt.Sprintf("%s: converted connector callouts %s", file, describe(conversions)))
	return notes, 1, os.WriteFile(path, []byte(fixed), 0o644)
}

// convertDoc turns each numbered connector callout in content, and its
// table row, into the next free letter of its diagram. A diagram it cannot
// convert keeps every byte and gets a note naming why.
func convertDoc(file, content string) (string, []conversion, []string) {
	lines := strings.Split(content, "\n")
	starts := lineStarts(lines)
	var edits []edit
	var conversions []conversion
	var notes []string
	for _, pair := range mermaid.PairDiagramsWithTables(mermaid.ParseDoc(content)).Pairs {
		pairEdits, pairConversions, skipped := convertPair(lines, starts, pair)
		edits = append(edits, pairEdits...)
		conversions = append(conversions, pairConversions...)
		if skipped != "" {
			notes = append(notes, fmt.Sprintf("%s: %s left the mermaid diagram at line %d unchanged: %s",
				file, fixFlag, pair.Diagram.Line, skipped))
		}
	}
	return applyEdits(content, edits), conversions, notes
}

// convertPair plans the edits that turn one diagram's numbered connectors
// and their rows into letters, or returns why it cannot.
func convertPair(lines []string, starts []int, pair mermaid.DiagramTablePair) ([]edit, []conversion, string) {
	numbered := edgeNumbers(pair.Diagram)
	if len(numbered) == 0 {
		return nil, nil, ""
	}
	if skipped := skipReason(pair, numbered); skipped != "" {
		return nil, nil, skipped
	}
	next := freeLetters(pair.Diagram, pair.Table)
	body := starts[pair.Diagram.Line]
	var edits []edit
	var conversions []conversion
	for _, c := range numbered {
		letter := next()
		edits = append(edits, edit{body + c.Offset, len(c.Label), letter}, rowEdit(lines, starts, pair.Table, c.Label, letter))
		conversions = append(conversions, conversion{c.Label, letter})
	}
	return edits, conversions, ""
}

// edgeNumbers lists a diagram's connector callouts written as numbers, as
// the check's own parser reads them, in document order.
func edgeNumbers(d *mermaid.DiagramBlock) []mermaid.Callout {
	var numbered []mermaid.Callout
	for _, c := range mermaid.ExtractCallouts(d.Body) {
		if _, err := strconv.Atoi(c.Label); err == nil && c.Edge {
			numbered = append(numbered, c)
		}
	}
	return numbered
}

// skipReason says why a diagram's numbered connectors have no one letter
// and one row each, or returns "" when every one does.
func skipReason(pair mermaid.DiagramTablePair, numbered []mermaid.Callout) string {
	if pair.Table == nil {
		return "it has no callout table"
	}
	for _, c := range numbered {
		if skipped := calloutSkip(pair.Diagram, pair.Table, c.Label); skipped != "" {
			return skipped
		}
	}
	return ""
}

// calloutSkip says why one numbered connector callout cannot become a
// letter.
// Its number also marks a node, marks another connector, or has no single row.
func calloutSkip(d *mermaid.DiagramBlock, t *mermaid.CalloutTableBlock, label string) string {
	n, _ := strconv.Atoi(label)
	switch {
	case slices.Contains(d.Numbers, n):
		return fmt.Sprintf("callout %s marks a node too", label)
	case countOf(d.EdgeNumbers, n) > 1:
		return fmt.Sprintf("callout %s marks %d connectors", label, countOf(d.EdgeNumbers, n))
	case countOf(t.Numbers, n) != 1:
		return fmt.Sprintf("callout %s has %d table rows", label, countOf(t.Numbers, n))
	}
	return ""
}

// countOf counts n in numbers.
func countOf(numbers []int, n int) int {
	count := 0
	for _, each := range numbers {
		if each == n {
			count++
		}
	}
	return count
}

// freeLetters returns a source of the letters in doubling order that the
// diagram and its table do not use yet.
func freeLetters(d *mermaid.DiagramBlock, t *mermaid.CalloutTableBlock) func() string {
	used := map[string]bool{}
	for _, label := range slices.Concat(d.Letters, d.BadLetters, t.Letters, t.BadLetters) {
		used[label] = true
	}
	i := 0
	return func() string {
		for used[letterAt(i)] {
			i++
		}
		used[letterAt(i)] = true
		return letterAt(i)
	}
}

// letterAt returns the i-th connector letter: A to Z, then AA to ZZ, then
// AAA.
func letterAt(i int) string {
	return strings.Repeat(string(rune('A'+i%26)), i/26+1)
}

// rowEdit plans the edit that turns the table row for label into letter,
// padding a shorter letter so an aligned table stays aligned.
func rowEdit(lines []string, starts []int, t *mermaid.CalloutTableBlock, label, letter string) edit {
	index := t.Line + 1 + rowIndex(t, label)
	line := lines[index]
	at := mermaid.RowLabelOffset(line)
	cell := mermaid.RowLabel(t.Rows[index-t.Line-1])
	end := at + len(cell)
	text := letter
	if strings.HasPrefix(line[end:], "]") {
		end++
		text += "]"
	}
	return edit{starts[index] + at, end - at, text + strings.Repeat(" ", max(0, len(cell)-len(letter)))}
}

// rowIndex returns the index of the one row whose number is label's.
func rowIndex(t *mermaid.CalloutTableBlock, label string) int {
	return slices.IndexFunc(t.Rows, func(row []string) bool {
		return sameNumber(mermaid.RowLabel(row), label)
	})
}

// sameNumber reports two labels that read as the same number, as 2 and 02.
func sameNumber(a, b string) bool {
	x, errX := strconv.Atoi(a)
	y, errY := strconv.Atoi(b)
	return errX == nil && errY == nil && x == y
}

// lineStarts returns the byte offset where each line starts.
func lineStarts(lines []string) []int {
	starts := make([]int, len(lines))
	offset := 0
	for i, line := range lines {
		starts[i] = offset
		offset += len(line) + 1
	}
	return starts
}

// applyEdits applies edits to content, which never overlap.
func applyEdits(content string, edits []edit) string {
	sort.Slice(edits, func(i, j int) bool { return edits[i].offset > edits[j].offset })
	for _, e := range edits {
		content = content[:e.offset] + e.text + content[e.offset+e.size:]
	}
	return content
}

// describe lists conversions as `2 to A, 6 to B`.
func describe(conversions []conversion) string {
	parts := make([]string, len(conversions))
	for i, c := range conversions {
		parts[i] = c.from + " to " + c.to
	}
	return strings.Join(parts, ", ")
}

// plural adds an s to word unless n is one.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
