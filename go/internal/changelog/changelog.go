// Package changelog finds the dated "### yyyy.mm.dd.HHMM" entries a
// branch, a staged commit, or a pull request adds to CHANGELOG.md. The commit
// gate, the pre-commit stamp, and pr-check share it, so all three agree that
// a pull request carries one entry, which later commits amend.
package changelog

import (
	"os/exec"
	"regexp"
	"strings"

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

// BranchEntries returns the dated entries the current branch's own commits
// add to CHANGELOG.md. It is nil on the default branch, on a detached HEAD,
// without origin refs, or when the branch's own commits add none.
func BranchEntries(root string) []string {
	fork, ok := forkPoint(root)
	if !ok {
		return nil
	}
	head, ok := show(root, "HEAD:"+File)
	if !ok {
		return nil
	}
	old, _ := show(root, fork+":"+File)
	return New(Headings(old), Headings(head))
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
	own := strings.Fields(gitOut(root, "rev-list", "--reverse", "HEAD", "--not", "--exclude=origin/"+branch, "--remotes=origin"))
	if len(own) == 0 {
		return "", false
	}
	parent := gitOut(root, "rev-parse", "-q", "--verify", own[0]+"^")
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

// StagedNew returns the dated entries the staged CHANGELOG.md adds over
// HEAD's, in file order.
func StagedNew(root string) []string {
	staged, ok := show(root, ":"+File)
	if !ok {
		return nil
	}
	head, _ := show(root, "HEAD:"+File)
	return New(Headings(head), Headings(staged))
}

// show returns a file as git holds it at spec, such as HEAD:CHANGELOG.md,
// and false when git cannot read it there.
func show(root, spec string) (string, bool) {
	out, err := exec.Command("git", "-C", root, "show", spec).Output()
	return string(out), err == nil
}

// DiffEntries reads a unified diff of CHANGELOG.md, as git or GitHub prints
// it. It returns the dated headings the diff adds and does not also remove,
// and whether it adds a new Release Please version section.
func DiffEntries(patch string) (added []string, release bool) {
	var entries, versions diffSides
	for _, line := range strings.Split(patch, "\n") {
		sign, text := diffLine(line)
		entries.add(sign, Headings(text))
		versions.add(sign, releaseVersions(text))
	}
	return entries.added(), len(versions.added()) > 0
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

// releaseVersions is the version a Release Please section heading names.
func releaseVersions(text string) []string {
	if m := releaseRe.FindStringSubmatch(text); m != nil {
		return []string{m[1]}
	}
	return nil
}

// diffLine splits a diff line into its sign and text. File headers and
// empty lines carry no sign.
func diffLine(line string) (byte, string) {
	if line == "" || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
		return 0, ""
	}
	return line[0], line[1:]
}
