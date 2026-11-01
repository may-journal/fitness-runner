// The shared parser lives in internal/mermaid, which keeps its JS space and fence helpers unexported, so this package duplicates them.

package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-diagram-table-gap", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsNotApplicable, Blob: true},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return validateDoc("(description)", content)
	}); err != nil {
		return checkkit.Result{}, err
	} else if handled {
		return res, nil
	}
	errors, filesChecked, err := mermaid.RunDocCheck(root, validateDoc)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) > 0 {
		return checkkit.Fail(filesChecked, errors...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

// jsWS is the JavaScript \s character class (ASCII whitespace incl. \v,
// Unicode spaces, line separators, and the BOM) as an RE2 class body —
// duplicated from internal/mermaid, where it is unexported.
const jsWS = `\t\n\v\f\r \x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`

// isJSSpace reports whether r is in the JavaScript \s class — the rune-level
// twin of jsWS.
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// trimStartJS trims leading JavaScript whitespace — String.prototype.trimStart,
// matching how internal/mermaid finds a fence's closing marker.
func trimStartJS(s string) string { return strings.TrimLeftFunc(s, isJSSpace) }

// fenceMarker captures a mermaid fence's opening backticks or tildes, so the closing fence matches as the parser does.
var fenceMarker = regexp.MustCompile("^[" + jsWS + "]*(`{3,}|~{3,})")

// captionLine matches the one caption the C4 docs place between diagram and table, opening "Numbers" and containing "match the callout table". Everything else in the gap is prose that belongs in the table.
var captionLine = regexp.MustCompile(`(?i)^numbers\b.*\bmatch the callout table\b`)

// validateDoc reports each non-blank, non-caption markdown line sitting
// between a numbered diagram (or its trailing legend block) and its callout
// table. Prose before the diagram and after the table is left alone.
func validateDoc(file, content string) []string {
	lines := strings.Split(content, "\n")
	blocks := mermaid.ParseDoc(content)
	paired := pairedTables(blocks)
	var errors []string
	for i := range blocks {
		prev, table := precedingDiagram(blocks, i, paired)
		if prev == nil || table == nil {
			continue
		}
		errors = append(errors, gapErrors(file, lines, prev, table)...)
	}
	return errors
}

// pairedTables is the set of callout tables paired with a numbered diagram, since an orphan table has no gap to police.
func pairedTables(blocks []mermaid.DocBlock) map[*mermaid.CalloutTableBlock]bool {
	paired := map[*mermaid.CalloutTableBlock]bool{}
	for _, pair := range mermaid.PairDiagramsWithTables(blocks).Pairs {
		if pair.Table != nil {
			paired[pair.Table] = true
		}
	}
	return paired
}

// precedingDiagram returns the callout table at blocks[i] and the mermaid block before it, the legend or else the diagram. Both are nil when blocks[i] is not a diagram-preceded table.
func precedingDiagram(blocks []mermaid.DocBlock, i int, paired map[*mermaid.CalloutTableBlock]bool) (*mermaid.DiagramBlock, *mermaid.CalloutTableBlock) {
	table, ok := blocks[i].(*mermaid.CalloutTableBlock)
	if !ok || i == 0 || !paired[table] {
		return nil, nil
	}
	prev, _ := blocks[i-1].(*mermaid.DiagramBlock)
	return prev, table
}

// gapErrors reports each non-blank, non-caption line between prev's closing
// fence and the table header — the prose that belongs in the table instead.
func gapErrors(file string, lines []string, prev *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) []string {
	var errors []string
	for ln := closingFenceLine(lines, prev.Line) + 1; ln < table.Line; ln++ {
		text := strings.TrimSpace(lines[ln-1])
		if text == "" || captionLine.MatchString(text) {
			continue
		}
		errors = append(errors, fmt.Sprintf(
			"%s: prose between the diagram and its callout table (line %d) — move detail into the callout table",
			file, ln))
	}
	return errors
}

// closingFenceLine returns the 1-based line of the fence closing the diagram opened at openLine, as internal/mermaid does. An unterminated block closes at end of file.
func closingFenceLine(lines []string, openLine int) int {
	open := openLine - 1
	marker := "```"
	if m := fenceMarker.FindStringSubmatch(lines[open]); m != nil {
		marker = m[1]
	}
	for i := open + 1; i < len(lines); i++ {
		if strings.HasPrefix(trimStartJS(lines[i]), marker) {
			return i + 1
		}
	}
	return len(lines)
}
