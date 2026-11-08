// Command workflow-timeouts fails a workflow timeout of 10 minutes or more
// whose comment cites no observed maximum and date.
package main

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// longTimeout is the fewest minutes whose limit must cite its observed maximum.
const longTimeout = 10

var (
	// timeoutRe matches a `timeout-minutes:` key and its whole-number value.
	timeoutRe = regexp.MustCompile(`^\s*timeout-minutes:\s*(\d+)\s*(#.*)?$`)
	// citeRe matches a cited observed maximum and the date it was seen.
	citeRe = regexp.MustCompile(`Observed max (\d+m)?\d+s on \d{4}-\d{2}-\d{2}`)
)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	files := workflows(ctx.InScope(ctx.Files(".yml", ".yaml")))
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		found = append(found, judge(f, strings.Split(text, "\n"))...)
	}
	return check.Found(len(files), found...), nil
}

// workflows keeps the files directly under .github/workflows.
func workflows(files []string) []string {
	var out []string
	for _, f := range files {
		if path.Dir(f) == ".github/workflows" {
			out = append(out, f)
		}
	}
	return out
}

// judge fails each long timeout in one workflow's lines that cites no observed maximum.
func judge(file string, lines []string) []check.Finding {
	var found []check.Finding
	for i, line := range lines {
		minutes, ok := timeout(line)
		if ok && minutes >= longTimeout && !cited(lines, i) {
			found = append(found, check.Finding{File: file, Line: i + 1, Rule: "timeout-observed-max",
				Message: fmt.Sprintf("timeout-minutes %d cites no observed maximum and date", minutes),
				Fix:     "add a comment above it such as `# Observed max 4m12s on 2026-11-08.`, or measure the job and set twice its maximum"})
		}
	}
	return found
}

// timeout is the minutes a `timeout-minutes:` line sets, and whether it sets a whole number.
func timeout(line string) (int, bool) {
	m := timeoutRe.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	minutes, err := strconv.Atoi(m[1])
	return minutes, err == nil
}

// cited reports whether line i, or the comment lines just above it, cite an observed maximum.
func cited(lines []string, i int) bool {
	if citeRe.MatchString(lines[i]) {
		return true
	}
	for j := i - 1; j >= 0 && strings.HasPrefix(strings.TrimSpace(lines[j]), "#"); j-- {
		if citeRe.MatchString(lines[j]) {
			return true
		}
	}
	return false
}
