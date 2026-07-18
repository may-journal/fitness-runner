// Command fitness-check-mermaid-callouts is the Go port of the
// mermaid-callouts check: every numbered mermaid diagram must have an
// associated callout table, and their callout numbers must be a 1-1 match
// (no orphan numbers, orphan rows, or duplicates). Un-numbered diagrams need
// no table; a table with no preceding diagram is an error.
package main

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-callouts"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errors, filesChecked, err := mermaid.RunDocCheck(root, validateDoc)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) == 0 {
		return checkkit.Pass(filesChecked), nil
	}
	return checkkit.Fail(filesChecked, errors...), nil
}

// counts tallies each number's occurrences, preserving first-seen order —
// the port of the TypeScript Map-based counts (Map iteration follows
// insertion order, so error ordering depends on it).
type counts struct {
	byNumber map[int]int
	order    []int
}

// countNumbers builds the ordered tally for one number list.
func countNumbers(numbers []int) counts {
	c := counts{byNumber: map[int]int{}}
	for _, n := range numbers {
		if _, seen := c.byNumber[n]; !seen {
			c.order = append(c.order, n)
		}
		c.byNumber[n]++
	}
	return c
}

// has reports whether n was tallied at all.
func (c counts) has(n int) bool {
	_, ok := c.byNumber[n]
	return ok
}

// duplicateErrors formats an error for each callout number appearing more
// than once.
func duplicateErrors(byNumber counts, format func(n, count int) string) []string {
	var errors []string
	for _, n := range byNumber.order {
		if count := byNumber.byNumber[n]; count > 1 {
			errors = append(errors, format(n, count))
		}
	}
	return errors
}

// missingErrors formats an error for each number present in from but absent
// from other.
func missingErrors(from, other counts, format func(n int) string) []string {
	var errors []string
	for _, n := range from.order {
		if !other.has(n) {
			errors = append(errors, format(n))
		}
	}
	return errors
}

// compareCallouts reports the ways a diagram and its table fail to be a 1-1
// match on callout numbers.
func compareCallouts(file string, diagram *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) []string {
	inDiagram := countNumbers(diagram.Numbers)
	inTable := countNumbers(table.Numbers)
	var errors []string
	errors = append(errors, duplicateErrors(inDiagram, func(n, count int) string {
		return fmt.Sprintf("%s: diagram callout %d appears %d times (line %d)", file, n, count, diagram.Line)
	})...)
	errors = append(errors, duplicateErrors(inTable, func(n, count int) string {
		return fmt.Sprintf("%s: callout table row %d appears %d times (line %d)", file, n, count, table.Line)
	})...)
	errors = append(errors, missingErrors(inDiagram, inTable, func(n int) string {
		return fmt.Sprintf("%s: diagram callout %d has no matching table row (line %d)", file, n, diagram.Line)
	})...)
	errors = append(errors, missingErrors(inTable, inDiagram, func(n int) string {
		return fmt.Sprintf("%s: callout table row %d has no matching diagram callout (line %d)", file, n, table.Line)
	})...)
	return errors
}

// validateDoc checks the 1-1 callout rule for one document: every numbered
// diagram has an associated callout table with a bijective set of callout
// numbers.
func validateDoc(file, content string) []string {
	result := mermaid.PairDiagramsWithTables(mermaid.ParseDoc(content))
	var errors []string
	for _, table := range result.OrphanTables {
		errors = append(errors, fmt.Sprintf(
			"%s: callout table at line %d has no preceding mermaid diagram", file, table.Line))
	}
	for _, pair := range result.Pairs {
		if len(pair.Diagram.Numbers) == 0 {
			continue
		}
		if pair.Table == nil {
			errors = append(errors, fmt.Sprintf(
				"%s: mermaid diagram at line %d has numbered callouts but no associated callout table",
				file, pair.Diagram.Line))
			continue
		}
		errors = append(errors, compareCallouts(file, pair.Diagram, pair.Table)...)
	}
	return errors
}
