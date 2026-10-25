// Command fitness-check-mermaid-callouts is the Go port of the
// mermaid-callouts check: every mermaid diagram with callouts must have an
// associated callout table, and their callouts must be a 1-1 match (no
// orphan callouts, orphan rows, or duplicates). Nodes carry number callouts
// and connectors carry letter callouts (A–Z, then AA–ZZ, then AAA…), so a
// connector numbered like a node, or a letter that breaks the doubling,
// fails. Diagrams without callouts need no table; a table with no preceding
// diagram is an error.
package check

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-callouts", Blob: true},
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
	if len(errors) == 0 {
		return checkkit.Pass(filesChecked), nil
	}
	return checkkit.Fail(filesChecked, errors...), nil
}

// doublingRule names the letter sequence in errors.
const doublingRule = "use A to Z, then AA to ZZ, then AAA"

// counts tallies each callout label's occurrences, preserving first-seen
// order — the port of the TypeScript Map-based counts (Map iteration follows
// insertion order, so error ordering depends on it).
type counts struct {
	byLabel map[string]int
	order   []string
}

// countLabels builds the ordered tally for one label list.
func countLabels(labels []string) counts {
	c := counts{byLabel: map[string]int{}}
	for _, label := range labels {
		if _, seen := c.byLabel[label]; !seen {
			c.order = append(c.order, label)
		}
		c.byLabel[label]++
	}
	return c
}

// has reports whether label was tallied at all.
func (c counts) has(label string) bool {
	_, ok := c.byLabel[label]
	return ok
}

// numberLabels writes numbers as callout labels.
func numberLabels(numbers []int) []string {
	labels := make([]string, len(numbers))
	for i, n := range numbers {
		labels[i] = strconv.Itoa(n)
	}
	return labels
}

// diagramLabels lists every callout a diagram carries: node numbers, then
// connector callouts as written, numbers and broken letters included so
// each is matched once and reported only for its form.
func diagramLabels(d *mermaid.DiagramBlock, t *mermaid.CalloutTableBlock) []string {
	labels := append(numberLabels(d.Numbers), d.Letters...)
	labels = append(labels, numberLabels(d.EdgeNumbers)...)
	return append(labels, brokenLetters(d, t)...)
}

// brokenLetters returns the diagram's labels that break the doubling and
// that the table lists too, so they were meant as callouts. A whole label
// such as YES that no row lists is prose.
func brokenLetters(d *mermaid.DiagramBlock, t *mermaid.CalloutTableBlock) []string {
	var broken []string
	for _, label := range d.BadLetters {
		if t != nil && slices.Contains(t.BadLetters, label) {
			broken = append(broken, label)
		}
	}
	return broken
}

// tableLabels lists every callout row label in a table.
func tableLabels(t *mermaid.CalloutTableBlock) []string {
	labels := append(numberLabels(t.Numbers), t.Letters...)
	return append(labels, t.BadLetters...)
}

// duplicateErrors formats an error for each callout label appearing more
// than once.
func duplicateErrors(byLabel counts, format func(label string, count int) string) []string {
	var errors []string
	for _, label := range byLabel.order {
		if count := byLabel.byLabel[label]; count > 1 {
			errors = append(errors, format(label, count))
		}
	}
	return errors
}

// missingErrors formats an error for each label present in from but absent
// from other.
func missingErrors(from, other counts, format func(label string) string) []string {
	var errors []string
	for _, label := range from.order {
		if !other.has(label) {
			errors = append(errors, format(label))
		}
	}
	return errors
}

// formErrors reports connector callouts written in the wrong form: a number
// where a letter belongs, or letters that break the doubling, in the diagram
// or the table.
func formErrors(file string, diagram *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) []string {
	var errors []string
	for _, n := range diagram.EdgeNumbers {
		errors = append(errors, fmt.Sprintf(
			"%s: connector callout %d must be a letter; numbers are for nodes (line %d)", file, n, diagram.Line))
	}
	for _, label := range brokenLetters(diagram, table) {
		errors = append(errors, fmt.Sprintf(
			"%s: connector callout %s breaks the letter doubling; %s (line %d)", file, label, doublingRule, diagram.Line))
	}
	if table == nil {
		return errors
	}
	for _, label := range table.BadLetters {
		errors = append(errors, fmt.Sprintf(
			"%s: callout table row %s breaks the letter doubling; %s (line %d)", file, label, doublingRule, table.Line))
	}
	return errors
}

// compareCallouts reports the ways a diagram and its table fail to be a 1-1
// match on callouts.
func compareCallouts(file string, diagram *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) []string {
	inDiagram := countLabels(diagramLabels(diagram, table))
	inTable := countLabels(tableLabels(table))
	var errors []string
	errors = append(errors, duplicateErrors(inDiagram, func(label string, count int) string {
		return fmt.Sprintf("%s: diagram callout %s appears %d times (line %d)", file, label, count, diagram.Line)
	})...)
	errors = append(errors, duplicateErrors(inTable, func(label string, count int) string {
		return fmt.Sprintf("%s: callout table row %s appears %d times (line %d)", file, label, count, table.Line)
	})...)
	errors = append(errors, missingErrors(inDiagram, inTable, func(label string) string {
		return fmt.Sprintf("%s: diagram callout %s has no matching table row (line %d)", file, label, diagram.Line)
	})...)
	errors = append(errors, missingErrors(inTable, inDiagram, func(label string) string {
		return fmt.Sprintf("%s: callout table row %s has no matching diagram callout (line %d)", file, label, table.Line)
	})...)
	return errors
}

// validateDoc checks the 1-1 callout rule for one document: every diagram
// with callouts has an associated callout table with a bijective set of
// callouts, and every connector callout is a letter that follows the
// doubling.
func validateDoc(file, content string) []string {
	result := mermaid.PairDiagramsWithTables(mermaid.ParseDoc(content))
	var errors []string
	for _, table := range result.OrphanTables {
		errors = append(errors, fmt.Sprintf(
			"%s: callout table at line %d has no preceding mermaid diagram", file, table.Line))
	}
	for _, pair := range result.Pairs {
		errors = append(errors, formErrors(file, pair.Diagram, pair.Table)...)
		if pair.Table == nil {
			errors = append(errors, fmt.Sprintf(
				"%s: mermaid diagram at line %d has callouts but no associated callout table",
				file, pair.Diagram.Line))
			continue
		}
		errors = append(errors, compareCallouts(file, pair.Diagram, pair.Table)...)
	}
	return errors
}
