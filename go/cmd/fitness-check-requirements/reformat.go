package check

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// listMarker is a bullet or ordered-list marker at the start of a line,
// which a formatter may rewrite, as `*` to `-` or `1)` to `1.`.
var listMarker = regexp.MustCompile(`(?m)^(\s*)(?:[-*+]|\d+[.)])(\s)`)

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
// the scope being judged: HEAD for a pull request, else the index, which
// holds what a commit records.
func reformatted(root, base, f string) bool {
	now := ":" + f
	if os.Getenv("GITHUB_BASE_REF") != "" {
		now = "HEAD:" + f
	}
	before, ok := gitShow(root, base+":"+f)
	if !ok {
		return false
	}
	after, ok := gitShow(root, now)
	return ok && wording(before) == wording(after)
}

// gitShow returns the content of a git object such as rev:path.
func gitShow(root, object string) (string, bool) {
	out, err := exec.Command("git", "-C", root, "show", object).Output()
	return string(out), err == nil
}

// wording is a doc's words in order, without its front matter, and with
// list markers made alike: what is left when a formatter's choices of
// blank lines, indents, line breaks, spacing, quotes in the front matter,
// and list markers are set aside. The doc check judges the structure.
func wording(doc string) string {
	body := listMarker.ReplaceAllString(mdx.StripFrontMatter(doc), "$1-$2")
	return strings.Join(strings.Fields(body), " ")
}
