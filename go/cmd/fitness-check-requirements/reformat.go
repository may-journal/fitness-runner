package check

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// listMarker is a bullet or ordered-list marker at the start of a line,
// which a formatter may rewrite, as `*` to `-` or `1)` to `1.`.
var listMarker = regexp.MustCompile(`(?m)^(\s*)(?:[-*+]|\d+[.)])(\s)`)

// sourceLine is a `Source:` line older docs end their Measurement with.
// Nothing reads it, so deleting it changes no wording.
var sourceLine = regexp.MustCompile(`(?m)^Source:.*$`)

// reformats returns the changed docs whose wording did not change since
// base, so a formatter's rewrite, such as `biome format`, needs no test
// change. It is empty when there is no base.
func reformats(root, base string, changed []string) map[string]bool {
	found := map[string]bool{}
	if base == "" {
		return found
	}
	for _, f := range changed {
		if docNumber(f) != "" && reformatted(root, base, f) {
			found[f] = true
		}
	}
	return found
}

// reformatted reports whether doc f holds the same wording at base and in
// the scope being judged. A doc renamed since base is compared with the
// base doc of its number, since a number is never reused.
func reformatted(root, base, f string) bool {
	before, ok := gitx.Show(root, base, f)
	if !ok {
		before, ok = gitx.Show(root, base, renamedFrom(root, base, f))
	}
	if !ok {
		return false
	}
	after, ok := gitx.Judged(root, f)
	return ok && wording(before, true) == wording(after, false)
}

// renamedFrom is the one doc at base with f's number, the path a renamed f
// had there. With none, or more than one, it is f.
func renamedFrom(root, base, f string) string {
	out, err := exec.Command("git", "-C", root, "ls-tree", "--name-only", base, docsDir).Output()
	if err != nil {
		return f
	}
	var same []string
	for _, doc := range strings.Fields(string(out)) {
		if docNumber(doc) == docNumber(f) {
			same = append(same, doc)
		}
	}
	if len(same) == 1 {
		return same[0]
	}
	return f
}

// wording is a doc's words in order, without its front matter or HTML
// comments. At base only, a leftover Source line is dropped, so deleting one
// is no change but adding or editing one is. Spacing, line breaks, and list
// markers are made alike, as a formatter may change them. The doc check
// judges the structure.
func wording(doc string, dropSource bool) string {
	body := comments.WithoutHTMLComments(mdx.StripFrontMatter(doc))
	if dropSource {
		body = sourceLine.ReplaceAllString(body, "")
	}
	body = listMarker.ReplaceAllString(body, "$1-$2")
	return strings.Join(strings.Fields(body), " ")
}
