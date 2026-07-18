// Command fitness-check-gitignore-why validates that every ignore pattern in
// the repo's .gitignore is immediately preceded by a `#` comment explaining
// why it exists — the Go port of the gitignore-why check. A missing
// .gitignore passes with zero files checked, exactly like the TypeScript
// original.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// noComment is the violation message tail, verbatim from the TS enUS catalog.
const noComment = "has no explanatory # comment on the line above"

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "gitignore-why"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	raw, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if os.IsNotExist(err) {
		return checkkit.Pass(0), nil
	}
	if err != nil {
		return checkkit.Result{}, err
	}
	violations := findViolations(string(raw))
	if len(violations) > 0 {
		return checkkit.Fail(1, violations...), nil
	}
	return checkkit.Pass(1), nil
}

// lineKind classifies one raw .gitignore line: blank, a `#` comment, or an
// actual ignore pattern.
type lineKind int

const (
	kindBlank lineKind = iota
	kindComment
	kindPattern
)

// classifyLine sorts a raw line into blank, comment (starts with `#`), or
// pattern (any other non-blank line).
func classifyLine(line string) lineKind {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return kindBlank
	case strings.HasPrefix(trimmed, "#"):
		return kindComment
	default:
		return kindPattern
	}
}

// isExplanatoryComment reports whether the line is a non-empty explanatory
// comment — a `#` followed by real text, not a bare `#`.
func isExplanatoryComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#") &&
		strings.TrimSpace(strings.TrimLeft(trimmed, "#")) != ""
}

// findViolations returns a violation message for every pattern line not
// immediately preceded by an explanatory `#` comment, mirroring the TS
// message template exactly.
func findViolations(content string) []string {
	lines := strings.Split(content, "\n")
	var errors []string
	for i, line := range lines {
		if classifyLine(line) != kindPattern {
			continue
		}
		above := ""
		if i > 0 {
			above = lines[i-1]
		}
		if !isExplanatoryComment(above) {
			errors = append(errors, fmt.Sprintf(
				".gitignore:%d: pattern \"%s\" %s", i+1, strings.TrimSpace(line), noComment))
		}
	}
	return errors
}
