package check

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// listMarker is a line's leading list marker, which a formatter may rewrite,
// as `*` to `-` or `1)` to `1.`.
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

// reformatted reports whether doc f, or the base doc of its never-reused
// number, kept its wording in HEAD or the index.
func reformatted(root, base, f string) bool {
	now := ":" + f
	if os.Getenv("GITHUB_BASE_REF") != "" {
		now = "HEAD:" + f
	}
	before, ok := gitShow(root, base+":"+f)
	if !ok {
		before, ok = gitShow(root, base+":"+renamedFrom(root, base, f))
	}
	if !ok {
		return false
	}
	after, ok := gitShow(root, now)
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

// gitShow returns the content of a git object such as rev:path.
func gitShow(root, object string) (string, bool) {
	out, err := exec.Command("git", "-C", root, "show", object).Output()
	return string(out), err == nil
}

// wording is a doc's words in order without front matter, formatting, list
// marker style or, at base only, a Source line.
// The doc check judges the structure.
func wording(doc string, dropSource bool) string {
	body := mdx.StripFrontMatter(doc)
	if dropSource {
		body = sourceLine.ReplaceAllString(body, "")
	}
	body = listMarker.ReplaceAllString(body, "$1-$2")
	return strings.Join(strings.Fields(body), " ")
}
