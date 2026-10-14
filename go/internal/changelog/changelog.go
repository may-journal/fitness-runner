// Package changelog finds the changelog/ entry files a branch, a staged
// commit, or a pull request adds, and reads CHANGELOG.md's dated
// "### yyyy.mm.dd.HHMM" history. The commit gate and pr-check share it, so
// both agree that a pull request carries one entry file, which later commits
// amend. It also names the anchor GitHub gives a release heading, so a
// release PR links its section.
package changelog

import (
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// File is the changelog every rule here reads.
const File = "CHANGELOG.md"

var (
	headingRe = regexp.MustCompile(`(?m)^### (\d{4}\.\d{2}\.\d{2}\.\d{4})`)
	// releaseRe is a Release Please version section, such as "## [1.6.0]".
	releaseRe = regexp.MustCompile(`^## \[(\d[^\]]*)\]`)
)

// Headings returns every dated heading's stamp in text, top to bottom.
func Headings(text string) []string {
	var stamps []string
	for _, m := range headingRe.FindAllStringSubmatch(text, -1) {
		stamps = append(stamps, m[1])
	}
	return stamps
}

// New returns the stamps in after that before lacks, in after's order.
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
// on another pushed branch so starts where that branch ends.
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

// Diff is what a unified diff of CHANGELOG.md adds and does not also remove.
type Diff struct {
	// Releases are new Release Please version heading lines.
	Releases []string
	// Bullets are the text of new list items, without their markers.
	Bullets []string
}

// ReadDiff reads a unified diff of CHANGELOG.md, as git or GitHub prints it.
// A line both removed and added, such as a moved one, counts as unchanged.
func ReadDiff(patch string) Diff {
	var releases, bullets diffSides
	for _, line := range strings.Split(patch, "\n") {
		sign, text := diffLine(line)
		releases.add(sign, matching(releaseRe, text))
		bullets.add(sign, bulletText(text))
	}
	return Diff{Releases: releases.added(), Bullets: bullets.added()}
}

// CreatesFile reports whether a unified diff, as git or GitHub prints it,
// creates its file: its first hunk starts from nothing.
func CreatesFile(patch string) bool {
	_, hunk, ok := strings.Cut("\n"+patch, "\n@@ ")
	return ok && strings.HasPrefix(hunk, "-0,0 ")
}

// Anchors are the fragments GitHub gives the release sections d adds.
func (d Diff) Anchors() []string {
	var anchors []string
	for _, h := range d.Releases {
		anchors = append(anchors, Anchor(h))
	}
	return anchors
}

var (
	// mdLinkRe is a markdown link; Anchor keeps only its text.
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	// bulletRe is a list item and its text.
	bulletRe = regexp.MustCompile(`^\s*[-*]\s+(\S.*)$`)
	// linkRe matches a markdown link to a CHANGELOG.md heading and
	// captures the heading's anchor. GitHub anchors are lowercase, so a
	// template placeholder such as #REPLACE-WITH-STAMP never matches.
	linkRe = regexp.MustCompile(`\]\([^)\s]*` + regexp.QuoteMeta(File) + `#([a-z0-9_-]+)\)`)
)

// Anchor is the fragment GitHub gives a CHANGELOG.md heading: its text with
// links reduced to their text, lowercased, punctuation dropped, and spaces
// made hyphens. The entry "### 2026.10.12.0919" anchors at 202610120919.
func Anchor(heading string) string {
	text := mdLinkRe.ReplaceAllString(strings.TrimLeft(heading, "# "), "$1")
	return strings.Map(anchorRune, strings.ToLower(strings.TrimSpace(text)))
}

// anchorRune is how r reads in an anchor: a space becomes a hyphen, a
// letter, digit, hyphen, or underscore stays, and anything else drops.
func anchorRune(r rune) rune {
	if r == ' ' {
		return '-'
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_", r) {
		return r
	}
	return -1
}

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

// Links returns the anchor of every CHANGELOG.md heading link in text.
func Links(text string) []string {
	var anchors []string
	for _, m := range linkRe.FindAllStringSubmatch(text, -1) {
		anchors = append(anchors, m[1])
	}
	return anchors
}

// matching is text when re matches it, else nothing.
func matching(re *regexp.Regexp, text string) []string {
	if re.MatchString(text) {
		return []string{text}
	}
	return nil
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
