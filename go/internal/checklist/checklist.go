// Package checklist reads the task list in an issue body, so `fitness
// close-check` and `fitness pr-check` agree on which items are still open.
package checklist

import (
	"regexp"
	"strings"
)

// uncheckedRe matches a GitHub task-list item left unchecked, `- [ ] text`,
// with any list marker and indent.
var uncheckedRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\[ \]\s+(.*\S)`)

// fenceRe matches a code fence opener or closer and captures its marker.
var fenceRe = regexp.MustCompile("^\\s*(`{3,}|~{3,})")

// Unchecked returns the text of each unchecked task-list item in body, in
// order. Items inside fenced code blocks are examples, not tasks, so they are
// skipped.
func Unchecked(body string) []string {
	var items []string
	fence := ""
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			fence = toggleFence(fence, m[1])
			continue
		}
		if fence != "" {
			continue
		}
		if m := uncheckedRe.FindStringSubmatch(line); m != nil {
			items = append(items, m[1])
		}
	}
	return items
}

// toggleFence opens a fence, or closes the open one when marker is the same
// character and at least as long, per CommonMark.
func toggleFence(open, marker string) string {
	if open == "" {
		return marker
	}
	if marker[0] == open[0] && len(marker) >= len(open) {
		return ""
	}
	return open
}
