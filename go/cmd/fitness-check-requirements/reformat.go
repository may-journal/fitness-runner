package check

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// withoutReformats drops each changed doc whose words did not change since
// base, so a formatter's rewrite, such as `biome format`, pairs with nothing.
// Every changed file is kept when there is no base.
func withoutReformats(root, base string, changed []string) []string {
	if base == "" {
		return changed
	}
	var kept []string
	for _, f := range changed {
		if docNumber(f) != "" && reformatted(root, base, f) {
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

// reformatted reports whether doc f holds the same words, line by line, at
// base and in the index. The index holds what a commit records, and in a
// pull request run it matches HEAD.
func reformatted(root, base, f string) bool {
	before := gitLine(root, "show", base+":"+f)
	after := gitLine(root, "show", ":"+f)
	return before != "" && wording(before) == wording(after)
}

// wording is a doc's text with front matter, blank lines, indentation and runs
// of spaces set aside: what a formatter may change without changing the doc.
func wording(doc string) string {
	var lines []string
	for _, l := range strings.Split(mdx.StripFrontMatter(doc), "\n") {
		if f := strings.Fields(l); len(f) > 0 {
			lines = append(lines, strings.Join(f, " "))
		}
	}
	return strings.Join(lines, "\n")
}
