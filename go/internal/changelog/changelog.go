// Package changelog finds the changelog/ entry files a branch, commit, pull
// request or release adds, and reads their dated headings.
package changelog

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

var headingRe = regexp.MustCompile(`(?m)^### (\d{4}\.\d{2}\.\d{2}\.\d{4})`)

// Headings returns every dated heading's stamp in text, top to bottom.
func Headings(text string) []string {
	var stamps []string
	for _, m := range headingRe.FindAllStringSubmatch(text, -1) {
		stamps = append(stamps, m[1])
	}
	return stamps
}

// New returns the items in after that before lacks, in after's order.
func New(before, after []string) []string {
	had := map[string]bool{}
	for _, s := range before {
		had[s] = true
	}
	var added []string
	for _, s := range after {
		if !had[s] {
			added = append(added, s)
		}
	}
	return added
}

// forkPoint is the commit the current branch's own commits start from: the
// parent of the oldest commit no other origin branch holds. A branch stacked
// on another pushed branch so starts where that branch ends. It is not found
// on the default branch, on a detached HEAD, or without origin refs.
func forkPoint(root string) (string, bool) {
	branch := gitOut(root, "symbolic-ref", "--short", "-q", "HEAD")
	base := gitx.DefaultBase(root)
	if branch == "" || "origin/"+branch == base || gitOut(root, "rev-parse", "-q", "--verify", base) == "" {
		return "", false
	}
	own := gitx.OwnCommits(root, "HEAD", branch)
	if len(own) == 0 {
		return "", false
	}
	parent := gitOut(root, "rev-parse", "-q", "--verify", own[len(own)-1]+"^")
	return parent, parent != ""
}

// gitOut returns git's trimmed output in root, empty on failure.
func gitOut(root string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Diff is what a unified diff of an entry file adds and does not also remove.
type Diff struct {
	// Bullets are the text of new list items, without their markers.
	Bullets []string
}

// ReadDiff reads a unified diff, as git or GitHub prints it. A line both
// removed and added, such as a moved one, counts as unchanged.
func ReadDiff(patch string) Diff {
	var bullets diffSides
	for _, line := range strings.Split(patch, "\n") {
		sign, text := diffLine(line)
		bullets.add(sign, bulletText(text))
	}
	return Diff{Bullets: bullets.added()}
}

// bulletRe is a list item and its text.
var bulletRe = regexp.MustCompile(`^\s*[-*]\s+(\S.*)$`)

// entryLinkRe matches a markdown link to an entry file and captures its
// path from the entry folder on.
var entryLinkRe = regexp.MustCompile(`\]\((?:[^)\s]*?/)?(` + EntryDir + `/[^)\s#?/]+\.md)[^)\s]*\)`)

// EntryLinks returns the path of every entry file link in text, such as
// changelog/fast-checks.md.
func EntryLinks(text string) []string {
	var paths []string
	for _, m := range entryLinkRe.FindAllStringSubmatch(text, -1) {
		paths = append(paths, m[1])
	}
	return paths
}

// bulletText is a list item's text without its marker.
func bulletText(text string) []string {
	if m := bulletRe.FindStringSubmatch(text); m != nil {
		return []string{strings.TrimSpace(m[1])}
	}
	return nil
}

// diffSides collects what a diff's added and removed lines hold.
type diffSides struct{ plus, minus []string }

// add files items under the side sign names; other signs are context.
func (d *diffSides) add(sign byte, items []string) {
	switch sign {
	case '+':
		d.plus = append(d.plus, items...)
	case '-':
		d.minus = append(d.minus, items...)
	}
}

// added is what the diff adds and does not also remove, such as a moved line.
func (d diffSides) added() []string {
	return New(d.minus, d.plus)
}

// diffLine splits a diff line into its sign and text. File headers and
// empty lines carry no sign.
func diffLine(line string) (byte, string) {
	if line == "" || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
		return 0, ""
	}
	return line[0], line[1:]
}
