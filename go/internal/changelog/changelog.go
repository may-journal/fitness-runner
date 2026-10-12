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
	releaseRe = regexp.MustCompile(`^## \[`)
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

// BranchEntries returns the dated entries HEAD's CHANGELOG.md adds over its
// merge base with origin's default branch. It is nil when no base is known,
// such as a repo without that remote ref, or when HEAD adds none.
func BranchEntries(root string) []string {
	out, err := exec.Command("git", "-C", root, "merge-base", "HEAD", gitx.DefaultBase(root)).Output()
	base := strings.TrimSpace(string(out))
	if err != nil || base == "" {
		return nil
	}
	head, ok := show(root, "HEAD:"+File)
	if !ok {
		return nil
	}
	old, _ := show(root, base+":"+File)
	return New(Headings(old), Headings(head))
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
// and whether it adds a Release Please version section.
func DiffEntries(patch string) (added []string, release bool) {
	var plus, minus []string
	for _, line := range strings.Split(patch, "\n") {
		switch sign, text := diffLine(line); sign {
		case '+':
			plus = append(plus, Headings(text)...)
			release = release || releaseRe.MatchString(text)
		case '-':
			minus = append(minus, Headings(text)...)
		}
	}
	return New(minus, plus), release
}

// diffLine splits a diff line into its sign and text. File headers and
// empty lines carry no sign.
func diffLine(line string) (byte, string) {
	if line == "" || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
		return 0, ""
	}
	return line[0], line[1:]
}
