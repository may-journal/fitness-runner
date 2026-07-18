// Command fitness-check-mermaid-callout-why requires each numbered callout
// table to carry a "Why" column and flags numbered rows whose "Why" cell is
// empty — so tables explain why each element exists, not just what it is.
// The Go port of the mermaid-callout-why check: same rule, same error
// templates, filesChecked counts the .md files with at least one mermaid
// diagram or callout table.
package main

import (
	"fmt"
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-callout-why"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errors, filesChecked, err := mermaid.RunDocCheck(root, validateDoc)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) > 0 {
		return checkkit.Fail(filesChecked, errors...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

var (
	// whyCell matches a header cell naming the Why column — the TS WHY regex.
	whyCell = regexp.MustCompile(`(?i)^why$`)
	// numberedCell matches a numbered first-column cell, e.g. "1" or "[1]" —
	// the TS NUMBERED regex.
	numberedCell = regexp.MustCompile(`^\[?\d+\]?$`)
)

// cellAt returns row[i], or the empty string past the end — the TS
// nullish-coalescing fallback on a missing cell.
func cellAt(row []string, i int) string {
	if i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}

// isNumberedRowMissingWhy reports whether a row is numbered but leaves its
// Why cell blank. The parser hands cells over already trimmed, so the TS
// `.trim()` reduces to an emptiness test.
func isNumberedRowMissingWhy(row []string, whyIndex int) bool {
	return numberedCell.MatchString(cellAt(row, 0)) && cellAt(row, whyIndex) == ""
}

// whyErrorsForTable returns the errors for one table: a missing Why column,
// or numbered rows with an empty Why cell.
func whyErrorsForTable(file string, table *mermaid.CalloutTableBlock) []string {
	whyIndex := whyColumnIndex(table.Header)
	if whyIndex == -1 {
		return []string{fmt.Sprintf(
			`%s: callout table at line %d is missing a "Why" column`, file, table.Line)}
	}
	var errors []string
	for _, row := range table.Rows {
		if isNumberedRowMissingWhy(row, whyIndex) {
			errors = append(errors, fmt.Sprintf(
				`%s: callout table row %s has an empty "Why" cell (line %d)`,
				file, row[0], table.Line))
		}
	}
	return errors
}

// whyColumnIndex returns the index of the first header cell naming the Why
// column, or -1 when the table has none.
func whyColumnIndex(header []string) int {
	for i, cell := range header {
		if whyCell.MatchString(cell) {
			return i
		}
	}
	return -1
}

// validateDoc collects the Why-column errors for every callout table in one
// markdown document.
func validateDoc(file, content string) []string {
	var errors []string
	for _, block := range mermaid.ParseDoc(content) {
		if table, ok := block.(*mermaid.CalloutTableBlock); ok {
			errors = append(errors, whyErrorsForTable(file, table)...)
		}
	}
	return errors
}
