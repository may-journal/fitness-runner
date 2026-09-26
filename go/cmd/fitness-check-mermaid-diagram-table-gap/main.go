// Command fitness-check-mermaid-diagram-table-gap flags loose markdown prose
// wedged between a numbered mermaid diagram (or its legend block) and the
// callout table that follows it. In the C4 doc convention the layout is
// diagram → legend → one caption line → callout table, and all detail lives
// in the table; a paragraph in that gap duplicates the table and should fail.
// The single caption line ("Numbers … match the callout table") is allowed,
// as is prose before the diagram or after the table — only the diagram-to-
// table gap is policed.
//
// The sibling check mermaid-diagram-prose owns a different rule: prose inside
// the diagram's own edge/relationship labels. This check never looks inside a
// block; it looks at the markdown lines around them.
//
// The shared parser lives in internal/mermaid; the JS \s class and the
// trim/fence helpers it uses are unexported there, so that small residue
// (jsWS, isJSSpace, trimStartJS, fenceMarker) is duplicated here instead of
// modifying the frozen package.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-diagram-table-gap"},
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

// fenceMarker captures a mermaid fence's opening run of backticks or tildes,
// so the closing fence can be matched exactly as the parser matches it.
var fenceMarker = regexp.MustCompile("^[" + jsWS + "]*(`{3,}|~{3,})")

// captionLine matches the one caption sentence the canonical C4 docs place
// between the diagram/legend and the table — it always opens with "Numbers"
// and states the numbers "match the callout table", with any wording in
// between and any trailing clause after (as 04-code.md and 05-user-journey.md
// use). Everything else in the gap is prose that belongs in the table.
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

// pairedTables is the set of callout tables that pair with a numbered
// diagram; an orphan table (no preceding numbered diagram) has no gap to
// police.
func pairedTables(blocks []mermaid.DocBlock) map[*mermaid.CalloutTableBlock]bool {
	paired := map[*mermaid.CalloutTableBlock]bool{}
	for _, pair := range mermaid.PairDiagramsWithTables(blocks).Pairs {
		if pair.Table != nil {
			paired[pair.Table] = true
		}
	}
	return paired
}

// precedingDiagram returns the callout table at blocks[i] and the mermaid
// block immediately before it — the legend when one sits between diagram and
// table, else the diagram itself, whose closing fence bounds the gap. Either
// return is nil when blocks[i] is not an in-scope, diagram-preceded table.
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

// closingFenceLine returns the 1-based line of the mermaid fence that closes
// the diagram opened at openLine (1-based) — the first later line whose
// trimmed start repeats the opening marker, matching internal/mermaid's
// parseDiagramBlock. An unterminated block closes at end of file.
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
