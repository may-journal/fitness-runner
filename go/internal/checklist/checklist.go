// Package checklist reads the task list in an issue body, so `fitness
// close-check` and `fitness pr-check` agree on which items are still open.
package checklist

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// uncheckedRe matches a GitHub task-list item left unchecked, `- [ ] text`,
// with any list marker and indent.
var uncheckedRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\[ \]\s+(.*\S)`)

// Unchecked returns the text of each unchecked task-list item in body, in
// order. Items inside fenced code blocks are examples, not tasks, so they are
// skipped.
func Unchecked(body string) []string {
	var items []string
	for _, line := range mdx.NonFencedLines(strings.ReplaceAll(body, "\r\n", "\n")) {
		if m := uncheckedRe.FindStringSubmatch(line.Text); m != nil {
			items = append(items, m[1])
		}
	}
	return items
}
