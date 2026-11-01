package check

import (
	"fmt"
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-callout-why", Scope: checkkit.ScopeFiles, Blob: true},
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

// whyCell matches a header cell naming the Why column — the TS WHY regex.
var whyCell = regexp.MustCompile(`(?i)^why$`)

// cellAt returns row[i], or the empty string past the end — the TS
// nullish-coalescing fallback on a missing cell.
func cellAt(row []string, i int) string {
	if i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}

// isNumberedRowMissingWhy reports whether a row carries a callout label but
// leaves its Why cell blank.
// The label is a node number such as "1" or "[1]", or connector letters such as "A".
// The parser hands cells over already trimmed, so the TS `.trim()` reduces to
// an emptiness test.
func isNumberedRowMissingWhy(row []string, whyIndex int) bool {
	return mermaid.RowLabel(row) != "" && cellAt(row, whyIndex) == ""
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
