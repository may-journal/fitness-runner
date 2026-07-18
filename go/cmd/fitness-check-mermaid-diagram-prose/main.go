// Command fitness-check-mermaid-diagram-prose flags relationship/edge labels
// that carry prose beyond a callout number when a diagram is paired with a
// callout table — the Go port of the mermaid-diagram-prose check. Arrows
// carry numbers only; descriptions belong in the callout table.
//
// The shared parser lives in internal/mermaid; the JS \s class and the
// styling-line pattern it uses are unexported there, so that small residue
// (jsWS, isJSSpace, trimJS, styleLine) is duplicated here instead of
// modifying the frozen package.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-diagram-prose"},
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

// trimJS trims JavaScript whitespace from both ends — String.prototype.trim.
func trimJS(s string) string { return strings.TrimFunc(s, isJSSpace) }

// numberThenProse matches a callout number followed by prose, e.g. `6 Uses`
// — the duplication #20 targets.
var numberThenProse = regexp.MustCompile(`^\d+[` + jsWS + `]+[A-Za-z]`)

// relCall matches C4 relationship calls whose label argument may carry prose.
var relCall = regexp.MustCompile(`\b(?:Bi)?Rel\w*\([^)]*\)`)

// quoted matches quoted argument content.
var quoted = regexp.MustCompile(`["']([^"']*)["']`)

// pipeLabel matches a flowchart pipe edge label: `-->|6 uses|`.
var pipeLabel = regexp.MustCompile(`\|[` + jsWS + `]*([^|]*?)[` + jsWS + `]*\|`)

// styleLine matches classDef/style lines, which carry non-label numbers;
// skip them for colon labels.
var styleLine = regexp.MustCompile(`^[` + jsWS + `]*(classDef|style|linkStyle|%%)`)

// colonLabel matches an edge label after a colon: `A --> B : 6 uses`.
var colonLabel = regexp.MustCompile(`:[` + jsWS + `]*(\d+[` + jsWS + `]+[A-Za-z][^|]*?)[` + jsWS + `]*$`)

// relCallProse returns prose labels from C4 `Rel(...)` call arguments.
func relCallProse(body string) []string {
	var found []string
	for _, rel := range relCall.FindAllString(body, -1) {
		for _, arg := range quoted.FindAllStringSubmatch(rel, -1) {
			if label := trimJS(arg[1]); numberThenProse.MatchString(label) {
				found = append(found, label)
			}
		}
	}
	return found
}

// pipeProse returns prose labels from a line's flowchart pipe edge labels
// (`-->|…|`).
func pipeProse(line string) []string {
	var found []string
	for _, pipe := range pipeLabel.FindAllStringSubmatch(line, -1) {
		if label := trimJS(pipe[1]); numberThenProse.MatchString(label) {
			found = append(found, label)
		}
	}
	return found
}

// colonProse returns the prose label from a line's colon edge label
// (`A --> B : 6 uses`), if any.
func colonProse(line string) []string {
	colon := colonLabel.FindStringSubmatch(line)
	if colon == nil {
		return nil
	}
	if label := trimJS(colon[1]); numberThenProse.MatchString(label) {
		return []string{label}
	}
	return nil
}

// lineEdgeProse returns prose labels from a single diagram line's edge
// labels (styling lines skipped).
func lineEdgeProse(line string) []string {
	if styleLine.MatchString(line) {
		return nil
	}
	return append(pipeProse(line), colonProse(line)...)
}

// proseLabels returns relationship/edge labels in a diagram body that carry
// prose beyond a callout number.
func proseLabels(diagram *mermaid.DiagramBlock) []string {
	found := relCallProse(diagram.Body)
	for _, line := range strings.Split(diagram.Body, "\n") {
		found = append(found, lineEdgeProse(line)...)
	}
	return found
}

// validateDoc flags relationship/edge labels that carry prose beyond a
// callout number when a diagram is paired with a callout table —
// descriptions belong in the table.
func validateDoc(file, content string) []string {
	errors := []string{}
	for _, pair := range mermaid.PairDiagramsWithTables(mermaid.ParseDoc(content)).Pairs {
		if pair.Table == nil || len(pair.Diagram.Numbers) == 0 {
			continue
		}
		for _, label := range proseLabels(pair.Diagram) {
			errors = append(errors, fmt.Sprintf(
				`%s: diagram at line %d has a relationship label with prose ("%s") — put descriptions in the callout table`,
				file, pair.Diagram.Line, label))
		}
	}
	return errors
}
